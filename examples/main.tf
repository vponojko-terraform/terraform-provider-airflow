terraform {
  required_providers {
    airflow = {
      source  = "example/airflow"
      version = "0.1.0"
    }
  }
}

provider "airflow" {
  host     = "https://airflow.example.com"
  username = "admin"
  password = "admin"
}

# Create a custom role for DAG readers
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
    resource = "Task Instances"
  }

  permission {
    action   = "can_read"
    resource = "Task Logs"
  }

  permission {
    action   = "menu_access"
    resource = "Browse"
  }
}

# Create a user with the custom role
resource "airflow_user" "data_engineer" {
  username   = "data_engineer"
  first_name = "Jane"
  last_name  = "Smith"
  email      = "jane.smith@example.com"
  password   = "SecurePassword123!"
  roles      = ["User", airflow_role.dag_reader.name]
  active     = true
}

# Look up an existing user
data "airflow_user" "admin" {
  username = "admin"
}

# Look up an existing role
data "airflow_role" "viewer" {
  name = "Viewer"
}

# Get all available permissions
data "airflow_permissions" "all" {}

output "admin_email" {
  value = data.airflow_user.admin.email
}

output "viewer_permissions_count" {
  value = length(data.airflow_role.viewer.permission)
}

output "available_actions" {
  value = data.airflow_permissions.all.actions
}
