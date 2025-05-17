INSERT INTO tenants (id, name, plan, status, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000001', 'Tenant Alpha', 'basic', 'active', '2024-01-01 10:00:00+00', '2024-01-01 10:00:00+00'),
('a0000000-0000-0000-0000-000000000002', 'Tenant Beta', 'basic', 'active', '2024-01-01 10:01:00+00', '2024-01-01 10:01:00+00');

-- Password for all users is 'password123'
INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'alpha_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha Admin', 'admin', 'active', '2024-01-01 11:00:00+00', '2024-01-01 11:00:00+00'),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'alpha_user@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Alpha User', 'user', 'active', '2024-01-01 11:01:00+00', '2024-01-01 11:01:00+00'),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', 'beta_admin@example.com', '$2a$10$oadek4URiwb4gMH1/Llscusq97X3jLWTB2skaIYh5.8yy3W9.kGsS', 'Beta Admin', 'admin', 'active', '2024-01-01 11:02:00+00', '2024-01-01 11:02:00+00');