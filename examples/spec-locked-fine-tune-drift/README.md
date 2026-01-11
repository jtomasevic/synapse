
# Spec-locked fine-tune drift → Release Gate

## Walkthrough (Customization + Values → Drift Alarm → Research Debug Session):

Consider a spec-locked fine-tuning workflow where the “spec” is not a document, but a deterministic contract enforced by 
checks, evals, and constraints. Each training run produces raw, local facts: dataset slices are modified, provenance flags 
appear, eval regressions are observed, and explicit spec constraints are violated.

SYNAPSE does not wait for perfect evidence. It begins by constructing meaning opportunistically: repeated provenance flags 
form a DataRiskCluster, repeated eval drops form an EvalRegressionCluster, and any alignment between a spec violation and 
either cluster inside a time window promotes a SpecDriftSignal. 
>This is not a verdict — it is a hypothesis, recorded as structure.

As time passes, further signals accumulate. User feedback that contradicts expected behavior, or external concerns reported 
by partners, independently corroborate the drift and promote GeneralizationRisk and then ReleaseRiskHigh. If a training run 
exists in this lineage, SYNAPSE derives GovernanceActionRequired — a concrete semantic statement that this run can no longer 
be treated as routine.

## Flowchart

```mermaid
flowchart TD

  %% =========================
  %% Spec-Locked Fine-Tune Drift -> Release Gate (POC semantics)
  %% =========================

  subgraph L0["Level 0 - Raw events (leaves)"]
    RS["run_started (training)"]
    DPF["dataset_provenance_flag (data)"]
    ER["eval_regression_observed (eval)"]
    SCV["spec_constraint_violation (spec)"]
    UFM["user_feedback_mismatch (product)"]
    ECR["external_concern_reported (external)"]
  end

  subgraph L1["Level 1 - Repetition clusters (peers within windows)"]
    DRC["data_risk_cluster (data)\nDPF repeats: 3 within 24h"]
    ERC["eval_regression_cluster (eval)\nER repeats: 2 within 6h"]
  end

  subgraph L2["Level 2 - Correlated drift (OR-gated peer checks)"]
    SDS["spec_drift_signal (meta)\nFrom anchor in {SCV, ERC, DRC}\nCondition: peers match SCV OR ERC OR DRC (48h)"]
  end

  subgraph L3["Level 3 - Generalization risk (OR-gated peer checks)"]
    GR["generalization_risk (meta)\nFrom anchor in {SDS, UFM}\nCondition: peers match SDS OR UFM (7d)"]
  end

  subgraph L4["Level 4 - Release risk (OR-gated peer checks)"]
    RRH["release_risk_high (governance)\nFrom anchor in {GR, ECR}\nCondition: peers match GR OR ECR (14d)"]
  end

  subgraph L5["Level 5 - Governance escalation (OR-gated peer checks)"]
    GAR["governance_action_required (governance)\nFrom anchor in {RRH, RS}\nCondition: peers match RRH OR RS (21d)"]
  end

  subgraph P["Pattern stabilization + composition"]
    PM_SDS["PatternWatcher: repeats of spec_drift_signal\nMinCount=3, Depth=4"]
    PM_GAR["PatternWatcher: repeats of governance_action_required\nMinCount=3, Depth=4"]
    RGT["release_gate_triggered (governance)\nDerived ONCE by PatternCompositionWatcher\nWindow: 14d"]
  end

  %% --- L1 derivations (true dependencies) ---
  DPF --> DRC
  ER --> ERC

  %% --- L2: rule registered for SCV, ERC, DRC (anchors) ---
  SCV --> SDS
  ERC --> SDS
  DRC --> SDS

  %% --- L3: rule registered for SDS, UFM (anchors) ---
  SDS --> GR
  UFM --> GR

  %% --- L4: rule registered for GR, ECR (anchors) ---
  GR --> RRH
  ECR --> RRH

  %% --- L5: rule registered for RRH, RS (anchors) ---
  RRH --> GAR
  RS --> GAR

  %% --- Pattern stabilization ---
  SDS --> PM_SDS
  GAR --> PM_GAR
  PM_SDS --> RGT
  PM_GAR --> RGT

```