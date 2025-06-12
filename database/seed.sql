INSERT INTO tenants (id, name, features_config, stripe_customer_id, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000001', 'Tenant Alpha', '{}', null, '2024-01-01 10:00:00+00', '2024-01-01 10:00:00+00'),
('a0000000-0000-0000-0000-000000000002', 'Tenant Beta', '{}', null, '2024-01-01 10:01:00+00', '2024-01-01 10:01:00+00');

INSERT INTO roles (id, tenant_id, name, description, is_default, created_at, updated_at) VALUES
('e0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '管理者', 'すべての管理機能にアクセス可能', true, '2024-01-01 09:30:00+00', '2024-01-01 09:30:00+00'),
('e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '編集者', '限定的な編集権限', true, '2024-01-01 09:31:00+00', '2024-01-01 09:31:00+00');

INSERT INTO roles (id, tenant_id, name, description, is_default, created_at, updated_at) VALUES
('e0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', '管理者', 'すべての管理機能にアクセス可能', true, '2024-01-01 09:32:00+00', '2024-01-01 09:32:00+00'),
('e0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002', '編集者', '限定的な編集権限', true, '2024-01-01 09:33:00+00', '2024-01-01 09:33:00+00');

INSERT INTO role_permissions (role_id, permission_id, tenant_id, created_at) VALUES
-- Admin roles get all permissions
-- Tenant Alpha admin (e...1)
('e0000000-0000-0000-0000-000000000001', '98ca3771-c1d3-46fc-8ec6-04d53ea38322', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:00+00'), -- tenant.view
('e0000000-0000-0000-0000-000000000001', '6e95ed87-f380-41fe-bc5b-f8af002345a4', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:01+00'), -- tenant.edit
('e0000000-0000-0000-0000-000000000001', '3a52c807-ddcb-4044-8682-658e04800a8e', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:02+00'), -- users.view
('e0000000-0000-0000-0000-000000000001', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:03+00'), -- users.edit
('e0000000-0000-0000-0000-000000000001', '44b8962d-5dc5-490e-8469-03078668dd52', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:04+00'), -- roles.view
('e0000000-0000-0000-0000-000000000001', 'cccf277b-5fd5-4f1d-b763-ebf69973e5b7', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 09:40:04+00'), -- roles.edit

-- Tenant Beta admin (e...3)
('e0000000-0000-0000-0000-000000000003', '98ca3771-c1d3-46fc-8ec6-04d53ea38322', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:00+00'), -- tenant.view
('e0000000-0000-0000-0000-000000000003', '6e95ed87-f380-41fe-bc5b-f8af002345a4', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:01+00'), -- tenant.edit
('e0000000-0000-0000-0000-000000000003', '3a52c807-ddcb-4044-8682-658e04800a8e', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:02+00'), -- users.view
('e0000000-0000-0000-0000-000000000003', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:03+00'), -- users.edit
('e0000000-0000-0000-0000-000000000003', '44b8962d-5dc5-490e-8469-03078668dd52', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:04+00'), -- roles.view
('e0000000-0000-0000-0000-000000000003', 'cccf277b-5fd5-4f1d-b763-ebf69973e5b7', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 09:40:04+00'); -- roles.edit

-- Password for all users is 'password123' (except pending users, which is 'password')

-- Users for Tenant Alpha (a...1)
INSERT INTO users (id, tenant_id, email, password_hash, name, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'alpha_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Admin', 'active', '2024-01-01 11:00:00+00', '2024-01-01 11:00:00+00'),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'alpha_editor@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Editor', 'active', '2024-01-01 11:01:00+00', '2024-01-01 11:01:00+00'),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_valid_token@example.com', '$2a$10$THcoOx7wWaJXBpVigz/d9etow0c5SZtARYw9V6Abx9Q5.ao8/EbZ', 'Pending Editor Valid Token', 'pending_verification', '2024-01-01 11:02:00+00', '2024-01-01 11:02:00+00'),
('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'suspended_editor@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Suspended Editor', 'suspended', '2024-01-01 11:03:00+00', '2024-01-01 11:03:00+00'),
('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_for_rate_limit_test@example.com', '$2a$10$THcoOx7wWaJXBpVigz/d9etow0c5SZtARYw9V6Abx9Q5.ao8/EbZ', 'Pending Editor Rate Limit Test', 'pending_verification', '2024-01-01 11:04:00+00', '2024-01-01 11:04:00+00'),
('b0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'pending_editor_tenant_A_for_x_tenant_test@example.com', '$2a$10$hssV3F7y33jE2etR.IIkUu2t6d33p2uQ0fG1gS.gY0qY8G.x0aLqS', 'PendingXT Editor', 'pending_verification', '2024-01-01 11:05:00+00', '2024-01-01 11:05:00+00');

-- User for Tenant Beta (a...2)
INSERT INTO users (id, tenant_id, email, password_hash, name, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000002', 'beta_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Beta Admin', 'active', '2024-01-01 11:06:00+00', '2024-01-01 11:06:00+00');

INSERT INTO user_roles (user_id, role_id, tenant_id, created_at) VALUES
-- Tenant Alpha users
('b0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:10:00+00'), -- Alpha Admin -> admin role
('b0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:11:00+00'), -- Alpha Editor -> editor role
('b0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:12:00+00'), -- Pending Editor Valid Token -> editor role
('b0000000-0000-0000-0000-000000000004', 'e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:13:00+00'), -- Suspended Editor -> editor role
('b0000000-0000-0000-0000-000000000005', 'e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:14:00+00'), -- Pending Editor Rate Limit -> editor role
('b0000000-0000-0000-0000-000000000006', 'e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '2024-01-01 11:15:00+00'), -- Pending Editor X Tenant -> editor role
-- Tenant Beta users  
('b0000000-0000-0000-0000-000000000007', 'e0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', '2024-01-01 11:16:00+00'); -- Beta Admin -> admin role

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