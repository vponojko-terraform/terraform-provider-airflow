package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceUserImport,
		},
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Unique username (1-64 chars, alphanumeric, underscores, hyphens)",
			},
			"first_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User's first name (1-64 chars)",
			},
			"last_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User's last name (1-64 chars)",
			},
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User's email address",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "User's password (write-only, cannot detect drift)",
			},
			"roles": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of role names assigned to the user",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the user is active",
			},
			"last_login": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of last login",
			},
			"login_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of successful logins",
			},
			"fail_login_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of failed login attempts",
			},
			"created_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when user was created",
			},
			"changed_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when user was last modified",
			},
			"use_basic_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Use basic auth instead of JWT for this resource (required for FAB < 3.2.0)",
			},
		},
	}
}

// resourceUserImport supports importing users with optional auth method specification.
// Import ID format: "username" or "username:basic"
// Using "username:basic" sets use_basic_auth=true in state so the subsequent read uses basic auth.
func resourceUserImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	importID := d.Id()

	// Check if import ID contains auth method hint (e.g., "myuser:basic")
	parts := strings.SplitN(importID, ":", 2)
	username := parts[0]

	if len(parts) == 2 {
		switch strings.ToLower(parts[1]) {
		case "basic":
			d.Set("use_basic_auth", true)
			tflog.Info(ctx, "Importing user with basic auth", map[string]interface{}{
				"username": username,
			})
		default:
			return nil, fmt.Errorf("unsupported import auth hint '%s', use 'basic' or omit (e.g., 'username' or 'username:basic')", parts[1])
		}
	}

	d.SetId(username)
	return []*schema.ResourceData{d}, nil
}

type createUserRequest struct {
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Roles     []RoleRef `json:"roles,omitempty"`
}

