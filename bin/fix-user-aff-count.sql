-- Repair historical invitation counts from users.inviter_id.
--
-- Compatible with SQLite, MySQL 5.7.8+, and PostgreSQL 9.6+.
-- Stop application writes and back up the database before running this file.
-- Soft-deleted users are included. Counts are only increased so invitees that
-- were hard-deleted do not cause valid historical counts to be reduced.

-- Preview rows that need repair.
SELECT
    u.id,
    u.username,
    u.aff_count AS stored_aff_count,
    COUNT(i.id) AS relation_aff_count
FROM users AS u
LEFT JOIN users AS i ON i.inviter_id = u.id
GROUP BY u.id, u.username, u.aff_count
HAVING COALESCE(u.aff_count, 0) < COUNT(i.id)
ORDER BY u.id;

BEGIN;

CREATE TEMPORARY TABLE user_aff_count_fix (
    user_id BIGINT PRIMARY KEY,
    relation_aff_count BIGINT NOT NULL
);

INSERT INTO user_aff_count_fix (user_id, relation_aff_count)
SELECT
    u.id,
    COUNT(i.id)
FROM users AS u
LEFT JOIN users AS i ON i.inviter_id = u.id
GROUP BY u.id;

UPDATE users
SET aff_count = (
    SELECT f.relation_aff_count
    FROM user_aff_count_fix AS f
    WHERE f.user_id = users.id
)
WHERE COALESCE(aff_count, 0) < (
    SELECT f.relation_aff_count
    FROM user_aff_count_fix AS f
    WHERE f.user_id = users.id
);

DROP TABLE user_aff_count_fix;

COMMIT;

-- This query should return no rows after the repair.
SELECT
    u.id,
    u.username,
    u.aff_count AS stored_aff_count,
    COUNT(i.id) AS relation_aff_count
FROM users AS u
LEFT JOIN users AS i ON i.inviter_id = u.id
GROUP BY u.id, u.username, u.aff_count
HAVING COALESCE(u.aff_count, 0) < COUNT(i.id)
ORDER BY u.id;
