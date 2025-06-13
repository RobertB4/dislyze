-- Localhost Development Seed Data
-- Password for all users: "1234567890"

-- Insert Tenants
INSERT INTO tenants (id, name, enterprise_features) VALUES
('11111111-1111-1111-1111-111111111111', 'エンタープライズ株式会社', '{"rbac": {"enabled": true}}'),
('22222222-2222-2222-2222-222222222222', 'SMB株式会社', '{"rbac": {"enabled": false}}'),
('33333333-3333-3333-3333-333333333333', '内部株式会社', '{"rbac": {"enabled": true}}');

-- Insert default roles for all tenants (管理者, 編集者, 閲覧者)
-- Enterprise tenant default roles
INSERT INTO roles (id, tenant_id, name, description, is_default) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', '管理者', 'すべての機能にアクセス可能', true),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', '編集者', 'ユーザー管理以外の編集権限', true),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', '閲覧者', '閲覧権限のみ', true),
-- Custom role for enterprise tenant: ユーザー管理者  
('dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111111', 'ユーザー管理者', 'ユーザーの閲覧と編集権限を持つ', false),

-- SMB tenant default roles
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '22222222-2222-2222-2222-222222222222', '管理者', 'すべての機能にアクセス可能', true),
('ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222', '編集者', 'ユーザー管理以外の編集権限', true),
('11111111-2222-3333-4444-555555555555', '22222222-2222-2222-2222-222222222222', '閲覧者', '閲覧権限のみ', true),

-- Internal tenant default roles
('22222222-3333-4444-5555-666666666666', '33333333-3333-3333-3333-333333333333', '管理者', 'すべての機能にアクセス可能', true),
('33333333-4444-5555-6666-777777777777', '33333333-3333-3333-3333-333333333333', '編集者', 'ユーザー管理以外の編集権限', true),
('44444444-5555-6666-7777-888888888888', '33333333-3333-3333-3333-333333333333', '閲覧者', '閲覧権限のみ', true);

-- Assign permissions to admin roles (all edit permissions)
-- Enterprise tenant admin role permissions
INSERT INTO role_permissions (role_id, permission_id, tenant_id) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '6e95ed87-f380-41fe-bc5b-f8af002345a4', '11111111-1111-1111-1111-111111111111'), -- tenant edit
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', '11111111-1111-1111-1111-111111111111'), -- users edit
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'cccf277b-5fd5-4f1d-b763-ebf69973e5b7', '11111111-1111-1111-1111-111111111111'), -- roles edit
-- Assign permissions to ユーザー管理者 role (users view and edit only)
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', '11111111-1111-1111-1111-111111111111'), -- users edit

-- SMB tenant admin role permissions
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '6e95ed87-f380-41fe-bc5b-f8af002345a4', '22222222-2222-2222-2222-222222222222'), -- tenant edit
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', '22222222-2222-2222-2222-222222222222'), -- users edit
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'cccf277b-5fd5-4f1d-b763-ebf69973e5b7', '22222222-2222-2222-2222-222222222222'), -- roles edit

-- Internal tenant admin role permissions
('22222222-3333-4444-5555-666666666666', '6e95ed87-f380-41fe-bc5b-f8af002345a4', '33333333-3333-3333-3333-333333333333'), -- tenant edit
('22222222-3333-4444-5555-666666666666', 'db994eda-6ff7-4ae5-a675-3abe735ce9cc', '33333333-3333-3333-3333-333333333333'), -- users edit
('22222222-3333-4444-5555-666666666666', 'cccf277b-5fd5-4f1d-b763-ebf69973e5b7', '33333333-3333-3333-3333-333333333333'); -- roles edit