type updateUserRequest struct {
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Password  *string   `json:"password,omitempty"`
	Roles     []RoleRef `json:"roles,omitempty"`
	Active    *bool     `json:"active,omitempty"`
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	useBasicAuth := d.Get("use_basic_auth").(bool)

	username := d.Get("username").(string)

	tflog.Info(ctx, "Creating Airflow user", map[string]interface{}{
		"username": username,
	})

	req := createUserRequest{
		Username:  username,
		FirstName: d.Get("first_name").(string),
		LastName:  d.Get("last_name").(string),
		Email:     d.Get("email").(string),
		Password:  d.Get("password").(string),
	}

	if v, ok := d.GetOk("roles"); ok {
		rolesList := v.([]interface{})
		req.Roles = make([]RoleRef, len(rolesList))
		for i, r := range rolesList {
			req.Roles[i] = RoleRef{Name: r.(string)}
		}
	}

	authMethod := AuthJWT
	if useBasicAuth {
		authMethod = AuthBasic
	}

	_, statusCode, err := client.DoRequestWithAuth(ctx, "POST", "/auth/fab/v1/users", req, authMethod)
	if err != nil {
		if statusCode == 409 {
			return diag.Errorf("user '%s' or email already exists", username)
		}
		// FAB 3.2.0 has a bug where user creation succeeds but response serialization fails
		// with a 500 error due to timezone-naive datetime fields. Retry create to verify.
		if statusCode == 500 {
			tflog.Warn(ctx, "Received 500 error on user create, retrying to check if user was created", map[string]interface{}{
				"username": username,
			})
			// Retry create - if we get 409, user was created successfully
			_, retryStatus, _ := client.DoRequestWithAuth(ctx, "POST", "/auth/fab/v1/users", req, authMethod)
			if retryStatus == 409 {
				tflog.Info(ctx, "User was created successfully despite 500 response (FAB datetime serialization bug)", map[string]interface{}{
					"username": username,
				})
				d.SetId(username)
				goto continueAfterCreate
			}
		}
		return diag.FromErr(err)
	}

	d.SetId(username)
continueAfterCreate:

	// If active is explicitly set to false, we need to update after create
	// since the API doesn't accept 'active' on POST
	if !d.Get("active").(bool) {
		active := false
		updateReq := updateUserRequest{Active: &active}
		_, _, err := client.DoRequestWithAuth(ctx, "PATCH", fmt.Sprintf("/auth/fab/v1/users/%s", URLEncode(username)), updateReq, authMethod)
		if err != nil {
			return diag.FromErr(fmt.Errorf("user created but failed to set active=false: %w", err))
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics
	useBasicAuth := d.Get("use_basic_auth").(bool)

	username := d.Id()

	tflog.Debug(ctx, "Reading Airflow user", map[string]interface{}{
		"username": username,
	})

	authMethod := AuthJWT
	if useBasicAuth {
		authMethod = AuthBasic
	}

	resp, statusCode, err := client.DoRequestWithAuth(ctx, "GET", fmt.Sprintf("/auth/fab/v1/users/%s", URLEncode(username)), nil, authMethod)

	// If JWT auth fails with 401, automatically retry with basic auth.
	// This handles the transition from use_basic_auth=false to use_basic_auth=true
	// where the state still has the old value but the config has changed.
	if err != nil && statusCode == 401 && authMethod == AuthJWT && client.Username != "" && client.Password != "" {
		tflog.Warn(ctx, "JWT auth returned 401, retrying with basic auth", map[string]interface{}{
			"username": username,
		})
		resp, statusCode, err = client.DoRequestWithAuth(ctx, "GET", fmt.Sprintf("/auth/fab/v1/users/%s", URLEncode(username)), nil, AuthBasic)
	}

	if err != nil {
		if statusCode == 404 {
			tflog.Warn(ctx, "User not found, removing from state", map[string]interface{}{
				"username": username,
			})
			d.SetId("")
			return diags
		}
		// FAB 3.2.0 has datetime serialization bug - user exists but can't be read
		// Keep current state rather than failing
		if statusCode == 500 {
			tflog.Warn(ctx, "Received 500 on user read (FAB datetime bug), preserving current state", map[string]interface{}{
				"username": username,
			})
			return diags
		}
		return diag.FromErr(err)
	}

	var user UserResponse
	if err := json.Unmarshal(resp, &user); err != nil {
		return diag.FromErr(err)
	}

	d.Set("username", user.Username)
	d.Set("first_name", user.FirstName)
	d.Set("last_name", user.LastName)
	d.Set("email", user.Email)
	d.Set("active", user.Active)
	if user.LastLogin != nil {
		d.Set("last_login", *user.LastLogin)
	}
	d.Set("login_count", user.LoginCount)
	d.Set("fail_login_count", user.FailLoginCount)
	d.Set("created_on", user.CreatedOn)
	d.Set("changed_on", user.ChangedOn)

	roles := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roles[i] = r.Name
	}
	d.Set("roles", roles)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	useBasicAuth := d.Get("use_basic_auth").(bool)

	username := d.Id()

	tflog.Info(ctx, "Updating Airflow user", map[string]interface{}{
		"username": username,
	})

	req := updateUserRequest{}
	hasChanges := false

	if d.HasChange("first_name") {
		v := d.Get("first_name").(string)
		req.FirstName = &v
		hasChanges = true
	}

	if d.HasChange("last_name") {
		v := d.Get("last_name").(string)
		req.LastName = &v
		hasChanges = true
	}

	if d.HasChange("email") {
		v := d.Get("email").(string)
		req.Email = &v
		hasChanges = true
	}

	if d.HasChange("password") {
		v := d.Get("password").(string)
		req.Password = &v
		hasChanges = true
	}

	if d.HasChange("active") {
		v := d.Get("active").(bool)
		req.Active = &v
		hasChanges = true
	}

	if d.HasChange("roles") {
		rolesList := d.Get("roles").([]interface{})
		req.Roles = make([]RoleRef, len(rolesList))
		for i, r := range rolesList {
			req.Roles[i] = RoleRef{Name: r.(string)}
		}
		hasChanges = true
	}

	if hasChanges {
		authMethod := AuthJWT
		if useBasicAuth {
			authMethod = AuthBasic
		}

		_, statusCode, err := client.DoRequestWithAuth(ctx, "PATCH", fmt.Sprintf("/auth/fab/v1/users/%s", URLEncode(username)), req, authMethod)
		if err != nil {
			if statusCode == 409 {
				return diag.Errorf("email already in use by another user")
			}
			return diag.FromErr(err)
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics
	useBasicAuth := d.Get("use_basic_auth").(bool)

	username := d.Id()

	tflog.Info(ctx, "Deleting Airflow user", map[string]interface{}{
		"username": username,
	})

	authMethod := AuthJWT
	if useBasicAuth {
		authMethod = AuthBasic
	}

	_, statusCode, err := client.DoRequestWithAuth(ctx, "DELETE", fmt.Sprintf("/auth/fab/v1/users/%s", URLEncode(username)), nil, authMethod)
	if err != nil {
		if statusCode == 404 {
			// Already deleted
			d.SetId("")
			return diags
		}
		if statusCode == 403 {
			return diag.Errorf("cannot delete your own user account")
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
