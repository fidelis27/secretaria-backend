CREATE TABLE audit_event_institutions (
    event_id VARCHAR(64) NOT NULL,
    institution_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (event_id, institution_id),
    CONSTRAINT fk_audit_event_institutions_event FOREIGN KEY (event_id)
        REFERENCES audit_events (event_id),
    CONSTRAINT fk_audit_event_institutions_institution FOREIGN KEY (institution_id)
        REFERENCES institutions (id),
    INDEX idx_audit_event_institutions_institution (institution_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;