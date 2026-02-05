package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var builtinRoles = map[string]bool{
	"Admin":  true,
	"Op":     true,
	"User":   true,
	"Viewer": true,
	"Public": true,
}

func resourceRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Unique role name (1-64 chars)",
			},
			"permission": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of permissions for the role",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The action name (e.g., can_read, can_edit, can_create, can_delete, menu_access)",
						},
						"resource": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The resource name (e.g., DAGs, Connections, Variables)",
						},
					},
				},
			},
		},
	}
}

type createRoleRequest struct {
	Name    string       `json:"name"`
	Actions []ActionPerm `json:"actions,omitempty"`
}

type updateRoleRequest struct {
	Actions []ActionPerm `json:"actions"`
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)

	name := d.Get("name").(string)

	tflog.Info(ctx, "Creating Airflow role", map[string]interface{}{
		"name": name,
	})

	// Check if trying to create a built-in role
	if builtinRoles[name] {
		return diag.Errorf("cannot create built-in role '%s', use terraform import instead", name)
	}

	req := createRoleRequest{
		Name: name,
	}

	if v, ok := d.GetOk("permission"); ok {
		permSet := v.(*schema.Set)
		req.Actions = make([]ActionPerm, permSet.Len())
		for i, p := range permSet.List() {
			perm := p.(map[string]interface{})
			req.Actions[i] = ActionPerm{
				Action:   ActionRef{Name: perm["action"].(string)},
				Resource: ResourceRef{Name: perm["resource"].(string)},
			}
		}
	}

	_, statusCode, err := client.DoRequest(ctx, "POST", "/auth/fab/v1/roles", req)
	if err != nil {
		if statusCode == 409 {
			return diag.Errorf("role '%s' already exists", name)
		}
		return diag.FromErr(err)
	}

	d.SetId(name)

	return resourceRoleRead(ctx, d, m)
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics

	name := d.Id()

	tflog.Debug(ctx, "Reading Airflow role", map[string]interface{}{
		"name": name,
	})

	resp, statusCode, err := client.DoRequest(ctx, "GET", fmt.Sprintf("/auth/fab/v1/roles/%s", URLEncode(name)), nil)
	if err != nil {
		if statusCode == 404 {
			tflog.Warn(ctx, "Role not found, removing from state", map[string]interface{}{
				"name": name,
			})
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	var role RoleResponse
	if err := json.Unmarshal(resp, &role); err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", role.Name)

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

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)

	name := d.Id()

	tflog.Info(ctx, "Updating Airflow role", map[string]interface{}{
		"name": name,
	})

	if d.HasChange("permission") {
		req := updateRoleRequest{
			Actions: []ActionPerm{},
		}

		if v, ok := d.GetOk("permission"); ok {
			permSet := v.(*schema.Set)
			req.Actions = make([]ActionPerm, permSet.Len())
			for i, p := range permSet.List() {
				perm := p.(map[string]interface{})
				req.Actions[i] = ActionPerm{
					Action:   ActionRef{Name: perm["action"].(string)},
					Resource: ResourceRef{Name: perm["resource"].(string)},
				}
			}
		}

		_, _, err := client.DoRequest(ctx, "PATCH", fmt.Sprintf("/auth/fab/v1/roles/%s", URLEncode(name)), req)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRoleRead(ctx, d, m)
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)
	var diags diag.Diagnostics

	name := d.Id()

	tflog.Info(ctx, "Deleting Airflow role", map[string]interface{}{
		"name": name,
	})

	// Check if trying to delete a built-in role
	if builtinRoles[name] {
		return diag.Errorf("cannot delete built-in role '%s'", name)
	}

	_, statusCode, err := client.DoRequest(ctx, "DELETE", fmt.Sprintf("/auth/fab/v1/roles/%s", URLEncode(name)), nil)
	if err != nil {
		if statusCode == 404 {
			// Already deleted
			d.SetId("")
			return diags
		}
		if statusCode == 403 {
			return diag.Errorf("cannot delete built-in role '%s'", name)
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
