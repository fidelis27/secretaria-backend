INSERT INTO institutions (id, name, cnpj, status)
VALUES ('transfer-origin', 'Transfer Origin', NULL, 'active')
ON DUPLICATE KEY UPDATE name = VALUES(name), cnpj = VALUES(cnpj), status = VALUES(status);
