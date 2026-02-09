-- ===========================================
-- Event Derivations CRUD
-- Maps to: materializeDerived() / materializeFromTemplate()
--          in synapse_runtime.go
-- ===========================================

-- name: CreateEventDerivation :exec
INSERT INTO event_derivations (
    derived_event_id, synapse_id, origin_id, origin_type,
    anchor_event_id, contributor_ids, derived_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetEventDerivation :one
SELECT * FROM event_derivations WHERE derived_event_id = $1;

-- name: GetDerivationsByOrigin :many
SELECT * FROM event_derivations
WHERE synapse_id = $1 AND origin_id = $2
ORDER BY derived_at;

-- name: GetDerivationsByAnchor :many
SELECT * FROM event_derivations
WHERE anchor_event_id = $1
ORDER BY derived_at;

-- name: GetAllDerivationsBySynapse :many
SELECT * FROM event_derivations
WHERE synapse_id = $1
ORDER BY derived_at;

-- name: GetDerivationsBySynapse :many
SELECT * FROM event_derivations
WHERE synapse_id = $1
ORDER BY derived_at
LIMIT $2 OFFSET $3;

-- name: CountDerivationsBySynapse :one
SELECT count(*) FROM event_derivations WHERE synapse_id = $1;

-- name: DeleteEventDerivation :exec
DELETE FROM event_derivations WHERE derived_event_id = $1;
