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

	// If username/password provided but no token, fetch JWT token
	if token == "" && username != "" && password != "" {
		tflog.Info(ctx, "Fetching JWT token using username/password")

		jwtToken, err := fetchJWTToken(ctx, httpClient, host, username, password)
		if err != nil {
			return nil, diag.Errorf("failed to obtain JWT token: %s", err)
		}
		token = jwtToken
		tflog.Info(ctx, "Successfully obtained JWT token")
	}

	client := &Client{
		Host:       host,
		Username:   username,
		Password:   password,
		Token:      token,
		HTTPClient: httpClient,
	}

	tflog.Info(ctx, "Configured Airflow provider", map[string]interface{}{
		"host":        host,
		"using_token": token != "",
	})

	return client, diags
}

// fetchJWTToken obtains a JWT token from the Airflow FAB auth manager
func fetchJWTToken(ctx context.Context, httpClient *http.Client, host, username, password string) (string, error) {
	tokenURL := fmt.Sprintf("%s/auth/token", strings.TrimRight(host, "/"))

	reqBody := tokenRequest{
		Username: username,
		Password: password,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}
