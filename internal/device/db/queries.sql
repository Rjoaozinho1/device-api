-- name: CreateDevice :one
INSERT INTO device (name, brand, state)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDevice :one
SELECT ID, Name, Brand, State, CreationTime
FROM device
WHERE ID = $1;

-- name: ListDevices :many
SELECT ID, Name, Brand, State, CreationTime
FROM device
WHERE (sqlc.narg('brand')::text IS NULL OR brand = sqlc.narg('brand')::text)
  AND (sqlc.narg('state')::device_state IS NULL OR state = sqlc.narg('state')::device_state)
ORDER BY CreationTime DESC;

-- name: UpdateDevice :one
UPDATE device
SET Name = $2, Brand = $3, State = $4
WHERE ID = $1 AND State <> 'in-use'
RETURNING *;

-- name: DeleteDevice :execrows
DELETE FROM device
WHERE ID = $1 AND State <> 'in-use';
