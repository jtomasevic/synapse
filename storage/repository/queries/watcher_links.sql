-- ===========================================
-- Watcher → Composition Wiring CRUD
-- Maps to: CompositePatternListener.watchers
--          in pattern_composition.go
-- ===========================================

-- name: CreateWatcherCompositionLink :exec
INSERT INTO watcher_composition_links (pattern_watcher_id, composition_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetCompositionsByWatcher :many
SELECT cs.*
FROM composition_specs cs
JOIN watcher_composition_links wcl ON cs.composition_id = wcl.composition_id
WHERE wcl.pattern_watcher_id = $1;

-- name: GetWatchersByComposition :many
SELECT pwc.*
FROM pattern_watcher_configs pwc
JOIN watcher_composition_links wcl ON pwc.id = wcl.pattern_watcher_id
WHERE wcl.composition_id = $1;

-- name: ListWatcherCompositionLinks :many
SELECT * FROM watcher_composition_links
WHERE pattern_watcher_id IN (
    SELECT id FROM pattern_watcher_configs WHERE synapse_id = $1
);

-- name: DeleteWatcherCompositionLink :exec
DELETE FROM watcher_composition_links
WHERE pattern_watcher_id = $1 AND composition_id = $2;

-- name: DeleteWatcherCompositionLinksByWatcher :exec
DELETE FROM watcher_composition_links
WHERE pattern_watcher_id = $1;

-- name: DeleteWatcherCompositionLinksByComposition :exec
DELETE FROM watcher_composition_links
WHERE composition_id = $1;
