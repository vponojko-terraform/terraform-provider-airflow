package internal

import (
	"context"
	"crypto/tls"
	"net/http"
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

	client := &Client{
		Host:     host,
		Username: username,
		Password: password,
		Token:    token,
		HTTPClient: &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: transport,
		},
	}

	tflog.Info(ctx, "Configured Airflow provider", map[string]interface{}{
		"host":        host,
		"using_token": token != "",
	})

	return client, diags
}
