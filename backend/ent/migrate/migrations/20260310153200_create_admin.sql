-- +goose Up
CREATE VIEW
	next_uuid AS
SELECT
	LOWER(hex (randomblob (4))) || '-' || LOWER(hex (randomblob (2))) || '-4' || SUBSTR(LOWER(hex (randomblob (2))), 2) || '-' || SUBSTR('89ab', ABS(RANDOM()) % 4 + 1, 1) || SUBSTR(LOWER(hex (randomblob (2))), 2) || '-' || LOWER(hex (randomblob (6))) AS val;

INSERT INTO
	users (
		id,
		username,
		created_at,
		updated_at,
		download_sessions_valid_from
	)
VALUES
	(
		(
			SELECT
				val
			FROM
				next_uuid
		),
		"admin",
		DATE ("now"),
		DATE ("now"),
		DATE ("now")
	);

-- +goose Down
DROP VIEW IF EXISTS next_uuid;

DELETE FROM users
WHERE
	username = "admin"
LIMIT
	1;