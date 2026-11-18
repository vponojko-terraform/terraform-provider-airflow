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

The FAB auth manager API (`/auth/fab/v1/*`) only accepts JWT bearer tokens. The provider supports two ways to get one:

### Username / Password

The provider exchanges the credentials for a JWT via `POST /auth/token`. Since FAB provider 3.11.0 this
only works for the configured `AUTH_TYPE` (database or LDAP); for OAuth or remote-user setups use a token.

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

- `username` (String, Sensitive) - Username used to obtain a JWT
- `password` (String, Sensitive) - Password used to obtain a JWT
- `token` (String, Sensitive) - JWT Bearer token (alternative to username/password)
- `insecure_skip_verify` (Boolean) - Skip TLS certificate verification. Default: `false`
- `timeout` (Number) - Request timeout in seconds. Default: `30`
- `use_basic_auth` (Boolean, Deprecated) - Send basic auth instead of JWT. Current FAB releases reject basic auth on `/auth/fab/v1/*`.

## Environment Variables

- `AIRFLOW_HOST` - Airflow API base URL
- `AIRFLOW_USERNAME` - Auth username
- `AIRFLOW_PASSWORD` - Auth password
- `AIRFLOW_TOKEN` - JWT token
- `AIRFLOW_USE_BASIC_AUTH` - Deprecated, see `use_basic_auth`
