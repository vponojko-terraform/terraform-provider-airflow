package internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns a terraform provider for Airflow FAB Auth Manager
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AIRFLOW_HOST", nil),
				Description: "The base URL of the Airflow instance (e.g., https://airflow.example.com)",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AIRFLOW_USERNAME", nil),
				Description: "Username for basic authentication",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AIRFLOW_PASSWORD", nil),
				Description: "Password for basic authentication",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AIRFLOW_TOKEN", nil),
				Description: "JWT Bearer token for authentication (alternative to username/password)",
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skip TLS certificate verification",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     30,
				Description: "Request timeout in seconds",
			},
			"use_basic_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				DefaultFunc: schema.EnvDefaultFunc("AIRFLOW_USE_BASIC_AUTH", false),
				Description: "Use basic auth for all requests instead of JWT. Useful when the token endpoint is broken (FAB 3.2.0)",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"airflow_user": resourceUser(),
			"airflow_role": resourceRole(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"airflow_user":        dataSourceUser(),
			"airflow_role":        dataSourceRole(),
			"airflow_permissions": dataSourcePermissions(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

// tokenResponse represents the JWT token response from Airflow
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// tokenRequest represents the login request body
type tokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	host := d.Get("host").(string)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	token := d.Get("token").(string)
	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	timeout := d.Get("timeout").(int)
	useBasicAuth := d.Get("use_basic_auth").(bool)

	// Validate auth configuration
	if token == "" && (username == "" || password == "") {
		return nil, diag.Errorf("either 'token' or both 'username' and 'password' must be provided")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	httpClient := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}

	// If username/password provided but no token, fetch JWT token (unless basic auth forced)
	if !useBasicAuth && token == "" && username != "" && password != "" {
		tflog.Info(ctx, "Fetching JWT token using username/password")

		var jwtToken string
		var err error

		// FAB 3.2.0 has a bug where the token endpoint may crash on first call
		// due to Flask-AppBuilder initialization issues. Retry once.
		for attempt := 1; attempt <= 2; attempt++ {
			jwtToken, err = fetchJWTToken(ctx, httpClient, host, username, password)
			if err == nil {
				break
			}
			if attempt == 1 {
				tflog.Warn(ctx, "JWT token fetch failed, retrying", map[string]interface{}{
					"error":   err.Error(),
					"attempt": attempt,
				})
			}
		}

		if err != nil {
			tflog.Warn(ctx, "Failed to obtain JWT token, falling back to basic auth", map[string]interface{}{
				"error": err.Error(),
			})
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "JWT token fetch failed, using basic auth fallback",
				Detail:   fmt.Sprintf("Could not obtain JWT token: %s. All requests will use basic auth. Note: some FAB endpoints (e.g., roles) may require JWT and reject basic auth. If you see auth errors, provide a pre-obtained token via the 'token' attribute or AIRFLOW_TOKEN env var.", err),
			})
			useBasicAuth = true
		} else {
			token = jwtToken
			tflog.Info(ctx, "Successfully obtained JWT token")
		}
	}

	if useBasicAuth {
		tflog.Info(ctx, "Using basic auth for all requests")
	}

	client := &Client{
		Host:         host,
		Username:     username,
		Password:     password,
		Token:        token,
		UseBasicAuth: useBasicAuth,
		HTTPClient:   httpClient,
	}

	tflog.Info(ctx, "Configured Airflow provider", map[string]interface{}{
		"host":        host,
		"using_token": token != "",
	})

	return client, diags
}

// fetchJWTToken obtains a JWT token from the Airflow FAB auth manager
func fetchJWTToken(ctx context.Context, httpClient *http.Client, host, username, password string) (string, error) {
	// Try FAB-specific token endpoint first, then fall back to legacy endpoint
	endpoints := []string{
		fmt.Sprintf("%s/auth/token", strings.TrimRight(host, "/")),
		fmt.Sprintf("%s/auth/fab/v1/token", strings.TrimRight(host, "/")),
	}

	reqBody := tokenRequest{
		Username: username,
		Password: password,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	var lastErr error
	for _, tokenURL := range endpoints {
		// Try with JSON body only
		tflog.Debug(ctx, "Trying token endpoint with JSON body", map[string]interface{}{
			"url": tokenURL,
		})
		token, err := doTokenRequest(ctx, httpClient, tokenURL, jsonBody, "", "")
		if err == nil {
			tflog.Info(ctx, "Successfully obtained JWT token", map[string]interface{}{
				"endpoint": tokenURL,
			})
			return token, nil
		}
		lastErr = fmt.Errorf("%s (JSON body): %w", tokenURL, err)

		// Try with basic auth header as well
		tflog.Debug(ctx, "Trying token endpoint with basic auth header", map[string]interface{}{
			"url": tokenURL,
		})
		token, err = doTokenRequest(ctx, httpClient, tokenURL, jsonBody, username, password)
		if err == nil {
			tflog.Info(ctx, "Successfully obtained JWT token via basic auth", map[string]interface{}{
				"endpoint": tokenURL,
			})
			return token, nil
		}
		lastErr = fmt.Errorf("%s (basic auth): %w", tokenURL, err)
	}

	return "", fmt.Errorf("all token endpoints failed, last error: %w", lastErr)
}

func doTokenRequest(ctx context.Context, httpClient *http.Client, tokenURL string, jsonBody []byte, username, password string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}
