-- name: AddDevice :one
INSERT INTO device (id)
VALUES (sqlc.arg('id'))
RETURNING id;

-- name: GetDeviceDetail :one
SELECT id
from device;