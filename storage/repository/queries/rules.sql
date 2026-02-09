-- ===========================================
-- Rules & Rule-EventType Bindings CRUD
-- Maps to: DeriveEventRule (rules.go)
--      and SynapseRuntime.rulesByType (synapse_runtime.go)
-- ===========================================

-- ---- Rules ----

-- name: CreateRule :one
INSERT INTO rules (
    id, synapse_id, action_type, condition_json,
    template_type, template_domain, template_props
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetRule :one
SELECT * FROM rules WHERE id = $1;

-- name: ListRules :many
SELECT * FROM rules
WHERE synapse_id = $1
ORDER BY created_at;

-- name: UpdateRule :exec
UPDATE rules SET
    action_type     = $2,
    condition_json  = $3,
    template_type   = $4,
    template_domain = $5,
    template_props  = $6
WHERE id = $1;

-- name: DeleteRule :exec
DELETE FROM rules WHERE id = $1;

-- ---- Rule <-> EventType bindings ----
-- Maps to: RegisterRule / RegisterRuleForTypes

-- name: AddRuleEventType :exec
INSERT INTO rule_event_types (rule_id, event_type)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetRuleEventTypes :many
SELECT * FROM rule_event_types WHERE rule_id = $1;

-- name: GetRulesByEventType :many
SELECT r.*
FROM rules r
JOIN rule_event_types ret ON r.id = ret.rule_id
WHERE ret.event_type = $1 AND r.synapse_id = $2;

-- name: ListRuleEventTypesBySynapse :many
SELECT ret.*
FROM rule_event_types ret
JOIN rules r ON ret.rule_id = r.id
WHERE r.synapse_id = $1;

-- name: DeleteRuleEventType :exec
DELETE FROM rule_event_types
WHERE rule_id = $1 AND event_type = $2;

-- name: DeleteRuleEventTypes :exec
DELETE FROM rule_event_types WHERE rule_id = $1;
