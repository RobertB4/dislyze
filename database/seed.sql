INSERT INTO tenants (id, name, plan, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000001', 'Tenant Alpha', 'basic', '2024-01-01 10:00:00+00', '2024-01-01 10:00:00+00'),
('a0000000-0000-0000-0000-000000000002', 'Tenant Beta', 'basic', '2024-01-01 10:01:00+00', '2024-01-01 10:01:00+00');

-- Password for all users is 'password123' (except pending users, which is 'password')

-- Users for Tenant Alpha (a...1)
INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'alpha_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Admin', 'admin', 'active', '2024-01-01 11:00:00+00', '2024-01-01 11:00:00+00'),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'alpha_editor@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Editor', 'editor', 'active', '2024-01-01 11:01:00+00', '2024-01-01 11:01:00+00'),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_valid_token@example.com', '$2a$10$THcoOx7wWaJXBpVigz/d9etow0c5SZtARYw9V6Abx9Q5.ao8/EbZ', 'Pending Editor Valid Token', 'editor', 'pending_verification', '2024-01-01 11:02:00+00', '2024-01-01 11:02:00+00'),
('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'suspended_editor@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Suspended Editor', 'editor', 'suspended', '2024-01-01 11:03:00+00', '2024-01-01 11:03:00+00'),
('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_for_rate_limit_test@example.com', '$2a$10$THcoOx7wWaJXBpVigz/d9etow0c5SZtARYw9V6Abx9Q5.ao8/EbZ', 'Pending Editor Rate Limit Test', 'editor', 'pending_verification', '2024-01-01 11:04:00+00', '2024-01-01 11:04:00+00'),
('b0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_tenant_A_for_x_tenant_test@example.com', '$2a$10$hssV3F7y33jE2etR.IIkUu2t6d33p2uQ0fG1gS.gY0qY8G.x0aLqS', 'PendingXT Editor', 'editor', 'pending_verification', '2024-01-01 11:05:00+00', '2024-01-01 11:05:00+00');

-- User for Tenant Beta (a...2)
INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000002', 'beta_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Beta Admin', 'admin', 'active', '2024-01-01 11:06:00+00', '2024-01-01 11:06:00+00');

INSERT INTO invitation_tokens (id, token_hash, tenant_id, user_id, expires_at, created_at) VALUES
-- Token for pending_editor_valid_token@example.com (User ID: b...3)
('c0000000-0000-0000-0000-000000000001', '9e70b7c75e108197dc3af8fdf61459b4bfeb05561069d48ea0fe9c2fbe1290a8', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000003', NOW() + INTERVAL '48 hours', NOW()),
-- Token for pending_editor_for_rate_limit_test@example.com (User ID: b...5)
('c0000000-0000-0000-0000-000000000002', '2b4c689321d8bb2d938595f719a0779b875075210102720ff709c1909811c11c', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000005', NOW() + INTERVAL '48 hours', NOW()),
-- Token for pending_editor_tenant_A_for_x_tenant_test@example.com (User ID: b...6)
('c0000000-0000-0000-0000-000000000003', '2e7b6795e9a2c911a1793371a2990905f6e1d8919571999a4f4b0e928603783c', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000006', NOW() + INTERVAL '48 hours', NOW());