# Airflow Terraform Provider

Terraform provider for managing Apache Airflow FAB Auth Manager resources.

# Tested against  
- Airflow 3.1.7 
- FAB Provider 3.6.0 

**Note:** 3.2.0 version of FAB is buggy, therefore delete operations aren't working for FAB 3.2.0. Might persist accross other FAB releases.


## Requirements

- Terraform >= 0.13
- Go >= 1.21
- Apache Airflow 3.1.6+ with FAB Provider 3.0.2+

## Airflow Configuration

To use basic authentication with the FAB Auth Manager API, you must configure the `auth_backends` setting in your Airflow configuration:

```ini
[fab]
auth_backends = airflow.providers.fab.auth_manager.api.auth.backend.basic_auth
```

For Helm chart deployments, add this to your values file under `config`:

```yaml
config:
  fab:
    auth_backends: 'airflow.providers.fab.auth_manager.api.auth.backend.basic_auth'
```

Without this configuration, the FAB API endpoints (`/auth/fab/v1/*`) will return 401 Unauthorized even with valid credentials.

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
