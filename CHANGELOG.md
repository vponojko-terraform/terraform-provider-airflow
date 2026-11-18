# Changelog

## 0.2.0

Tested against Airflow 3.4.0 with FAB provider 3.11.0 and 3.10.2.

### Bug Fixes
- `airflow_role`: updating permissions no longer fails with 422 (PATCH now sends the required `name`)
- `airflow_user`: setting `roles = []` now removes all roles (previously the change was silently dropped)
- `airflow_user` / `airflow_role`: a 403 on delete now reports the API's permission error instead of a guessed cause

### Breaking / Behavior Changes
- `airflow_user.active` is now read-only and deprecated. The FAB API has no field for it, so `active = false` previously failed with 422; it is now rejected at validation.
- The provider no longer silently falls back to basic auth when it cannot obtain a JWT; it fails with the token error instead.
- `use_basic_auth` (provider, `airflow_user` resource and data source) is deprecated: current FAB releases accept only JWT on `/auth/fab/v1/*`.

### Dependencies
- terraform-plugin-sdk/v2 v2.40.1, terraform-plugin-log v0.11.0 (requires Go 1.25)

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
