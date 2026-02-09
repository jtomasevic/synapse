-- ===========================================
-- Pattern Watcher Configs CRUD
-- Maps to: PatternWatcher / PatternConfig (pattern_watcher.go)
--      and WatchSpec (pattern_watcher.go)
-- ===========================================

-- name: CreatePatternWatcherConfig :one
INSERT INTO pattern_watcher_configs (id, synapse_id, depth, min_count, derived_types, domains)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPatternWatcherConfig :one
SELECT * FROM pattern_watcher_configs WHERE id = $1;

-- name: ListPatternWatcherConfigs :many
SELECT * FROM pattern_watcher_configs
WHERE synapse_id = $1
ORDER BY created_at;

-- name: UpdatePatternWatcherConfig :exec
UPDATE pattern_watcher_configs SET
    depth         = $2,
    min_count     = $3,
    derived_types = $4,
    domains       = $5
WHERE id = $1;

-- name: DeletePatternWatcherConfig :exec
DELETE FROM pattern_watcher_configs WHERE id = $1;