-- Insert Users
INSERT INTO users (id, tenant_id, email, password_hash, name, status) VALUES
-- Enterprise Users (101 users)
('a0000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'enterprise1@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '田中 太郎', 'active'),
('a0000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'enterprise2@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '佐藤 花子', 'active'),
('a0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'enterprise3@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '鈴木 一郎', 'active'),
('a0000000-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'enterprise4@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '高橋 美咲', 'active'),
('a0000000-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'enterprise5@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '伊藤 健太', 'active'),
('a0000000-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'enterprise6@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '渡辺 真由美', 'active'),
('a0000000-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'enterprise7@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '山本 慎太郎', 'active'),
('a0000000-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'enterprise8@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '中村 由美', 'active'),
('a0000000-0000-0000-0000-000000000009', '11111111-1111-1111-1111-111111111111', 'enterprise9@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '小林 大輔', 'active'),
('a0000000-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111', 'enterprise10@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '加藤 恵', 'active'),
('a0000000-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111', 'enterprise11@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '吉田 雄二', 'pending_verification'),
('a0000000-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111', 'enterprise12@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '福田 あかり', 'pending_verification'),
('a0000000-0000-0000-0000-000000000013', '11111111-1111-1111-1111-111111111111', 'enterprise13@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '森 拓也', 'pending_verification'),
('a0000000-0000-0000-0000-000000000014', '11111111-1111-1111-1111-111111111111', 'enterprise14@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '清水 愛', 'pending_verification'),
('a0000000-0000-0000-0000-000000000015', '11111111-1111-1111-1111-111111111111', 'enterprise15@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '三浦 翔太', 'pending_verification'),
('a0000000-0000-0000-0000-000000000016', '11111111-1111-1111-1111-111111111111', 'enterprise16@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '橋本 詩織', 'suspended'),
('a0000000-0000-0000-0000-000000000017', '11111111-1111-1111-1111-111111111111', 'enterprise17@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '岡田 浩一', 'suspended'),
('a0000000-0000-0000-0000-000000000018', '11111111-1111-1111-1111-111111111111', 'enterprise18@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '前田 麻衣', 'suspended'),
('a0000000-0000-0000-0000-000000000019', '11111111-1111-1111-1111-111111111111', 'enterprise19@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '長谷川 竜也', 'suspended'),
('a0000000-0000-0000-0000-000000000020', '11111111-1111-1111-1111-111111111111', 'enterprise20@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '石川 結衣', 'suspended'),
('a0000000-0000-0000-0000-000000000021', '11111111-1111-1111-1111-111111111111', 'enterprise21@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '田中 次郎', 'active'),
('a0000000-0000-0000-0000-000000000022', '11111111-1111-1111-1111-111111111111', 'enterprise22@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '佐藤 美代子', 'active'),
('a0000000-0000-0000-0000-000000000023', '11111111-1111-1111-1111-111111111111', 'enterprise23@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '鈴木 和子', 'active'),
('a0000000-0000-0000-0000-000000000024', '11111111-1111-1111-1111-111111111111', 'enterprise24@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '高橋 直樹', 'active'),
('a0000000-0000-0000-0000-000000000025', '11111111-1111-1111-1111-111111111111', 'enterprise25@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '渡辺 智子', 'active'),
('a0000000-0000-0000-0000-000000000026', '11111111-1111-1111-1111-111111111111', 'enterprise26@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '伊藤 孝志', 'active'),
('a0000000-0000-0000-0000-000000000027', '11111111-1111-1111-1111-111111111111', 'enterprise27@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '山本 恵美', 'active'),
('a0000000-0000-0000-0000-000000000028', '11111111-1111-1111-1111-111111111111', 'enterprise28@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '中村 裕司', 'active'),
('a0000000-0000-0000-0000-000000000029', '11111111-1111-1111-1111-111111111111', 'enterprise29@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '小林 千佳', 'active'),
('a0000000-0000-0000-0000-000000000030', '11111111-1111-1111-1111-111111111111', 'enterprise30@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '加藤 雄介', 'active'),
('a0000000-0000-0000-0000-000000000031', '11111111-1111-1111-1111-111111111111', 'enterprise31@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '吉田 明美', 'active'),
('a0000000-0000-0000-0000-000000000032', '11111111-1111-1111-1111-111111111111', 'enterprise32@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '村上 博之', 'active'),
('a0000000-0000-0000-0000-000000000033', '11111111-1111-1111-1111-111111111111', 'enterprise33@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '森田 由香', 'active'),
('a0000000-0000-0000-0000-000000000034', '11111111-1111-1111-1111-111111111111', 'enterprise34@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '松本 隆志', 'active'),
('a0000000-0000-0000-0000-000000000035', '11111111-1111-1111-1111-111111111111', 'enterprise35@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '井上 理恵', 'active'),
('a0000000-0000-0000-0000-000000000036', '11111111-1111-1111-1111-111111111111', 'enterprise36@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '木村 康弘', 'active'),
('a0000000-0000-0000-0000-000000000037', '11111111-1111-1111-1111-111111111111', 'enterprise37@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '林 さゆり', 'active'),
('a0000000-0000-0000-0000-000000000038', '11111111-1111-1111-1111-111111111111', 'enterprise38@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '清水 信一', 'active'),
('a0000000-0000-0000-0000-000000000039', '11111111-1111-1111-1111-111111111111', 'enterprise39@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '石川 和美', 'active'),
('a0000000-0000-0000-0000-000000000040', '11111111-1111-1111-1111-111111111111', 'enterprise40@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '山田 俊彦', 'active'),
('a0000000-0000-0000-0000-000000000041', '11111111-1111-1111-1111-111111111111', 'enterprise41@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '近藤 美穂', 'active'),
('a0000000-0000-0000-0000-000000000042', '11111111-1111-1111-1111-111111111111', 'enterprise42@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '橋本 秀樹', 'active'),
('a0000000-0000-0000-0000-000000000043', '11111111-1111-1111-1111-111111111111', 'enterprise43@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '池田 麻紀', 'active'),
('a0000000-0000-0000-0000-000000000044', '11111111-1111-1111-1111-111111111111', 'enterprise44@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '前田 正男', 'active'),
('a0000000-0000-0000-0000-000000000045', '11111111-1111-1111-1111-111111111111', 'enterprise45@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '岡田 夏美', 'active'),
('a0000000-0000-0000-0000-000000000046', '11111111-1111-1111-1111-111111111111', 'enterprise46@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '長谷川 健', 'active'),
('a0000000-0000-0000-0000-000000000047', '11111111-1111-1111-1111-111111111111', 'enterprise47@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '藤田 智恵', 'active'),
('a0000000-0000-0000-0000-000000000048', '11111111-1111-1111-1111-111111111111', 'enterprise48@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '後藤 良一', 'active'),
('a0000000-0000-0000-0000-000000000049', '11111111-1111-1111-1111-111111111111', 'enterprise49@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '新井 陽子', 'active'),
('a0000000-0000-0000-0000-000000000050', '11111111-1111-1111-1111-111111111111', 'enterprise50@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '竹内 浩二', 'active'),
('a0000000-0000-0000-0000-000000000051', '11111111-1111-1111-1111-111111111111', 'enterprise51@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '金子 香織', 'active'),
('a0000000-0000-0000-0000-000000000052', '11111111-1111-1111-1111-111111111111', 'enterprise52@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '原田 武史', 'active'),
('a0000000-0000-0000-0000-000000000053', '11111111-1111-1111-1111-111111111111', 'enterprise53@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '田村 真理', 'active'),
('a0000000-0000-0000-0000-000000000054', '11111111-1111-1111-1111-111111111111', 'enterprise54@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '西村 雅人', 'active'),
('a0000000-0000-0000-0000-000000000055', '11111111-1111-1111-1111-111111111111', 'enterprise55@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '中島 紀子', 'active'),
('a0000000-0000-0000-0000-000000000056', '11111111-1111-1111-1111-111111111111', 'enterprise56@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '小川 洋平', 'active'),
('a0000000-0000-0000-0000-000000000057', '11111111-1111-1111-1111-111111111111', 'enterprise57@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '中田 涼子', 'active'),
('a0000000-0000-0000-0000-000000000058', '11111111-1111-1111-1111-111111111111', 'enterprise58@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '松田 亮太', 'active'),
('a0000000-0000-0000-0000-000000000059', '11111111-1111-1111-1111-111111111111', 'enterprise59@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '内田 郁美', 'active'),
('a0000000-0000-0000-0000-000000000060', '11111111-1111-1111-1111-111111111111', 'enterprise60@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '斎藤 光男', 'active'),
('a0000000-0000-0000-0000-000000000061', '11111111-1111-1111-1111-111111111111', 'enterprise61@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '服部 恵子', 'active'),
('a0000000-0000-0000-0000-000000000062', '11111111-1111-1111-1111-111111111111', 'enterprise62@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '野村 晃', 'active'),
('a0000000-0000-0000-0000-000000000063', '11111111-1111-1111-1111-111111111111', 'enterprise63@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '吉野 瞳', 'active'),
('a0000000-0000-0000-0000-000000000064', '11111111-1111-1111-1111-111111111111', 'enterprise64@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '小野 健司', 'active'),
('a0000000-0000-0000-0000-000000000065', '11111111-1111-1111-1111-111111111111', 'enterprise65@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '平野 順子', 'active'),
('a0000000-0000-0000-0000-000000000066', '11111111-1111-1111-1111-111111111111', 'enterprise66@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '大橋 良太', 'active'),
('a0000000-0000-0000-0000-000000000067', '11111111-1111-1111-1111-111111111111', 'enterprise67@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '望月 まり', 'active'),
('a0000000-0000-0000-0000-000000000068', '11111111-1111-1111-1111-111111111111', 'enterprise68@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '星野 孝', 'active'),
('a0000000-0000-0000-0000-000000000069', '11111111-1111-1111-1111-111111111111', 'enterprise69@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '永田 静香', 'active'),
('a0000000-0000-0000-0000-000000000070', '11111111-1111-1111-1111-111111111111', 'enterprise70@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '久保 勇', 'active'),
('a0000000-0000-0000-0000-000000000071', '11111111-1111-1111-1111-111111111111', 'enterprise71@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '上田 亜紀', 'active'),
('a0000000-0000-0000-0000-000000000072', '11111111-1111-1111-1111-111111111111', 'enterprise72@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '福田 正樹', 'active'),
('a0000000-0000-0000-0000-000000000073', '11111111-1111-1111-1111-111111111111', 'enterprise73@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '宮崎 千春', 'active'),
('a0000000-0000-0000-0000-000000000074', '11111111-1111-1111-1111-111111111111', 'enterprise74@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '今井 元', 'active'),
('a0000000-0000-0000-0000-000000000075', '11111111-1111-1111-1111-111111111111', 'enterprise75@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '三浦 文子', 'active'),
('a0000000-0000-0000-0000-000000000076', '11111111-1111-1111-1111-111111111111', 'enterprise76@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '本田 信二', 'active'),
('a0000000-0000-0000-0000-000000000077', '11111111-1111-1111-1111-111111111111', 'enterprise77@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '横田 貴子', 'active'),
('a0000000-0000-0000-0000-000000000078', '11111111-1111-1111-1111-111111111111', 'enterprise78@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '田口 光一', 'active'),
('a0000000-0000-0000-0000-000000000079', '11111111-1111-1111-1111-111111111111', 'enterprise79@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '植田 弘美', 'active'),
('a0000000-0000-0000-0000-000000000080', '11111111-1111-1111-1111-111111111111', 'enterprise80@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '岩田 昭', 'active'),
('a0000000-0000-0000-0000-000000000081', '11111111-1111-1111-1111-111111111111', 'enterprise81@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '石田 真由美', 'active'),
('a0000000-0000-0000-0000-000000000082', '11111111-1111-1111-1111-111111111111', 'enterprise82@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '小島 徹', 'active'),
('a0000000-0000-0000-0000-000000000083', '11111111-1111-1111-1111-111111111111', 'enterprise83@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '古川 雅子', 'active'),
('a0000000-0000-0000-0000-000000000084', '11111111-1111-1111-1111-111111111111', 'enterprise84@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '安田 哲也', 'active'),
('a0000000-0000-0000-0000-000000000085', '11111111-1111-1111-1111-111111111111', 'enterprise85@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '杉山 英子', 'active'),
('a0000000-0000-0000-0000-000000000086', '11111111-1111-1111-1111-111111111111', 'enterprise86@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '田島 進', 'active'),
('a0000000-0000-0000-0000-000000000087', '11111111-1111-1111-1111-111111111111', 'enterprise87@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '山口 明美', 'active'),
('a0000000-0000-0000-0000-000000000088', '11111111-1111-1111-1111-111111111111', 'enterprise88@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '坂本 健', 'active'),
('a0000000-0000-0000-0000-000000000089', '11111111-1111-1111-1111-111111111111', 'enterprise89@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '森本 幸子', 'active'),
('a0000000-0000-0000-0000-000000000090', '11111111-1111-1111-1111-111111111111', 'enterprise90@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '大野 修', 'active'),
('a0000000-0000-0000-0000-000000000091', '11111111-1111-1111-1111-111111111111', 'enterprise91@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '細川 美樹', 'active'),
('a0000000-0000-0000-0000-000000000092', '11111111-1111-1111-1111-111111111111', 'enterprise92@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '谷口 豪', 'active'),
('a0000000-0000-0000-0000-000000000093', '11111111-1111-1111-1111-111111111111', 'enterprise93@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '増田 律子', 'active'),
('a0000000-0000-0000-0000-000000000094', '11111111-1111-1111-1111-111111111111', 'enterprise94@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '菅原 圭', 'active'),
('a0000000-0000-0000-0000-000000000095', '11111111-1111-1111-1111-111111111111', 'enterprise95@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '千葉 節子', 'active'),
('a0000000-0000-0000-0000-000000000096', '11111111-1111-1111-1111-111111111111', 'enterprise96@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '浜田 剛', 'active'),
('a0000000-0000-0000-0000-000000000097', '11111111-1111-1111-1111-111111111111', 'enterprise97@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '大塚 舞', 'active'),
('a0000000-0000-0000-0000-000000000098', '11111111-1111-1111-1111-111111111111', 'enterprise98@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '川村 明', 'active'),
('a0000000-0000-0000-0000-000000000099', '11111111-1111-1111-1111-111111111111', 'enterprise99@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '富田 和代', 'active'),
('a0000000-0000-0000-0000-000000000100', '11111111-1111-1111-1111-111111111111', 'enterprise100@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '石井 正', 'active'),
('a0000000-0000-0000-0000-000000000101', '11111111-1111-1111-1111-111111111111', 'enterprise101@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '鎌田 由紀', 'active'),

-- SMB Users (10 users)
('b0000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'smb1@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '青木 直人', 'active'),
('b0000000-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'smb2@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '松井 知子', 'active'),
('b0000000-0000-0000-0000-000000000003', '22222222-2222-2222-2222-222222222222', 'smb3@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '大石 悟', 'active'),
('b0000000-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222', 'smb4@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '野口 裕子', 'active'),
('b0000000-0000-0000-0000-000000000005', '22222222-2222-2222-2222-222222222222', 'smb5@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '森 達夫', 'active'),
('b0000000-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'smb6@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '菊池 典子', 'active'),
('b0000000-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'smb7@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '中山 拓', 'active'),
('b0000000-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'smb8@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '水野 彩', 'active'),
('b0000000-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'smb9@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '柴田 勝', 'active'),
('b0000000-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'smb10@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '坂田 恵里', 'active'),

-- Internal Users (2 users)  
('c0000000-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', 'internal1@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '管理 太郎', 'active'),
('c0000000-0000-0000-0000-000000000002', '33333333-3333-3333-3333-333333333333', 'internal2@localhost.com', '$2a$10$nAveWwnSGnoVTo91fCikNOBFOxptLVx1jnh0sRtpwQWxcJAXGfaRC', '運営 花子', 'active');

-- Assign user roles
INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES
-- Enterprise tenant role assignments
('a0000000-0000-0000-0000-000000000001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111'), -- user 1: 管理者
('a0000000-0000-0000-0000-000000000002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'), -- user 2: 編集者
('a0000000-0000-0000-0000-000000000003', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'), -- user 3: 編集者
('a0000000-0000-0000-0000-000000000004', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'), -- user 4: 編集者
('a0000000-0000-0000-0000-000000000005', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'), -- user 5: 編集者
('a0000000-0000-0000-0000-000000000006', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'), -- user 6: 編集者
('a0000000-0000-0000-0000-000000000007', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 7: 閲覧者
('a0000000-0000-0000-0000-000000000008', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 8: 閲覧者
('a0000000-0000-0000-0000-000000000009', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 9: 閲覧者
('a0000000-0000-0000-0000-000000000010', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 10: 閲覧者
('a0000000-0000-0000-0000-000000000011', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 11: 閲覧者
('a0000000-0000-0000-0000-000000000012', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 12: 閲覧者
('a0000000-0000-0000-0000-000000000013', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 13: 閲覧者
('a0000000-0000-0000-0000-000000000014', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 14: 閲覧者
('a0000000-0000-0000-0000-000000000015', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 15: 閲覧者
('a0000000-0000-0000-0000-000000000016', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 16: 閲覧者
('a0000000-0000-0000-0000-000000000017', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 17: 閲覧者
('a0000000-0000-0000-0000-000000000018', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 18: 閲覧者
('a0000000-0000-0000-0000-000000000019', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 19: 閲覧者
('a0000000-0000-0000-0000-000000000020', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 20: 閲覧者
('a0000000-0000-0000-0000-000000000021', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 21: 閲覧者
('a0000000-0000-0000-0000-000000000022', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 22: 閲覧者
('a0000000-0000-0000-0000-000000000023', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 23: 閲覧者
('a0000000-0000-0000-0000-000000000024', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 24: 閲覧者
('a0000000-0000-0000-0000-000000000025', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 25: 閲覧者
('a0000000-0000-0000-0000-000000000026', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 26: 閲覧者
('a0000000-0000-0000-0000-000000000027', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 27: 閲覧者
('a0000000-0000-0000-0000-000000000028', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 28: 閲覧者
('a0000000-0000-0000-0000-000000000029', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 29: 閲覧者
('a0000000-0000-0000-0000-000000000030', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 30: 閲覧者
('a0000000-0000-0000-0000-000000000031', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 31: 閲覧者
('a0000000-0000-0000-0000-000000000032', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 32: 閲覧者
('a0000000-0000-0000-0000-000000000033', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 33: 閲覧者
('a0000000-0000-0000-0000-000000000034', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 34: 閲覧者
('a0000000-0000-0000-0000-000000000035', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 35: 閲覧者
('a0000000-0000-0000-0000-000000000036', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 36: 閲覧者
('a0000000-0000-0000-0000-000000000037', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 37: 閲覧者
('a0000000-0000-0000-0000-000000000038', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 38: 閲覧者
('a0000000-0000-0000-0000-000000000039', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 39: 閲覧者
('a0000000-0000-0000-0000-000000000040', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 40: 閲覧者
('a0000000-0000-0000-0000-000000000041', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 41: 閲覧者
('a0000000-0000-0000-0000-000000000042', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 42: 閲覧者
('a0000000-0000-0000-0000-000000000043', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 43: 閲覧者
('a0000000-0000-0000-0000-000000000044', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 44: 閲覧者
('a0000000-0000-0000-0000-000000000045', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 45: 閲覧者
('a0000000-0000-0000-0000-000000000046', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 46: 閲覧者
('a0000000-0000-0000-0000-000000000047', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 47: 閲覧者
('a0000000-0000-0000-0000-000000000048', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 48: 閲覧者
('a0000000-0000-0000-0000-000000000049', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 49: 閲覧者
('a0000000-0000-0000-0000-000000000050', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 50: 閲覧者
('a0000000-0000-0000-0000-000000000051', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 51: 閲覧者
('a0000000-0000-0000-0000-000000000052', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 52: 閲覧者
('a0000000-0000-0000-0000-000000000053', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 53: 閲覧者
('a0000000-0000-0000-0000-000000000054', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 54: 閲覧者
('a0000000-0000-0000-0000-000000000055', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 55: 閲覧者
('a0000000-0000-0000-0000-000000000056', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 56: 閲覧者
('a0000000-0000-0000-0000-000000000057', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 57: 閲覧者
('a0000000-0000-0000-0000-000000000058', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 58: 閲覧者
('a0000000-0000-0000-0000-000000000059', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 59: 閲覧者
('a0000000-0000-0000-0000-000000000060', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 60: 閲覧者
('a0000000-0000-0000-0000-000000000061', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 61: 閲覧者
('a0000000-0000-0000-0000-000000000062', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 62: 閲覧者
('a0000000-0000-0000-0000-000000000063', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 63: 閲覧者
('a0000000-0000-0000-0000-000000000064', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 64: 閲覧者
('a0000000-0000-0000-0000-000000000065', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 65: 閲覧者
('a0000000-0000-0000-0000-000000000066', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 66: 閲覧者
('a0000000-0000-0000-0000-000000000067', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 67: 閲覧者
('a0000000-0000-0000-0000-000000000068', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 68: 閲覧者
('a0000000-0000-0000-0000-000000000069', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 69: 閲覧者
('a0000000-0000-0000-0000-000000000070', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 70: 閲覧者
('a0000000-0000-0000-0000-000000000071', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 71: 閲覧者
('a0000000-0000-0000-0000-000000000072', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 72: 閲覧者
('a0000000-0000-0000-0000-000000000073', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 73: 閲覧者
('a0000000-0000-0000-0000-000000000074', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 74: 閲覧者
('a0000000-0000-0000-0000-000000000075', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 75: 閲覧者
('a0000000-0000-0000-0000-000000000076', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 76: 閲覧者
('a0000000-0000-0000-0000-000000000077', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 77: 閲覧者
('a0000000-0000-0000-0000-000000000078', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 78: 閲覧者
('a0000000-0000-0000-0000-000000000079', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 79: 閲覧者
('a0000000-0000-0000-0000-000000000080', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 80: 閲覧者
('a0000000-0000-0000-0000-000000000081', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 81: 閲覧者
('a0000000-0000-0000-0000-000000000082', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 82: 閲覧者
('a0000000-0000-0000-0000-000000000083', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 83: 閲覧者
('a0000000-0000-0000-0000-000000000084', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 84: 閲覧者
('a0000000-0000-0000-0000-000000000085', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 85: 閲覧者
('a0000000-0000-0000-0000-000000000086', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 86: 閲覧者
('a0000000-0000-0000-0000-000000000087', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 87: 閲覧者
('a0000000-0000-0000-0000-000000000088', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 88: 閲覧者
('a0000000-0000-0000-0000-000000000089', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 89: 閲覧者
('a0000000-0000-0000-0000-000000000090', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 90: 閲覧者
('a0000000-0000-0000-0000-000000000091', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 91: 閲覧者
('a0000000-0000-0000-0000-000000000092', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 92: 閲覧者
('a0000000-0000-0000-0000-000000000093', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 93: 閲覧者
('a0000000-0000-0000-0000-000000000094', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 94: 閲覧者
('a0000000-0000-0000-0000-000000000095', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 95: 閲覧者
('a0000000-0000-0000-0000-000000000096', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 96: 閲覧者
('a0000000-0000-0000-0000-000000000097', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 97: 閲覧者
('a0000000-0000-0000-0000-000000000098', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 98: 閲覧者
('a0000000-0000-0000-0000-000000000099', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 99: 閲覧者
('a0000000-0000-0000-0000-000000000100', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 100: 閲覧者
('a0000000-0000-0000-0000-000000000101', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111'), -- user 101: 閲覧者

-- SMB tenant role assignments
('b0000000-0000-0000-0000-000000000001', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '22222222-2222-2222-2222-222222222222'), -- user 1: 管理者
('b0000000-0000-0000-0000-000000000002', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222'), -- user 2: 編集者
('b0000000-0000-0000-0000-000000000003', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222'), -- user 3: 編集者
('b0000000-0000-0000-0000-000000000004', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222'), -- user 4: 編集者
('b0000000-0000-0000-0000-000000000005', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222'), -- user 5: 編集者
('b0000000-0000-0000-0000-000000000006', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222'), -- user 6: 編集者
('b0000000-0000-0000-0000-000000000007', '11111111-2222-3333-4444-555555555555', '22222222-2222-2222-2222-222222222222'), -- user 7: 閲覧者
('b0000000-0000-0000-0000-000000000008', '11111111-2222-3333-4444-555555555555', '22222222-2222-2222-2222-222222222222'), -- user 8: 閲覧者
('b0000000-0000-0000-0000-000000000009', '11111111-2222-3333-4444-555555555555', '22222222-2222-2222-2222-222222222222'), -- user 9: 閲覧者
('b0000000-0000-0000-0000-000000000010', '11111111-2222-3333-4444-555555555555', '22222222-2222-2222-2222-222222222222'), -- user 10: 閲覧者

-- Internal tenant role assignments
('c0000000-0000-0000-0000-000000000001', '22222222-3333-4444-5555-666666666666', '33333333-3333-3333-3333-333333333333'), -- user 1: 管理者
('c0000000-0000-0000-0000-000000000002', '33333333-4444-5555-6666-777777777777', '33333333-3333-3333-3333-333333333333'); -- user 2: 編集者

-- Invitation tokens for pending users
INSERT INTO invitation_tokens (id, token_hash, tenant_id, user_id, expires_at, created_at) VALUES
-- Token for enterprise11@localhost.com (User ID: a0000000-0000-0000-0000-000000000011)
-- unhashed token value: 26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ
('d0000000-0000-0000-0000-000000000001', '3874d22b39c01882df8ee09c464ebc7441293d6e295299d35e26a8ec12f68a3d', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000011', NOW() + INTERVAL '48 hours', NOW()),
-- Token for enterprise12@localhost.com (User ID: a0000000-0000-0000-0000-000000000012) 
-- unhashed token value: MO-Cw4btd5KDj1TK16yxNo-zkFtkyjyjOlqUZ5AFWYA=
('d0000000-0000-0000-0000-000000000002', '7de910a7f03bb39e3c24375b8f34d787d8af449706904edc83986224de97c163', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000012', NOW() + INTERVAL '48 hours', NOW()),
-- Token for enterprise13@localhost.com (User ID: a0000000-0000-0000-0000-000000000013)
-- unhashed token value: CM-rBsZ3PDoIenm_Od4pRdUMAcIgqUWlqs3rMSvmrk0=
('d0000000-0000-0000-0000-000000000003', 'e6418e6d62b63b8a0a0eac30ba45797ac9273590c2110efc8ed5453711310473', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000013', NOW() + INTERVAL '48 hours', NOW());