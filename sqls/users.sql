-- name: CreateUser :one
INSERT INTO users (id, external_id, name)
VALUES (?1, ?2, ?3)
RETURNING id, external_id, created_at, name;

-- name: GetUser :one
SELECT id, name, external_id, created_at
FROM users
WHERE id = ?1;

-- name: GetUserByExternalID :one
SELECT id, name, external_id, created_at
FROM users
WHERE external_id = ?1;

-- name: DeleteUser :exec
DELETE
FROM users
WHERE id = ?1;
