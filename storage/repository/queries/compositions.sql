-- ===========================================
-- Composition Specs & Required Patterns CRUD
-- Maps to: PatternCompositionSpec (pattern_composition.go)
--      and PatternIdentifier
-- ===========================================

-- ---- Composition specs ----

-- name: CreateCompositionSpec :one
INSERT INTO composition_specs (
    composition_id, synapse_id,
    time_window_within, time_window_unit,
    derived_template_type, derived_template_domain, derived_template_props
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCompositionSpec :one
SELECT * FROM composition_specs WHERE composition_id = $1;

-- name: ListCompositionSpecs :many
SELECT * FROM composition_specs
WHERE synapse_id = $1
ORDER BY created_at;

-- name: UpdateCompositionSpec :exec
UPDATE composition_specs SET
    time_window_within      = $2,
    time_window_unit        = $3,
    derived_template_type   = $4,
    derived_template_domain = $5,
    derived_template_props  = $6
WHERE composition_id = $1;

-- name: DeleteCompositionSpec :exec
DELETE FROM composition_specs WHERE composition_id = $1;

-- ---- Required patterns per composition ----
-- Maps to: PatternCompositionSpec.RequiredPatterns and MinOccurrences

-- name: AddCompositionRequiredPattern :exec
INSERT INTO composition_required_patterns (
    composition_id, event_type, event_domain, min_occurrences
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (composition_id, event_type, event_domain) DO UPDATE
    SET min_occurrences = EXCLUDED.min_occurrences;

-- name: GetCompositionRequiredPatterns :many
SELECT * FROM composition_required_patterns
WHERE composition_id = $1;

-- name: DeleteCompositionRequiredPattern :exec
DELETE FROM composition_required_patterns
WHERE composition_id = $1 AND event_type = $2 AND event_domain = $3;

-- name: DeleteCompositionRequiredPatterns :exec
DELETE FROM composition_required_patterns
WHERE composition_id = $1;
