ALTER TABLE `groups`
    ADD COLUMN name VARCHAR(255) NULL AFTER institution_id;

UPDATE `groups` g
INNER JOIN institutions i ON i.id = g.institution_id
SET g.name = CONCAT('Grupo ', i.name)
WHERE g.name IS NULL;

ALTER TABLE `groups`
    MODIFY COLUMN name VARCHAR(255) NOT NULL;

ALTER TABLE `groups`
    ADD CONSTRAINT uq_groups_institution_name UNIQUE (institution_id, name);
