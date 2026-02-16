-- ===========================================
-- Audit / Logging CRUD
-- Maps to: PatternMatch (pattern_watcher.go)
--      and PatternCompositionMatch (pattern_composition.go)
-- ===========================================

-- ---- Pattern match log ----

-- name: CreatePatternMatchLog :one
INSERT INTO pattern_match_log (
    synapse_id,
    derived_type, derived_domain, depth, sig,
    occurrence, at, derived_id, rule_id,
    contributor_ids, anchor_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListPatternMatchLogs :many
SELECT * FROM pattern_match_log
WHERE synapse_id = $1
ORDER BY at DESC
LIMIT $2 OFFSET $3;

-- name: ListPatternMatchLogsByType :many
SELECT * FROM pattern_match_log
WHERE synapse_id = $1 AND derived_type = $2 AND derived_domain = $3
ORDER BY at DESC
LIMIT $4 OFFSET $5;

-- name: GetPatternMatchLogsByDerivedID :many
SELECT * FROM pattern_match_log
WHERE derived_id = $1;

-- name: DeletePatternMatchLog :exec
DELETE FROM pattern_match_log WHERE id = $1;

-- ---- Composition match log ----

-- name: CreateCompositionMatchLog :one
INSERT INTO composition_match_log (
    synapse_id, composition_id, recognized_at, derived_event_id, patterns
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListCompositionMatchLogs :many
SELECT * FROM composition_match_log
WHERE synapse_id = $1
ORDER BY recognized_at DESC
LIMIT $2 OFFSET $3;

-- name: ListCompositionMatchLogsBySpec :many
SELECT * FROM composition_match_log
WHERE synapse_id = $1 AND composition_id = $2
ORDER BY recognized_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteCompositionMatchLog :exec
DELETE FROM composition_match_log WHERE id = $1;
