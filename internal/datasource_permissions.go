package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// PermissionsResponse represents the permissions list response
type PermissionsResponse struct {
	Permissions  []ActionPerm `json:"permissions"`
	TotalEntries int          `json:"total_entries"`
}

// ActionsResponse represents the actions list response
type ActionsResponse struct {
	Actions      []ActionRef `json:"actions"`
	TotalEntries int         `json:"total_entries"`
}

// ResourcesResponse represents the resources list response
type ResourcesResponse struct {
	Resources    []ResourceRef `json:"resources"`
	TotalEntries int           `json:"total_entries"`
}

func dataSourcePermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePermissionsRead,
		Schema: map[string]*schema.Schema{
			"actions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available actions",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"resources": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available resources",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"permissions": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "List of all available action-resource permission combinations",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The action name",
						},
						"resource": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource name",
						},
					},
				},
			},
		},
	}
}

func dataSourcePermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics

	// Fetch actions
	actionsResp, _, err := client.DoRequest(ctx, "GET", "/auth/fab/v1/actions", nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var actionsRes ActionsResponse
	if err := json.Unmarshal(actionsResp, &actionsRes); err != nil {
		return diag.FromErr(err)
	}

	actions := make([]string, len(actionsRes.Actions))
	for i, a := range actionsRes.Actions {
		actions[i] = a.Name
	}
	d.Set("actions", actions)

	// Fetch resources
	resourcesResp, _, err := client.DoRequest(ctx, "GET", "/auth/fab/v1/resources", nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var resourcesRes ResourcesResponse
	if err := json.Unmarshal(resourcesResp, &resourcesRes); err != nil {
		return diag.FromErr(err)
	}

	resources := make([]string, len(resourcesRes.Resources))
	for i, r := range resourcesRes.Resources {
		resources[i] = r.Name
	}
	d.Set("resources", resources)

	// Fetch all permissions (paginated)
	var allPermissions []map[string]interface{}
	offset := 0
	limit := 100

	for {
		permResp, _, err := client.DoRequest(ctx, "GET", fmt.Sprintf("/auth/fab/v1/permissions?limit=%d&offset=%d", limit, offset), nil)
		if err != nil {
			return diag.FromErr(err)
		}

		var permRes PermissionsResponse
		if err := json.Unmarshal(permResp, &permRes); err != nil {
			return diag.FromErr(err)
		}

		for _, p := range permRes.Permissions {
			allPermissions = append(allPermissions, map[string]interface{}{
				"action":   p.Action.Name,
				"resource": p.Resource.Name,
			})
		}

		if len(permRes.Permissions) < limit {
			break
		}
		offset += limit
	}

	d.Set("permissions", allPermissions)
	d.SetId("permissions")

	return diags
}
