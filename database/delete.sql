DELETE FROM email_change_tokens;
DELETE FROM invitation_tokens;
DELETE FROM refresh_tokens;
DELETE FROM password_reset_tokens;
DELETE FROM user_roles;
DELETE FROM role_permissions;
-- permission data is hardcoded and global for all tenants, no need to delete
DELETE FROM roles;
DELETE FROM users;
DELETE FROM tenants;