# airflow_role (Data Source)

Retrieves information about an existing Airflow role.

## Example Usage

```hcl
data "airflow_role" "viewer" {
  name = "Viewer"
}

output "viewer_permissions" {
  value = data.airflow_role.viewer.permission
}
```

## Schema

### Required

- `name` (String) - The role name to look up

### Read-Only

- `permission` (Set of Object) - List of permissions
  - `action` (String) - The action name
  - `resource` (String) - The resource name
