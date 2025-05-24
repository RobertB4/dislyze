INSERT INTO tenants (id, name, plan, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000001', 'Tenant Alpha', 'basic', '2024-01-01 10:00:00+00', '2024-01-01 10:00:00+00'),
('a0000000-0000-0000-0000-000000000002', 'Tenant Beta', 'basic', '2024-01-01 10:01:00+00', '2024-01-01 10:01:00+00');

-- Password for all users is 'password123'
INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'alpha_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Admin', 'admin', 'active', '2024-01-01 11:00:00+00', '2024-01-01 11:00:00+00'),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'alpha_user@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha User', 'editor', 'active', '2024-01-01 11:01:00+00', '2024-01-01 11:01:00+00'),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', 'beta_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Beta Admin', 'admin', 'active', '2024-01-01 11:02:00+00', '2024-01-01 11:02:00+00'),
('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'pending_user_valid_token@example.com', '$2a$10$THcoOx7wWaJXBpVigz/d9etow0c5SZtARYw9V6Abx9Q5.ao8/EbZ', 'Pending User Valid Token', 'editor', 'pending_verification', '2024-01-01 11:03:00+00', '2024-01-01 11:03:00+00');

INSERT INTO invitation_tokens (id, token_hash, tenant_id, user_id, expires_at, created_at) VALUES
-- Token for the pending_user_valid_token@example.com (SHA256 hash of 'accept-invite-plain-valid-token-for-testing-123' to be replaced)
('c0000000-0000-0000-0000-000000000001', '9e70b7c75e108197dc3af8fdf61459b4bfeb05561069d48ea0fe9c2fbe1290a8', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000004', NOW() + INTERVAL '48 hours', NOW());