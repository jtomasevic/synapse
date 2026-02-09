-- ===========================================
-- Synapse Instances CRUD
-- Maps to: SynapseRuntime (synapse_runtime.go)
-- ===========================================

-- name: CreateSynapseInstance :one
INSERT INTO synapse_instances (id, description, max_depth, max_samples_per_lineage)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSynapseInstance :one
SELECT * FROM synapse_instances WHERE id = $1;

-- name: ListSynapseInstances :many
SELECT * FROM synapse_instances ORDER BY created_at;

-- name: UpdateSynapseInstance :exec
UPDATE synapse_instances SET
    description             = $2,
    max_depth               = $3,
    max_samples_per_lineage = $4,
    updated_at              = now()
WHERE id = $1;

-- name: DeleteSynapseInstance :exec
DELETE FROM synapse_instances WHERE id = $1;
