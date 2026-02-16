-- ===========================================
-- Events CRUD
-- Maps to: Event (event.go)
-- ===========================================

-- name: CreateEvent :one
INSERT INTO events (id, synapse_id, event_type, event_domain, properties, timestamp, origin)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetEventByID :one
SELECT * FROM events WHERE id = $1;

-- name: GetEventsByIDs :many
SELECT * FROM events WHERE id = ANY($1::uuid[]);

-- name: GetEventsByType :many
SELECT * FROM events
WHERE synapse_id = $1 AND event_type = $2
ORDER BY timestamp;

-- name: GetEventsByTypeAndDomain :many
SELECT * FROM events
WHERE synapse_id = $1 AND event_type = $2 AND event_domain = $3
ORDER BY timestamp;

-- name: GetAllEventsBySynapse :many
SELECT * FROM events
WHERE synapse_id = $1
ORDER BY timestamp;

-- name: GetEventsBySynapse :many
SELECT * FROM events
WHERE synapse_id = $1
ORDER BY timestamp
LIMIT $2 OFFSET $3;

-- name: GetEventsByOrigin :many
SELECT * FROM events
WHERE synapse_id = $1 AND origin = $2
ORDER BY timestamp;

-- name: CountEvents :one
SELECT count(*) FROM events WHERE synapse_id = $1;

-- name: CountEventsByOrigin :one
SELECT count(*) FROM events WHERE synapse_id = $1 AND origin = $2;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;
