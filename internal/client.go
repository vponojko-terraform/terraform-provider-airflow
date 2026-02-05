package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client holds the authenticated client information
type Client struct {
	Host       string
	Username   string
	Password   string
	Token      string
	HTTPClient *http.Client
}

// AuthMethod specifies which authentication to use
type AuthMethod int

const (
	AuthJWT AuthMethod = iota
	AuthBasic
)

// APIError represents an error response from the Airflow API
type APIError struct {
	Detail string `json:"detail"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Type   string `json:"type"`
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Title
}

// DoRequest executes a request with JWT authentication (default)
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	return c.DoRequestWithAuth(ctx, method, path, body, AuthJWT)
}

// DoRequestWithAuth executes a request with specified authentication method
func (c *Client) DoRequestWithAuth(ctx context.Context, method, path string, body interface{}, authMethod AuthMethod) ([]byte, int, error) {
	var req *http.Request
	var err error

	// Ensure path is properly encoded (but don't double-encode)
	fullURL := fmt.Sprintf("%s%s", strings.TrimRight(c.Host, "/"), path)

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			return nil, 0, err
		}
	}

	req.Header.Set("Accept", "application/json")

	// Set authorization based on method
	switch authMethod {
	case AuthBasic:
		if c.Username != "" && c.Password != "" {
			req.SetBasicAuth(c.Username, c.Password)
		}
	case AuthJWT:
		fallthrough
	default:
		if c.Token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		} else if c.Username != "" && c.Password != "" {
			// Fallback to basic auth if no token
			req.SetBasicAuth(c.Username, c.Password)
		}
	}

	tflog.Debug(ctx, "Making API request", map[string]interface{}{
		"method":      method,
		"url":         fullURL,
		"auth_method": authMethod,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		tflog.Error(ctx, "API request failed", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
			"method":      method,
			"url":         fullURL,
		})

		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Detail != "" {
			return respBody, resp.StatusCode, &apiErr
		}

		return respBody, resp.StatusCode, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// URLEncode safely encodes a path segment
func URLEncode(s string) string {
	return url.PathEscape(s)
}
