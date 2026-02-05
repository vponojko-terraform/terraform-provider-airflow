# airflow_user (Data Source)

Retrieves information about an existing Airflow user.

## Example Usage

```hcl
data "airflow_user" "admin" {
  username = "admin"
}

output "admin_email" {
  value = data.airflow_user.admin.email
}
```

## Schema

### Required

- `username` (String) - The username to look up

### Read-Only

- `id` (Number) - User's internal ID
- `first_name` (String) - User's first name
- `last_name` (String) - User's last name
- `email` (String) - User's email address
- `active` (Boolean) - Whether the user is active
- `roles` (List of String) - List of role names
- `last_login` (String) - Timestamp of last login
- `login_count` (Number) - Number of successful logins
- `fail_login_count` (Number) - Number of failed login attempts
- `created_on` (String) - Creation timestamp
- `changed_on` (String) - Last modification timestamp
