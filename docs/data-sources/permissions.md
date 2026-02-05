# airflow_permissions (Data Source)

Retrieves all available actions, resources, and permission combinations.

## Example Usage

```hcl
data "airflow_permissions" "all" {}

output "available_actions" {
  value = data.airflow_permissions.all.actions
}

output "available_resources" {
  value = data.airflow_permissions.all.resources
}
```

## Schema

### Read-Only

- `actions` (List of String) - List of available action names
- `resources` (List of String) - List of available resource names
- `permissions` (Set of Object) - All action-resource permission combinations
  - `action` (String) - The action name
  - `resource` (String) - The resource name
