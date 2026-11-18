# airflow_user

Manages an Airflow user.

## Example Usage

```hcl
resource "airflow_user" "example" {
  username   = "data_engineer"
  first_name = "Jane"
  last_name  = "Smith"
  email      = "jane@example.com"
  password   = "SecurePass123!"
  roles      = ["User", "Viewer"]
}
```

## Schema

### Required

- `username` (String) - Unique username (1-64 chars). Cannot be changed after creation.
- `first_name` (String) - User's first name
- `last_name` (String) - User's last name
- `email` (String) - User's email address
- `password` (String, Sensitive) - User's password. Write-only, cannot detect drift.

### Optional

- `roles` (List of String) - List of role names. Roles are replaced on update, not merged. An empty list removes all roles.
- `active` (Boolean, Deprecated) - The FAB API cannot activate or deactivate users. Only `true` is accepted (with a deprecation warning); remove it from configuration.
- `use_basic_auth` (Boolean, Deprecated) - Current FAB releases reject basic auth on `/auth/fab/v1/*`.

### Read-Only

- `active` (Boolean) - Whether the user is active, as reported by Airflow
- `last_login` (String) - Timestamp of last login
- `login_count` (Number) - Number of successful logins
- `fail_login_count` (Number) - Number of failed login attempts
- `created_on` (String) - Creation timestamp
- `changed_on` (String) - Last modification timestamp

## Import

```bash
terraform import airflow_user.example data_engineer
```
