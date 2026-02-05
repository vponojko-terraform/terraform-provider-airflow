package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// RoleResponse represents a role from the Airflow API
type RoleResponse struct {
	Name    string       `json:"name"`
	Actions []ActionPerm `json:"actions"`
}

// ActionPerm represents a permission (action + resource)
type ActionPerm struct {
	Action   ActionRef   `json:"action"`
	Resource ResourceRef `json:"resource"`
}

// ActionRef represents an action reference
type ActionRef struct {
	Name string `json:"name"`
}

// ResourceRef represents a resource reference
type ResourceRef struct {
	Name string `json:"name"`
}

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRoleRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The role name to look up",
			},
			"permission": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "List of permissions assigned to the role",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The action name (e.g., can_read, can_edit)",
						},
						"resource": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource name (e.g., DAGs, Connections)",
						},
					},
				},
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	resp, statusCode, err := client.DoRequest(ctx, "GET", fmt.Sprintf("/auth/fab/v1/roles/%s", name), nil)
	if err != nil {
		if statusCode == 404 {
			return diag.Errorf("role '%s' not found", name)
		}
		return diag.FromErr(err)
	}

	var role RoleResponse
	if err := json.Unmarshal(resp, &role); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)

	permissions := make([]map[string]interface{}, len(role.Actions))
	for i, action := range role.Actions {
		permissions[i] = map[string]interface{}{
			"action":   action.Action.Name,
			"resource": action.Resource.Name,
		}
	}
	d.Set("permission", permissions)

	return diags
}
