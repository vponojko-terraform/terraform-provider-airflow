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
	Actions      []ActionPerm `json:"actions"`
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

	// Fetch all permissions (paginated) — FAB 3.6.0+ returns action-resource pairs
	var allPermissions []map[string]interface{}
	actionsSet := make(map[string]bool)
	resourcesSet := make(map[string]bool)
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

		for _, p := range permRes.Actions {
			allPermissions = append(allPermissions, map[string]interface{}{
				"action":   p.Action.Name,
				"resource": p.Resource.Name,
			})
			actionsSet[p.Action.Name] = true
			resourcesSet[p.Resource.Name] = true
		}

		if len(permRes.Actions) < limit {
			break
		}
		offset += limit
	}

	// Derive unique actions and resources from permissions
	actions := make([]string, 0, len(actionsSet))
	for a := range actionsSet {
		actions = append(actions, a)
	}
	d.Set("actions", actions)

	resources := make([]string, 0, len(resourcesSet))
	for r := range resourcesSet {
		resources = append(resources, r)
	}
	d.Set("resources", resources)

	d.Set("permissions", allPermissions)
	d.SetId("permissions")

	return diags
}
