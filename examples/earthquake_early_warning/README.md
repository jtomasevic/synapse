# Earthquake Early Warning – Layered ML → Synapse → ML Demo (Go-only)

This example demonstrates a **layered architecture** where:

1) **Layer 0 (Local ML Producer)** turns raw, noisy text into typed events  
2) **Synapse (Derivation Engine)** stabilizes meaning across time + domains through multi-layer derived events  
3) **Layer 2 (Local ML Consumer)** reads Synapse-derived signals and produces a machine-readable **incident brief**

Everything is **Go-only**, deterministic, and requires **no API keys**, no external services, and no storage.

---

## Story: “Earthquake Early Warning from weak signals”

We ingest hundreds of weak, ambiguous observations from two domains:

- **Geology**: micro-tremor sensor notes (often noisy)
- **Animal Observation**: ranger notes about unusual bird behavior / zebra movement anomalies

Individually, these are weak signals. The demo shows how Synapse turns them into **stable, actionable meaning** using multi-step derivation logic.

---

## Architecture Overview

### Layer 0 — Local ML Producer (TF-IDF + cosine)
**Goal:** convert raw text into structured base events.

- Input: text notes (simulated but realistic)
- Model: **TF-IDF** sparse vectors + **cosine similarity**
- Output: base events with types/domains:
  - `minor_tremors` (geology)
  - `unusual_bird_behavior` (animal_observation)
  - `zebras_migration` (animal_observation)

Each ingested event includes:
- `raw`: original note text
- `sim`: similarity score to the prototype class
- `ml`: `"tfidf+cosine"`

This layer exists to show: **ML produces signals**, Synapse doesn’t need raw text parsing rules.

---

### Synapse — Semantic Stabilization / Derivation Engine
**Goal:** derive higher-level meaning using temporal + cross-domain rules.

Synapse ingests base events and applies rules that produce a **multi-layer derivation ladder**:

#### Derived Layer 1 (within domain)
- From geology:
  - `high_frequency_of_minor_tremors`  
    Trigger: many `minor_tremors` within a window
- From animal observation:
  - `multiple_animal_unexpected_behavior`  
    Trigger: repeated `unusual_bird_behavior` or `zebras_migration` within a window

#### Derived Layer 2 (cross-domain stabilization)
- `potential_natural_catastrophic`  
  Trigger: co-occurrence of derived L1 signals across domains

#### Derived Layer 3 (meta escalation)
- `crisis_protocol_activated`  
  Trigger: repeated `potential_natural_catastrophic` within a broader window

This ladder demonstrates that Synapse can **stabilize meaning**:
- from noisy events
- across time
- across domains
- and escalate when patterns repeat

---

### Layer 2 — Local ML Consumer (reads Synapse outputs)
**Goal:** turn stabilized Synapse signals into an actionable artifact.

When Synapse derives `crisis_protocol_activated`, a consumer rule:
- reads evidence from the derivation graph (children + descendants)
- computes a deterministic severity/confidence score
- emits a derived event:
  - `ai_incident_brief` with `brief_json` (pretty-printed JSON)

This layer proves: **Synapse outputs become inputs for the next “AI layer.”**

---

## Event Flow Summary

Raw text notes (320)
→ Local ML classify (TF-IDF + cosine)
→ base events: tremors / birds / zebras
→ Synapse derivations:
   L1: tremor burst + animal anomaly
→ L2: cross-domain catastrophic candidate
→ L3: crisis protocol activation
→ Local ML consumer:
   incident brief JSON

---

## Files

- `example_test.go`
  - sets up Synapse rules (L1 → L2 → L3)
  - generates 320 raw notes
  - runs Layer 0 local ML classification and ingests events
  - asserts derived layers exist
  - prints the final incident brief JSON

- `local_ml.go`
  - TF-IDF vectorizer (sparse vectors)
  - cosine similarity (sparse, normalized)

- `ai_incident_brief_rule.go`
  - local “AI consumer” rule that reacts to `crisis_protocol_activated`
  - reads evidence from the event graph
  - emits `ai_incident_brief` with deterministic JSON

---

## How to Run

From the repo root:

```bash
go test ./... -run Test_Dance_EarthquakeEarlyWarning_AI_Synapse_AI -v
````

You should see output similar to:

* counts of derived events (implicit via assertions)
* printed JSON:

```json
{
  "severity": "sev2",
  "confidence": 0.63,
  "summary": "...",
  "likely_causes": [...],
  "recommended_actions": [...],
  "evidence_counts": {...},
  "signals_sample": [...],
  "generated_at": "...",
  "engine": "local_ml_consumer_v1"
}
```

---

## Why this design is interesting

### 1) ML as a producer, not “the brain”

Layer 0 ML simply turns unstructured inputs into typed events. It doesn’t “own” the system.

### 2) Synapse as semantic stabilization

Synapse provides:

* temporal windows
* cross-domain correlation
* multi-step derivation
* evidence lineage (graph)

This is the key: stable meaning emerges from **composition**, not from any single model call.

### 3) Downstream automation is clean

The consumer doesn’t need raw notes and heuristics; it consumes derived meaning:

* `crisis_protocol_activated`
  …and can produce consistent artifacts.

