-- +goose Up
CREATE VIEW next_uuid AS
SELECT 
    lower(hex(randomblob(4))) || '-' ||
    lower(hex(randomblob(2))) || '-4' ||
    substr(lower(hex(randomblob(2))), 2) || '-' ||
    substr('89ab', abs(random()) % 4 + 1, 1) || 
    substr(lower(hex(randomblob(2))), 2) || '-' ||
    lower(hex(randomblob(6))) AS val;
INSERT INTO
	users (id, username, download_sessions_valid_from)
VALUES
	((SELECT val FROM next_uuid), "admin", DATE("now"));

-- +goose Down
DROP VIEW IF EXISTS next_uuid;
DELETE FROM users
WHERE
	username = "admin"
LIMIT
	1;