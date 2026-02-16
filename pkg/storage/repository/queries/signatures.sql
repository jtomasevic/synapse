-- ===========================================
-- Event Signatures CRUD
-- Maps to: sigs map[EventID][]uint64
--          in InMemoryStructuralMemory
-- ===========================================

-- name: UpsertEventSignature :exec
INSERT INTO event_signatures (event_id, depth, sig)
VALUES ($1, $2, $3)
ON CONFLICT (event_id, depth) DO UPDATE
    SET sig = EXCLUDED.sig;

-- name: GetEventSignature :one
SELECT * FROM event_signatures
WHERE event_id = $1 AND depth = $2;

-- name: GetEventSignatures :many
SELECT * FROM event_signatures
WHERE event_id = $1
ORDER BY depth;

-- name: GetSignaturesBySynapse :many
SELECT es.* FROM event_signatures es
JOIN events ev ON es.event_id = ev.id
WHERE ev.synapse_id = $1
ORDER BY es.event_id, es.depth;

-- name: DeleteEventSignatures :exec
DELETE FROM event_signatures WHERE event_id = $1;
