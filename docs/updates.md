Below is a **formal, spec-grade section** you can drop directly into your documentation.
It is written to be precise, defensible, and aligned with your bottom-up / semantic-derivation model.

---

## Semantic Relationships

SYNAPSE models **semantic derivation**, not causal explanation.
All relationships between events are defined relative to how meaning is *constructed* from events, rather than how events occurred in time.

This section formally defines the semantic meaning of core graph relationships.

---

### 1. Fundamental Directionality

SYNAPSE operates under a **bottom-up derivation model**:

* **Leaf events** represent externally observed facts.
* **Derived events** represent semantic aggregation or interpretation of other events.
* Edges are directed from **contributors → derived event**.

As a result:

* Semantic derivation flows **upward**
* Explanatory evidence flows **downward**
* These directions are intentionally asymmetric

This asymmetry is fundamental and enforced across all relationship semantics.

---

### 2. Children

**Definition**

> The *children* of an event are the events that directly contributed to its derivation.

**Semantic meaning**

* Children are **semantic inputs**
* They explain *what evidence produced this event*
* Children exist only for derived events

**Formal rule**

An event `A` has child `B` if and only if there exists an edge:

```
B → A
```

Children are never derived from the anchor; they are always explanatory inputs.

---

### 3. Parents

**Definition**

> The *parents* of an event are derived events that were created using this event as one of their inputs.

**Semantic meaning**

* Parents represent **higher-level meaning**
* Parents summarize or interpret the anchor event
* Parents may themselves participate in further derivations

**Formal rule**

An event `A` has parent `P` if and only if there exists an edge:

```
A → P
```

---

### 4. Descendants

**Definition**

> The *descendants* of an event are events that were **derived from it**, directly or indirectly.

**Semantic meaning**

* Descendants represent **semantic consequences**
* They are higher-level interpretations that build upon the anchor event
* Descendants flow **upward** through derivation

**Important constraint**

Descendants do **not** include the events used to derive the anchor.
That explanatory subgraph is accessed via **Children**, not Descendants.

**Formal rule**

Descendants of `A` are obtained by recursively traversing **Parents**:

```
A → P₁ → P₂ → … → Pₙ
```

Depth is measured in derivation levels.

---

### 5. Siblings

**Definition**

> Siblings are events that are semantically related at the same derivation level.

SYNAPSE defines siblings using a **two-tier semantic rule**.

---

#### 5.1 Derivation-based siblings (primary)

If an event has one or more parents:

> Siblings are events that share at least one common derived parent.

**Semantic meaning**

* Events are siblings because they **co-contributed to the same semantic interpretation**
* This is the preferred and dominant sibling relationship

**Formal rule**

Events `A` and `B` are siblings if there exists a derived event `P` such that:

```
A → P AND B → P
```

---

#### 5.2 Structural siblings (fallback)

If an event has **no parents**:

> Siblings are events of the same type that also have no parents.

**Semantic meaning**

* These events represent **unaggregated peers**
* They occupy the same semantic level
* They are siblings by **structural equivalence**, not derivation

This fallback preserves meaningful grouping in early or sparse graphs.

**Formal rule**

Events `A` and `B` are siblings if:

* `A.EventType == B.EventType`
* `A` has no parents
* `B` has no parents
* `A ≠ B`

---

### 6. Cousins

**Definition**

> Cousins are events that are semantically related through **shared derivational ancestry**, but are neither direct siblings nor direct contributors.

**Semantic meaning**

* Cousins express **contextual relatedness**
* They capture correlations that emerge only at higher semantic levels
* Cousins are not inputs, and not direct peers

**Formal intuition**

Two events are cousins if:

* They do not share a direct parent
* But their parents (or grandparents, up to a defined depth) converge

Cousins are useful for:

* correlation analysis
* anomaly grouping
* cross-domain pattern recognition

---

### 7. Design Invariants

The following invariants are enforced throughout SYNAPSE:

1. **Derivation is directional and irreversible**
2. **Children explain; parents interpret**
3. **Descendants are semantic consequences, not evidence**
4. **Siblinghood prefers derivation over structure**
5. **Fallback semantics apply only when derivation is absent**

These rules ensure that:

* semantic meaning is preserved,
* expressions remain interpretable,
* and derived events never masquerade as raw evidence.

---

### 8. Why this matters

This semantic relationship model allows SYNAPSE to:

* distinguish explanation from meaning,
* avoid causal ambiguity,
* support incremental, localized reasoning,
* and remain explainable even in deep derivation graphs.

Most importantly:

> **SYNAPSE does not ask “why did this happen?”
> It asks “how did this meaning emerge?”**

That distinction underpins the entire platform.
