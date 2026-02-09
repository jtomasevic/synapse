-- ============================================================
-- Synapse Event Network - PostgreSQL Schema
-- ============================================================
-- This script is idempotent: all objects use IF NOT EXISTS.
--
-- Entity mapping from Go (pkg/event_network) → SQL:
--
--   SynapseRuntime          → synapse_instances (top-level owner)
--   InMemoryStructuralMemory config → synapse_instances.max_depth, max_samples_per_lineage
--   Event                   → events (with synapse_id FK + origin tracking)
--   Edge                    → edges
--   materializeDerived()    → event_derivations (who derived what, from whom)
--   revision counters       → revision_global, revision_by_event, revision_by_type
--   sigs[EventID][]uint64   → event_signatures
--   MotifKey/Stats/Instance → motif_stats, motif_instances
--   LineageKey/Stats/Sample → lineage_stats, lineage_rule_counts, lineage_samples
--   DeriveEventRule         → rules (with condition_json serialization)
--   RegisterRule/ForTypes   → rule_event_types
--   PatternConfig/WatchSpec → pattern_watcher_configs
--   PatternCompositionSpec  → composition_specs, composition_required_patterns
--   CompositePatternListener wiring → watcher_composition_links
--   PatternMatch (audit)    → pattern_match_log
--   PatternCompositionMatch → composition_match_log
-- ============================================================

-- ============================================================
-- 0. SYNAPSE INSTANCE (top-level owner of everything)
-- ============================================================
-- Maps to: SynapseRuntime (synapse_runtime.go)
-- A Synapse instance owns one EventNetwork, one StructuralMemory,
-- a set of Rules, PatternWatchers, and CompositionWatchers.

