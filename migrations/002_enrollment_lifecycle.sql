ALTER TABLE enrollments
    ADD COLUMN suspension_reason VARCHAR(500) NULL AFTER status,
    ADD COLUMN suspended_at TIMESTAMP NULL AFTER suspension_reason;