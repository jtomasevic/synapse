# Earthquake Early Warning — Patterns, Composition & ML Demo (Go-only)

This example demonstrates a **layered architecture** where:

1) **Layer 0 (Local ML Producer)** turns raw, noisy text into typed events  
2) **Synapse (Pattern & Composition Engine)** stabilizes meaning using within-domain rules, pattern repetition detection, and cross-domain composition  
3) **AI Consumer** reads Synapse composition-derived signals and produces a machine-readable **incident brief**

Everything is **Go-only**, deterministic, and requires **no API keys**, no external services, and no storage.

---

## Story: "Earthquake Early Warning from weak signals"

We ingest hundreds of weak, ambiguous observations from two domains:

- **Geology**: micro-tremor sensor notes (often noisy)
- **Animal Observation**: ranger notes about unusual bird behavior / zebra movement anomalies

Individually, these are weak signals. The demo shows how Synapse turns them into **stable, actionable meaning** using patterns and cross-domain composition.

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

---

### Synapse — Pattern Recognition & Cross-Domain Composition

**Goal:** derive higher-level meaning using temporal rules, pattern repetition, and cross-domain composition.

#### L1 Rules (within-domain derivation)
- From geology:
  - `high_frequency_of_minor_tremors`  
    Rule: `minor_tremors` has 5+ peers within 8 hours
- From animal observation:
  - `multiple_animal_unexpected_behavior`  
    Rule: `unusual_bird_behavior` or `zebras_migration` has peers within 8 hours

#### Pattern Watchers (repetition detection)
- Watch for `multiple_animal_unexpected_behavior` motif repeating ≥ 3 times
- Watch for `high_frequency_of_minor_tremors` motif repeating ≥ 3 times

When a motif repeats enough, the PatternWatcher emits a `PatternMatch`.

#### Pattern Composition (cross-domain trigger)
- **`PatternCompositionSpec`** requires BOTH domain patterns to be recognized within an 8-hour window
- When satisfied → derives `potential_natural_catastrophic`

This replaces manual L2/L3 rules with the declarative composition system:
- No hand-coded cross-domain join rules
- No manual descendant traversal or event counting
- The composition engine handles all of it

---

### AI Consumer — reads composition output

When Synapse derives `potential_natural_catastrophic` (via composition), the AI consumer rule:
- knows both domain patterns are confirmed (the composition guarantees it)
- collects evidence from the derivation graph
- emits `ai_incident_brief` with a deterministic JSON brief

---

## Event Flow Summary

```
Raw text notes (320)
  → Local ML classify (TF-IDF + cosine)
  → base events: tremors / birds / zebras

  → L1 Rules: within-domain derivation
      high_frequency_of_minor_tremors
      multiple_animal_unexpected_behavior

  → Pattern Watchers: detect motif repetition (MinCount=3)
      PatternMatch(animal) + PatternMatch(tremors)

  → Pattern Composition: cross-domain co-occurrence
      → derives potential_natural_catastrophic

  → AI Consumer: reads composition output
      → ai_incident_brief JSON
```

---

## Files

- `example_test.go`
  - sets up PatternConfig watchers (animal + tremors)
  - sets up PatternCompositionSpec (cross-domain trigger)
  - registers L1 rules (only 2 rules!)
  - generates 320 raw notes via Layer 0 ML classification
  - asserts patterns are recognized and composition fires
  - prints the final incident brief JSON

- `local_ml.go`
  - TF-IDF vectorizer (sparse vectors)
  - cosine similarity (sparse, normalized)

- `ai_incident_brief_rule.go`
  - local "AI consumer" rule that reacts to `potential_natural_catastrophic`
  - emits `ai_incident_brief` with deterministic JSON
  - no manual event counting — composition already validated the signal

---

## How to Run

From the repo root:

```bash
go test ./examples/earthquake_early_warning/ -run Test_Dance_EarthquakeEarlyWarning_AI_Synapse_AI -v
```

---

## Why this design is interesting

### 1) Patterns replace manual counting
Instead of writing rules that traverse the graph and count events by type,
`PatternConfig` detects when a derivation motif repeats. The count threshold
is declared, not coded.

### 2) Composition replaces cross-domain join rules
Instead of writing an L2 rule that checks for peers across domains,
`PatternCompositionSpec` declares which patterns must co-occur. Synapse
handles the temporal correlation automatically.

### 3) ML as a producer, not "the brain"
Layer 0 ML simply turns unstructured inputs into typed events. It doesn't "own" the system.

### 4) Downstream automation is clean
The AI consumer doesn't need to count descendants or parse the graph structure.
It receives `potential_natural_catastrophic` — which by definition means both
patterns are confirmed — and produces a clean artifact.
