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
-- unhashed token value: 26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ
('c0000000-0000-0000-0000-000000000001', '3874d22b39c01882df8ee09c464ebc7441293d6e295299d35e26a8ec12f68a3d', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000003', NOW() + INTERVAL '48 hours', NOW()),
-- Token for pending_editor_for_rate_limit_test@example.com (User ID: b...5)
-- unhashed token value: MO-Cw4btd5KDj1TK16yxNo-zkFtkyjyjOlqUZ5AFWYA=
('c0000000-0000-0000-0000-000000000002', '7de910a7f03bb39e3c24375b8f34d787d8af449706904edc83986224de97c163', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000005', NOW() + INTERVAL '48 hours', NOW()),
-- Token for pending_editor_tenant_A_for_x_tenant_test@example.com (User ID: b...6)
-- unhashed token value: CM-rBsZ3PDoIenm_Od4pRdUMAcIgqUWlqs3rMSvmrk0=
('c0000000-0000-0000-0000-000000000003', 'e6418e6d62b63b8a0a0eac30ba45797ac9273590c2110efc8ed5453711310473', 'a0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000006', NOW() + INTERVAL '48 hours', NOW());