-- ===========================================
-- Lineage Stats, Rule Counts & Samples CRUD
-- Maps to: lineageStats map[LineageKey]*LineageStats
--          in InMemoryStructuralMemory
-- ===========================================

-- ---- Lineage stats ----

-- name: IncrementLineageStats :exec
INSERT INTO lineage_stats (synapse_id, derived_type, derived_domain, depth, sig, count, last_seen)
VALUES ($1, $2, $3, $4, $5, 1, $6)
ON CONFLICT (synapse_id, derived_type, derived_domain, depth, sig) DO UPDATE
    SET count     = lineage_stats.count + 1,
        last_seen = EXCLUDED.last_seen;

-- name: UpsertLineageStats :exec
INSERT INTO lineage_stats (synapse_id, derived_type, derived_domain, depth, sig, count, last_seen)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (synapse_id, derived_type, derived_domain, depth, sig) DO UPDATE
    SET count     = EXCLUDED.count,
        last_seen = EXCLUDED.last_seen;

-- name: GetLineageStats :one
SELECT * FROM lineage_stats
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;

-- name: GetAllLineageStats :many
SELECT * FROM lineage_stats
WHERE synapse_id = $1
ORDER BY last_seen DESC;

-- name: ListLineageStats :many
SELECT * FROM lineage_stats
WHERE synapse_id = $1
ORDER BY last_seen DESC
LIMIT $2 OFFSET $3;

-- name: DeleteLineageStats :exec
DELETE FROM lineage_stats
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;

-- ---- Lineage rule counts ----

-- name: IncrementLineageRuleCount :exec
INSERT INTO lineage_rule_counts (synapse_id, derived_type, derived_domain, depth, sig, rule_id, count)
VALUES ($1, $2, $3, $4, $5, $6, 1)
ON CONFLICT (synapse_id, derived_type, derived_domain, depth, sig, rule_id) DO UPDATE
    SET count = lineage_rule_counts.count + 1;

-- name: GetLineageRuleCounts :many
SELECT * FROM lineage_rule_counts
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;

-- name: GetAllLineageRuleCountsBySynapse :many
SELECT * FROM lineage_rule_counts
WHERE synapse_id = $1;

-- name: DeleteLineageRuleCounts :exec
DELETE FROM lineage_rule_counts
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;

-- ---- Lineage samples ----

-- name: CreateLineageSample :one
INSERT INTO lineage_samples (
    synapse_id, derived_type, derived_domain, depth, sig,
    at, rule_id, derived_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetLineageSamples :many
SELECT * FROM lineage_samples
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5
ORDER BY at DESC;

-- name: CountLineageSamples :one
SELECT count(*) FROM lineage_samples
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;

-- name: GetAllLineageSamplesBySynapse :many
SELECT * FROM lineage_samples
WHERE synapse_id = $1
ORDER BY at DESC;

-- name: DeleteLineageSamples :exec
DELETE FROM lineage_samples
WHERE synapse_id     = $1
  AND derived_type   = $2
  AND derived_domain = $3
  AND depth          = $4
  AND sig            = $5;
