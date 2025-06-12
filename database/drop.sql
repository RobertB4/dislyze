DROP TABLE IF EXISTS goose_db_version;
DROP TABLE IF EXISTS email_change_tokens;
DROP TABLE IF EXISTS invitation_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
-- permission data is hardcoded and global for all tenants, no need to delete
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;