package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// UserResponse represents a user from the Airflow API
type UserResponse struct {
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `json:"email"`
	Active         bool      `json:"active"`
	LastLogin      *string   `json:"last_login"`
	LoginCount     int       `json:"login_count"`
	FailLoginCount int       `json:"fail_login_count"`
	Roles          []RoleRef `json:"roles"`
	CreatedOn      string    `json:"created_on"`
	ChangedOn      string    `json:"changed_on"`
}

// RoleRef represents a role reference in user responses
type RoleRef struct {
	Name string `json:"name"`
}

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserRead,
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username to look up",
			},
			"id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The user's internal ID",
			},
			"first_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User's first name",
			},
			"last_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User's last name",
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User's email address",
			},
			"active": {
				Type:        schema.TypeBool,
				Computed:    true,
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
			"roles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of role names assigned to the user",
				Elem:        &schema.Schema{Type: schema.TypeString},
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
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics

	username := d.Get("username").(string)

	resp, statusCode, err := client.DoRequest(ctx, "GET", fmt.Sprintf("/auth/fab/v1/users/%s", username), nil)
	if err != nil {
		if statusCode == 404 {
			return diag.Errorf("user '%s' not found", username)
		}
		return diag.FromErr(err)
	}

	var user UserResponse
	if err := json.Unmarshal(resp, &user); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(username)
	d.Set("id", user.ID)
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
