INSERT INTO users (id, name, email, status, super_admin)
VALUES ('demo-active', 'Demo Active', 'demo-active@example.com', 'active', TRUE)
ON DUPLICATE KEY UPDATE status = VALUES(status), super_admin = VALUES(super_admin);
