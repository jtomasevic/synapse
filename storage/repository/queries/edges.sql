-- ===========================================
-- Edges CRUD
-- Maps to: Edge (edge.go)
-- ===========================================

-- name: CreateEdge :exec
INSERT INTO edges (from_event_id, to_event_id, relation)
VALUES ($1, $2, $3);

-- name: GetEdge :one
SELECT * FROM edges
WHERE from_event_id = $1 AND to_event_id = $2 AND relation = $3;

-- name: GetEdgesFrom :many
SELECT * FROM edges WHERE from_event_id = $1;

-- name: GetEdgesTo :many
SELECT * FROM edges WHERE to_event_id = $1;

-- name: DeleteEdge :exec
DELETE FROM edges
WHERE from_event_id = $1 AND to_event_id = $2 AND relation = $3;

-- name: GetEdgesBySynapse :many
SELECT e.* FROM edges e
JOIN events ev ON e.from_event_id = ev.id
WHERE ev.synapse_id = $1;

-- name: DeleteEdgesFrom :exec
DELETE FROM edges WHERE from_event_id = $1;

-- name: DeleteEdgesTo :exec
DELETE FROM edges WHERE to_event_id = $1;
