# airflow_role

Manages an Airflow role with granular permissions.

## Example Usage

```hcl
resource "airflow_role" "dag_reader" {
  name = "dag_reader"

  permission {
    action   = "can_read"
    resource = "DAGs"
  }

  permission {
    action   = "can_read"
    resource = "DAG Runs"
  }

  permission {
    action   = "can_read"
    resource = "Task Logs"
  }
}
```

## Schema

### Required

- `name` (String) - Unique role name. Cannot be changed after creation.

### Optional

- `permission` (Block Set) - List of permissions. Permissions are replaced on update.
  - `action` (String) - Action name: `can_read`, `can_edit`, `can_create`, `can_delete`, `menu_access`
  - `resource` (String) - Resource name (e.g., `DAGs`, `Connections`, `Variables`)

## Available Resources

Common resources: `Admin`, `Connections`, `DAGs`, `DAG Runs`, `DAG Code`, `Pools`, `Variables`, `Task Instances`, `Task Logs`, `Users`, `Roles`

For DAG-specific permissions, use `DAG:{dag_id}` format (supports wildcards: `DAG:team_*`).

## Built-in Roles

Built-in roles (`Admin`, `Op`, `User`, `Viewer`, `Public`) cannot be created or deleted. Use `terraform import` to manage them.

## Import

```bash
terraform import airflow_role.example custom_role
```
