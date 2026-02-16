-- ===========================================
-- Revision Counters CRUD
-- Maps to: global, inRev, outRev, typeRev
--          in InMemoryStructuralMemory
-- ===========================================

-- ---- Global revision (per synapse) ----

-- name: InitGlobalRevision :exec
INSERT INTO revision_global (synapse_id, revision)
VALUES ($1, 0)
ON CONFLICT (synapse_id) DO NOTHING;

-- name: GetGlobalRevision :one
SELECT revision FROM revision_global WHERE synapse_id = $1;

-- name: IncrementGlobalRevision :one
UPDATE revision_global
SET revision = revision + 1
WHERE synapse_id = $1
RETURNING revision;

-- ---- Per-event revisions (in_rev / out_rev) ----

-- name: GetEventRevision :one
SELECT * FROM revision_by_event WHERE event_id = $1;

-- name: IncrementEventInRev :exec
INSERT INTO revision_by_event (event_id, synapse_id, in_rev, out_rev)
VALUES ($1, $2, 1, 0)
ON CONFLICT (event_id) DO UPDATE
    SET in_rev = revision_by_event.in_rev + 1;

-- name: IncrementEventOutRev :exec
INSERT INTO revision_by_event (event_id, synapse_id, in_rev, out_rev)
VALUES ($1, $2, 0, 1)
ON CONFLICT (event_id) DO UPDATE
    SET out_rev = revision_by_event.out_rev + 1;

-- name: ListEventRevisions :many
SELECT * FROM revision_by_event WHERE synapse_id = $1;

-- name: ListTypeRevisions :many
SELECT * FROM revision_by_type WHERE synapse_id = $1;

-- name: DeleteEventRevision :exec
DELETE FROM revision_by_event WHERE event_id = $1;

-- ---- Per-type revisions (per synapse) ----

-- name: GetTypeRevision :one
SELECT * FROM revision_by_type WHERE synapse_id = $1 AND event_type = $2;

-- name: IncrementTypeRevision :exec
INSERT INTO revision_by_type (synapse_id, event_type, type_rev)
VALUES ($1, $2, 1)
ON CONFLICT (synapse_id, event_type) DO UPDATE
    SET type_rev = revision_by_type.type_rev + 1;

-- name: DeleteTypeRevision :exec
DELETE FROM revision_by_type WHERE synapse_id = $1 AND event_type = $2;
