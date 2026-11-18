# Airflow Terraform Provider

Terraform provider for managing Apache Airflow FAB Auth Manager resources.

# Tested against

- Airflow 3.4.0 (`apache/airflow:3.4.0` Docker image)
- FAB Provider 3.11.0 (also 3.10.2, the version bundled in the 3.4.0 image)

**Note:** FAB 3.2.0 is buggy (delete operations fail); avoid it.

## Requirements

- Terraform >= 0.13
- Go >= 1.25 (to build)
- Apache Airflow 3.x with the FAB auth manager (`AIRFLOW__CORE__AUTH_MANAGER=airflow.providers.fab.auth_manager.fab_auth_manager.FabAuthManager`)

## Authentication

The FAB auth manager API (`/auth/fab/v1/*`) only accepts JWT bearer tokens. With `username`/`password`
the provider obtains a token from `POST /auth/token`; alternatively pass a pre-obtained `token`.
Basic auth (and the `[fab] auth_backends` setting) is not used by these endpoints, so `use_basic_auth`
is deprecated.

Since FAB 3.11.0, `POST /auth/token` only accepts a username/password for the configured `AUTH_TYPE`
(database or LDAP). For OAuth / remote-user setups, supply `token` instead.

The authenticating user needs permission to manage users and roles (e.g. the `Admin` role).

## Setup

```bash
make build
make install
```

## Configuration

```hcl
provider "airflow" {
  host     = "https://airflow.example.com"
  username = "admin"
  password = "password"
}
```

## Environment Variables

```bash
export AIRFLOW_HOST="https://airflow.example.com"
export AIRFLOW_USERNAME="admin"
export AIRFLOW_PASSWORD="password"
```

## Resources

- `airflow_user` - Manage users
- `airflow_role` - Manage roles with permissions

## Data Sources

- `airflow_user` - Look up existing user
- `airflow_role` - Look up existing role
- `airflow_permissions` - List available actions and resources

## Quick Example

```hcl
resource "airflow_role" "dag_reader" {
  name = "dag_reader"

  permission {
    action   = "can_read"
    resource = "DAGs"
  }
}

resource "airflow_user" "example" {
  username   = "data_engineer"
  first_name = "Jane"
  last_name  = "Smith"
  email      = "jane@example.com"
  password   = "SecurePass123!"
  roles      = ["User", airflow_role.dag_reader.name]
}
```
