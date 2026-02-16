-- ===========================================
-- Motif Stats & Instances CRUD
-- Maps to: motifs map[MotifKey]*MotifStats
--          in InMemoryStructuralMemory
-- ===========================================

-- ---- Motif stats ----

-- name: IncrementMotifStats :exec
INSERT INTO motif_stats (synapse_id, derived_type, derived_domain, contributor_sig, rule_id, count, last_seen)
VALUES ($1, $2, $3, $4, $5, 1, $6)
ON CONFLICT (synapse_id, derived_type, derived_domain, contributor_sig, rule_id) DO UPDATE
    SET count     = motif_stats.count + 1,
        last_seen = EXCLUDED.last_seen;

-- name: UpsertMotifStats :exec
INSERT INTO motif_stats (synapse_id, derived_type, derived_domain, contributor_sig, rule_id, count, last_seen)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (synapse_id, derived_type, derived_domain, contributor_sig, rule_id) DO UPDATE
    SET count     = EXCLUDED.count,
        last_seen = EXCLUDED.last_seen;

-- name: GetMotifStats :one
SELECT * FROM motif_stats
WHERE synapse_id      = $1
  AND derived_type    = $2
  AND derived_domain  = $3
  AND contributor_sig = $4
  AND rule_id         = $5;

-- name: ListMotifStats :many
SELECT * FROM motif_stats
WHERE synapse_id = $1
ORDER BY last_seen DESC;

-- name: DeleteMotifStats :exec
DELETE FROM motif_stats
WHERE synapse_id      = $1
  AND derived_type    = $2
  AND derived_domain  = $3
  AND contributor_sig = $4
  AND rule_id         = $5;

-- ---- Motif instances ----

-- name: CreateMotifInstance :one
INSERT INTO motif_instances (
    synapse_id, derived_type, derived_domain, contributor_sig, rule_id,
    at, derived_id, contributor_ids
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetMotifInstances :many
SELECT * FROM motif_instances
WHERE synapse_id      = $1
  AND derived_type    = $2
  AND derived_domain  = $3
  AND contributor_sig = $4
  AND rule_id         = $5
ORDER BY at DESC;

-- name: CountMotifInstances :one
SELECT count(*) FROM motif_instances
WHERE synapse_id      = $1
  AND derived_type    = $2
  AND derived_domain  = $3
  AND contributor_sig = $4
  AND rule_id         = $5;

-- name: GetAllMotifInstancesBySynapse :many
SELECT * FROM motif_instances
WHERE synapse_id = $1
ORDER BY at DESC;

-- name: DeleteMotifInstances :exec
DELETE FROM motif_instances
WHERE synapse_id      = $1
  AND derived_type    = $2
  AND derived_domain  = $3
  AND contributor_sig = $4
  AND rule_id         = $5;
