# Airflow Provider

The Airflow provider manages Apache Airflow FAB (Flask-AppBuilder) Auth Manager resources.

## Example Usage

```hcl
provider "airflow" {
  host     = "https://airflow.example.com"
  username = "admin"
  password = "admin"
}

resource "airflow_user" "example" {
  username   = "data_engineer"
  first_name = "Jane"
  last_name  = "Smith"
  email      = "jane@example.com"
  password   = "SecurePass123!"
  roles      = ["User", "Viewer"]
}
```

## Authentication

The provider supports two authentication methods:

### Basic Authentication
```hcl
provider "airflow" {
  host     = "https://airflow.example.com"
  username = "admin"
  password = "password"
}
```

### JWT Token
```hcl
provider "airflow" {
  host  = "https://airflow.example.com"
  token = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."
}
```

## Schema

### Required

- `host` (String) - The base URL of the Airflow instance

### Optional

- `username` (String, Sensitive) - Username for basic authentication
- `password` (String, Sensitive) - Password for basic authentication
- `token` (String, Sensitive) - JWT Bearer token (alternative to username/password)
- `insecure_skip_verify` (Boolean) - Skip TLS certificate verification. Default: `false`
- `timeout` (Number) - Request timeout in seconds. Default: `30`

## Environment Variables

- `AIRFLOW_HOST` - Airflow API base URL
- `AIRFLOW_USERNAME` - Auth username
- `AIRFLOW_PASSWORD` - Auth password
- `AIRFLOW_TOKEN` - JWT token