CREATE TABLE IF NOT EXISTS synapse_instances (
    id              TEXT        PRIMARY KEY,
    description     TEXT        NOT NULL DEFAULT '',

    -- StructuralMemory config (from InMemoryStructuralMemory)
    -- Maps to: maxDepth, maxSamplesPerLineage in in_memory_structural_memory.go
    max_depth               INT     NOT NULL DEFAULT 5,
    max_samples_per_lineage INT     NOT NULL DEFAULT 20,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- 1. CORE GRAPH
-- ============================================================

-- Maps to: Event (event.go)
-- Fields: ID, EventType, EventDomain, Properties, Timestamp
CREATE TABLE IF NOT EXISTS events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    synapse_id      TEXT        NOT NULL REFERENCES synapse_instances(id),
    event_type      TEXT        NOT NULL,
    event_domain    TEXT        NOT NULL,
    properties      JSONB       NOT NULL DEFAULT '{}',
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Origin tracking: leaf events are ingested externally, derived events
    -- are created by rules (materializeDerived) or composition watchers.
    -- Maps to: the distinction between Synapse.Ingest() leaf path vs
    --          materializeFromTemplate() derived path in synapse_runtime.go
    origin          TEXT        NOT NULL DEFAULT 'ingested'
                    CHECK (origin IN ('ingested', 'derived', 'composition')),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_synapse    ON events (synapse_id);
CREATE INDEX IF NOT EXISTS idx_events_type       ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_domain     ON events (event_domain);
CREATE INDEX IF NOT EXISTS idx_events_type_domain ON events (event_type, event_domain);
CREATE INDEX IF NOT EXISTS idx_events_timestamp  ON events (timestamp);
CREATE INDEX IF NOT EXISTS idx_events_origin     ON events (origin);
CREATE INDEX IF NOT EXISTS idx_events_props      ON events USING GIN (properties);

-- Maps to: Edge (edge.go)
-- Fields: From, To, Relation
CREATE TABLE IF NOT EXISTS edges (
    from_event_id   UUID    NOT NULL REFERENCES events(id),
    to_event_id     UUID    NOT NULL REFERENCES events(id),
    relation        TEXT    NOT NULL,

    PRIMARY KEY (from_event_id, to_event_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges (from_event_id);
CREATE INDEX IF NOT EXISTS idx_edges_to   ON edges (to_event_id);

-- ============================================================
-- 1b. EVENT DERIVATION PROVENANCE
-- ============================================================
-- Maps to: the materializeDerived() / materializeFromTemplate() call in synapse_runtime.go
-- Records: "event X was derived by rule/composition Y from anchor Z and contributors [A,B,C]"
-- This is the fact that the in-memory code tracks transiently via
-- contributedEvents map[EventID][]Event and rulesId map[EventID]string in Ingest().

CREATE TABLE IF NOT EXISTS event_derivations (
    derived_event_id    UUID    NOT NULL REFERENCES events(id),
    synapse_id          TEXT    NOT NULL REFERENCES synapse_instances(id),

    -- Which rule or composition created this event
    -- Maps to: rule.GetID() or PatternCompositionSpec.CompositionID
    origin_id           TEXT    NOT NULL,
    -- 'rule' for DeriveEventRule, 'composition' for PatternCompositionWatcher
    origin_type         TEXT    NOT NULL DEFAULT 'rule'
                        CHECK (origin_type IN ('rule', 'composition')),

    -- The anchor event that triggered the rule
    -- Maps to: the 'cur' event in the Ingest() loop, or pattern DerivedIDs for composition
    anchor_event_id     UUID    REFERENCES events(id),

    -- All contributor events (including anchor)
    -- Maps to: contributedEvents[derived.ID] in Ingest()
    contributor_ids     UUID[]  NOT NULL,

    -- Timestamp of derivation
    derived_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (derived_event_id)
);

CREATE INDEX IF NOT EXISTS idx_event_derivations_synapse ON event_derivations (synapse_id);
CREATE INDEX IF NOT EXISTS idx_event_derivations_origin  ON event_derivations (origin_id);
CREATE INDEX IF NOT EXISTS idx_event_derivations_anchor  ON event_derivations (anchor_event_id);

-- ============================================================
-- 2. STRUCTURAL MEMORY
-- ============================================================
-- Maps to: InMemoryStructuralMemory (in_memory_structural_memory.go)
-- All structural memory tables are scoped to a synapse instance.

-- 2a. Revision counters (cache invalidation)
-- Maps to: global, inRev, outRev, typeRev fields

CREATE TABLE IF NOT EXISTS revision_global (
    synapse_id  TEXT    PRIMARY KEY REFERENCES synapse_instances(id),
    revision    BIGINT  NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS revision_by_event (
    event_id    UUID    NOT NULL REFERENCES events(id),
    synapse_id  TEXT    NOT NULL REFERENCES synapse_instances(id),
    in_rev      BIGINT  NOT NULL DEFAULT 0,
    out_rev     BIGINT  NOT NULL DEFAULT 0,

    PRIMARY KEY (event_id)
);

CREATE INDEX IF NOT EXISTS idx_revision_by_event_synapse ON revision_by_event (synapse_id);

CREATE TABLE IF NOT EXISTS revision_by_type (
    synapse_id  TEXT    NOT NULL REFERENCES synapse_instances(id),
    event_type  TEXT    NOT NULL,
    type_rev    BIGINT  NOT NULL DEFAULT 0,

    PRIMARY KEY (synapse_id, event_type)
);

-- 2b. Event signatures (lineage hashes at depth 0..maxDepth)
-- Maps to: sigs map[EventID][]uint64

CREATE TABLE IF NOT EXISTS event_signatures (
    event_id    UUID    NOT NULL REFERENCES events(id),
    depth       INT     NOT NULL CHECK (depth >= 0),
    sig         BIGINT  NOT NULL,

    PRIMARY KEY (event_id, depth)
);

-- 2c. Motif stats (1-hop derivation shape memory)
-- Maps to: motifs map[MotifKey]*MotifStats

CREATE TABLE IF NOT EXISTS motif_stats (
    synapse_id          TEXT    NOT NULL REFERENCES synapse_instances(id),
    derived_type        TEXT    NOT NULL,
    derived_domain      TEXT    NOT NULL,
    contributor_sig     TEXT    NOT NULL,
    rule_id             TEXT    NOT NULL DEFAULT '',

    count               INT         NOT NULL DEFAULT 0,
    last_seen           TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (synapse_id, derived_type, derived_domain, contributor_sig, rule_id)
);

-- Maps to: MotifInstance in MotifStats.Instances []MotifInstance
CREATE TABLE IF NOT EXISTS motif_instances (
    id                  BIGSERIAL   PRIMARY KEY,
    synapse_id          TEXT        NOT NULL REFERENCES synapse_instances(id),
    derived_type        TEXT        NOT NULL,
    derived_domain      TEXT        NOT NULL,
    contributor_sig     TEXT        NOT NULL,
    rule_id             TEXT        NOT NULL DEFAULT '',

    at                  TIMESTAMPTZ NOT NULL,
    derived_id          UUID        NOT NULL REFERENCES events(id),
    contributor_ids     UUID[]      NOT NULL,

    FOREIGN KEY (synapse_id, derived_type, derived_domain, contributor_sig, rule_id)
        REFERENCES motif_stats(synapse_id, derived_type, derived_domain, contributor_sig, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_motif_instances_key
    ON motif_instances (synapse_id, derived_type, derived_domain, contributor_sig, rule_id);

-- 2d. Lineage stats (multi-hop pattern memory)
-- Maps to: lineageStats map[LineageKey]*LineageStats

CREATE TABLE IF NOT EXISTS lineage_stats (
    synapse_id      TEXT    NOT NULL REFERENCES synapse_instances(id),
    derived_type    TEXT    NOT NULL,
    derived_domain  TEXT    NOT NULL,
    depth           INT     NOT NULL,
    sig             BIGINT  NOT NULL,

    count           INT         NOT NULL DEFAULT 0,
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (synapse_id, derived_type, derived_domain, depth, sig)
);

-- Maps to: LineageStats.RuleCounts map[string]int
CREATE TABLE IF NOT EXISTS lineage_rule_counts (
    synapse_id      TEXT    NOT NULL REFERENCES synapse_instances(id),
    derived_type    TEXT    NOT NULL,
    derived_domain  TEXT    NOT NULL,
    depth           INT     NOT NULL,
    sig             BIGINT  NOT NULL,
    rule_id         TEXT    NOT NULL,

    count           INT     NOT NULL DEFAULT 0,

    PRIMARY KEY (synapse_id, derived_type, derived_domain, depth, sig, rule_id),
    FOREIGN KEY (synapse_id, derived_type, derived_domain, depth, sig)
        REFERENCES lineage_stats(synapse_id, derived_type, derived_domain, depth, sig)
);

-- Maps to: LineageStats.Samples []LineageSample
CREATE TABLE IF NOT EXISTS lineage_samples (
    id              BIGSERIAL   PRIMARY KEY,
    synapse_id      TEXT        NOT NULL REFERENCES synapse_instances(id),
    derived_type    TEXT        NOT NULL,
    derived_domain  TEXT        NOT NULL,
    depth           INT         NOT NULL,
    sig             BIGINT      NOT NULL,

    at              TIMESTAMPTZ NOT NULL,
    rule_id         TEXT        NOT NULL DEFAULT '',
    derived_id      UUID        NOT NULL REFERENCES events(id),

    FOREIGN KEY (synapse_id, derived_type, derived_domain, depth, sig)
        REFERENCES lineage_stats(synapse_id, derived_type, derived_domain, depth, sig)
);

CREATE INDEX IF NOT EXISTS idx_lineage_samples_key
    ON lineage_samples (synapse_id, derived_type, derived_domain, depth, sig);

-- ============================================================
-- 3. RULES & CONFIGURATION
-- ============================================================

-- 3a. Derive event rules
-- Maps to: DeriveEventRule (rules.go)
-- Fields: ID, ActionType, Condition, EventTemplate
--
-- condition_json serialization format for Condition.tokens []specToken:
-- [
--   { "kind": "term", "term": {
--       "kind": "has_peers",              -- termKind: termIsType|termInDomain|termHasChild|termHasDescendants|termHasSiblings|termHasPeers|termHasCousin
--       "event_type": "CpuCritical",      -- specTerm.eventType
--       "domain": "",                      -- specTerm.domain (for termInDomain)
--       "conditions": {                    -- specTerm.cond (Conditions struct)
--           "max_depth": 1,               -- Conditions.MaxDepth
--           "counter": {                   -- Conditions.Counter (*Counter, nullable)
--               "how_many": 2,            -- Counter.HowMany
--               "how_many_or_more": true  -- Counter.HowManyOrMore
--           },
--           "time_window": {              -- Conditions.TimeWindow (*TimeWindow, nullable)
--               "within": 5,             -- TimeWindow.Within
--               "time_unit": "minute"    -- TimeWindow.TimeUnit (year|month|day|hour|minute|second|millisecond|microsecond)
--           },
--           "property_values": {"k": "v"},-- Conditions.PropertyValues map[string]any
--           "of_event_type": ""           -- Conditions.OfEventType
--       }
--   }},
--   { "kind": "op", "op": "or" },         -- opKind: and|or
--   { "kind": "lparen" },
--   { "kind": "rparen" },
--   { "kind": "term", "term": { ... } }
-- ]

CREATE TABLE IF NOT EXISTS rules (
    id              TEXT        PRIMARY KEY,
    synapse_id      TEXT        NOT NULL REFERENCES synapse_instances(id),
    action_type     TEXT        NOT NULL DEFAULT 'DeriveNode',

    -- Serialized Condition token list (see format above)
    condition_json  JSONB       NOT NULL,

    -- EventTemplate fields (from templates.go)
    template_type   TEXT        NOT NULL,
    template_domain TEXT        NOT NULL,
    template_props  JSONB       NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rules_synapse ON rules (synapse_id);

-- Maps to: SynapseRuntime.rulesByType map[EventType][]Rule
-- and RegisterRule(eventType, rule) / RegisterRuleForTypes(eventTypes, rule)
CREATE TABLE IF NOT EXISTS rule_event_types (
    rule_id         TEXT    NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    event_type      TEXT    NOT NULL,

    PRIMARY KEY (rule_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_rule_event_types_by_type ON rule_event_types (event_type);

-- 3b. Pattern watcher configuration
-- Maps to: PatternWatcher (pattern_watcher.go) and PatternConfig
-- Fields: Depth, MinCount, Spec (WatchSpec)
--
-- WatchSpec.DerivedTypes → derived_types TEXT[]  (NULL = watch all)
-- WatchSpec.Domains      → domains TEXT[]        (NULL = watch all)

CREATE TABLE IF NOT EXISTS pattern_watcher_configs (
    id              TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    synapse_id      TEXT        NOT NULL REFERENCES synapse_instances(id),

    depth           INT         NOT NULL,
    min_count       INT         NOT NULL DEFAULT 2,

    derived_types   TEXT[],
    domains         TEXT[],

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pattern_watcher_configs_synapse ON pattern_watcher_configs (synapse_id);

-- 3c. Pattern composition specs
-- Maps to: PatternCompositionSpec (pattern_composition.go)
-- Fields: CompositionID, RequiredPatterns, TimeWindow, MinOccurrences, DerivedEventTemplate

CREATE TABLE IF NOT EXISTS composition_specs (
    composition_id              TEXT        PRIMARY KEY,
    synapse_id                  TEXT        NOT NULL REFERENCES synapse_instances(id),

    -- TimeWindow (nullable = no time constraint)
    -- Maps to: PatternCompositionSpec.TimeWindow *TimeWindow
    time_window_within          INT,
    time_window_unit            TEXT,

    -- DerivedEventTemplate
    -- Maps to: PatternCompositionSpec.DerivedEventTemplate EventTemplate
    derived_template_type       TEXT        NOT NULL,
    derived_template_domain     TEXT        NOT NULL,
    derived_template_props      JSONB       NOT NULL DEFAULT '{}',

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_composition_specs_synapse ON composition_specs (synapse_id);

-- Maps to: PatternCompositionSpec.RequiredPatterns map[PatternIdentifier]struct{}
--      and PatternCompositionSpec.MinOccurrences   map[PatternIdentifier]int
-- PatternIdentifier = { EventType, EventDomain }
CREATE TABLE IF NOT EXISTS composition_required_patterns (
    composition_id  TEXT    NOT NULL REFERENCES composition_specs(composition_id) ON DELETE CASCADE,
    event_type      TEXT    NOT NULL,
    event_domain    TEXT    NOT NULL,
    min_occurrences INT     NOT NULL DEFAULT 1,

    PRIMARY KEY (composition_id, event_type, event_domain)
);

-- 3d. Pattern watcher → composition watcher wiring
-- Maps to: CompositePatternListener.watchers []*PatternCompositionWatcher
-- Records which pattern watchers feed which composition watchers.
-- On restore, this wiring is reconstructed to connect the observer pipeline.

CREATE TABLE IF NOT EXISTS watcher_composition_links (
    pattern_watcher_id  TEXT    NOT NULL REFERENCES pattern_watcher_configs(id) ON DELETE CASCADE,
    composition_id      TEXT    NOT NULL REFERENCES composition_specs(composition_id) ON DELETE CASCADE,

    PRIMARY KEY (pattern_watcher_id, composition_id)
);

-- ============================================================
-- 4. AUDIT TABLES (useful for debugging and replay)
-- ============================================================

-- Maps to: PatternMatch (pattern_watcher.go)
-- Logged when PatternWatcher fires OnPatternRepeated
CREATE TABLE IF NOT EXISTS pattern_match_log (
    id              BIGSERIAL   PRIMARY KEY,
    synapse_id      TEXT        NOT NULL REFERENCES synapse_instances(id),

    -- LineageKey fields
    derived_type    TEXT        NOT NULL,
    derived_domain  TEXT        NOT NULL,
    depth           INT         NOT NULL,
    sig             BIGINT      NOT NULL,

    -- PatternMatch fields
    occurrence      INT         NOT NULL,
    at              TIMESTAMPTZ NOT NULL,
    derived_id      UUID        NOT NULL REFERENCES events(id),
    rule_id         TEXT        NOT NULL DEFAULT '',
    contributor_ids UUID[]      NOT NULL,
    anchor_id       UUID
);

CREATE INDEX IF NOT EXISTS idx_pattern_match_log_synapse ON pattern_match_log (synapse_id);
CREATE INDEX IF NOT EXISTS idx_pattern_match_log_type    ON pattern_match_log (derived_type, derived_domain);
CREATE INDEX IF NOT EXISTS idx_pattern_match_log_time    ON pattern_match_log (at);

-- Maps to: PatternCompositionMatch (pattern_composition.go)
-- Logged when PatternCompositionWatcher fires OnCompositionRecognized
CREATE TABLE IF NOT EXISTS composition_match_log (
    id               BIGSERIAL   PRIMARY KEY,
    synapse_id       TEXT        NOT NULL REFERENCES synapse_instances(id),
    composition_id   TEXT        NOT NULL REFERENCES composition_specs(composition_id),
    recognized_at    TIMESTAMPTZ NOT NULL,
    derived_event_id UUID        REFERENCES events(id),

    -- The individual PatternMatch records that triggered composition
    -- Serialized as JSON array of PatternMatch objects
    patterns         JSONB       NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_composition_match_log_synapse ON composition_match_log (synapse_id);
CREATE INDEX IF NOT EXISTS idx_composition_match_log_comp    ON composition_match_log (composition_id);
CREATE INDEX IF NOT EXISTS idx_composition_match_log_time    ON composition_match_log (recognized_at);
