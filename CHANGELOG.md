# Changelog

## 0.1.0

Initial release with FAB Auth Manager support for Apache Airflow 3.x.

### Resources
- `airflow_user` - Manage users with role assignments
- `airflow_role` - Manage custom roles with granular permissions

### Data Sources
- `airflow_user` - Look up existing users
- `airflow_role` - Look up existing roles
- `airflow_permissions` - Discover available actions and resources

### Features
- Basic Auth and JWT token authentication
- Import support for existing resources
- Built-in role protection (Admin, Op, User, Viewer, Public)
