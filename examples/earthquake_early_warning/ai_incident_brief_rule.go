package earthquake_early_warning

import (
	"encoding/json"
	. "github.com/jtomasevic/synapse/pkg/event_network"
	"strings"
	"time"
)

const (
	// AI consumer output (derived via a Rule reacting to PatternComposition)
	AIIncidentBrief = "ai_incident_brief"

	// domains
	NaturalDisasterWarningSystem = "natural_disaster_warning"
	Geology                      = "geology"
	AnimalObservation            = "animal_observation"
	// events: geology
	MinorTremors = "minor_tremors"
	// derive event:
	HighFrequencyOfMinorTremors = "high_frequency_of_minor_tremors"
	// events: AnimalObservation
	ZebrasMigration     = "zebras_migration"
	UnusualBirdBehavior = "unusual_bird_behavior"
	// derive event
	MultipleAnimalUnexpectedBehavior = "multiple_animal_unexpected_behavior"

	// cross domain derived event (produced by PatternComposition, not a manual rule)
	PotentialNaturalCatastrophic = "potential_natural_catastrophic"
)

/*
Local "AI consumer" rule

Reacts to the composition-derived PotentialNaturalCatastrophic event.
The fact that this event exists means Synapse's PatternComposition engine
confirmed that BOTH domain patterns (tremor bursts + animal anomalies)
were repeatedly observed within the configured time window.

The consumer extracts evidence from the derivation graph and emits a
deterministic incident brief JSON.
*/

type AIIncidentBriefRule struct {
	id  string
	net EventNetwork

	lastTemplate EventTemplate
}

func NewAIIncidentBriefRule(id string) *AIIncidentBriefRule {
	return &AIIncidentBriefRule{id: id}
}

func (r *AIIncidentBriefRule) BindNetwork(network EventNetwork) { r.net = network }
func (r *AIIncidentBriefRule) GetActionType() ActionType        { return DeriveNode }
func (r *AIIncidentBriefRule) GetID() string                    { return r.id }

func (r *AIIncidentBriefRule) GetActionTemplate() EventTemplate {
	return r.lastTemplate
}

type IncidentBrief struct {
	Severity           string   `json:"severity"`
	Confidence         float64  `json:"confidence"`
	Summary            string   `json:"summary"`
	LikelyCauses       []string `json:"likely_causes"`
	RecommendedActions []string `json:"recommended_actions"`
	SignalsSample      []string `json:"signals_sample"`
	GeneratedAt        string   `json:"generated_at"`
	Engine             string   `json:"engine"`
}

func (r *AIIncidentBriefRule) Process(event Event) (bool, []Event, error) {
	if event.EventType != PotentialNaturalCatastrophic || r.net == nil {
		return false, nil, ErrNotSatisfied
	}

	// The PatternComposition engine already confirmed that both domain patterns
	// (tremor bursts + animal anomalies) co-occurred and repeated above threshold.
	// High confidence by construction.
	evidence := r.collectEvidence(event.ID, 10)

	brief := IncidentBrief{
		Severity:   "sev1",
		Confidence: 0.95,
		Summary: "Cross-domain PatternComposition confirmed: repeated tremor bursts (geology) " +
			"and repeated animal anomalies (observation) co-occurring within time window. " +
			"Derived by Synapse PatternComposition engine.",
		LikelyCauses: []string{
			"microtremor burst near active fault line (pattern repeated ≥ threshold)",
			"precursor behavior changes correlated in time — birds / herd movement (pattern repeated ≥ threshold)",
			"cross-domain composition of both patterns satisfied simultaneously",
		},
		RecommendedActions: []string{
			"verify seismic station health and increase monitoring frequency",
			"notify incident commander; activate comms draft and field checks",
			"cross-check with independent sensors and geological advisories",
		},
		SignalsSample: evidence,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Engine:       "pattern_composition_consumer_v1",
	}

	b, _ := json.MarshalIndent(brief, "", "  ")
	out := strings.TrimSpace(string(b))

	r.lastTemplate = EventTemplate{
		EventType:   AIIncidentBrief,
		EventDomain: NaturalDisasterWarningSystem,
		EventProps: map[string]any{
			"brief_json": out,
			"source":     "local_ml_consumer_reading_synapse_composition",
		},
	}

	return true, nil, nil
}

func (r *AIIncidentBriefRule) collectEvidence(anchor EventID, max int) []string {
	children, err := r.net.Children(anchor)
	if err != nil {
		return []string{"No raw evidence extracted from graph."}
	}

	evidence := make([]string, 0, max)
	addRaw := func(e Event) {
		if raw, ok := e.Properties["raw"].(string); ok {
			raw = strings.TrimSpace(raw)
			if raw != "" {
				evidence = append(evidence, raw)
			}
		}
	}

	for _, c := range children {
		addRaw(c)
		desc, _ := r.net.Descendants(c.ID, 4)
		for _, d := range desc {
			if d.EventType == MinorTremors || d.EventType == UnusualBirdBehavior || d.EventType == ZebrasMigration {
				addRaw(d)
			}
			if len(evidence) >= max {
				break
			}
		}
		if len(evidence) >= max {
			break
		}
	}

	if len(evidence) == 0 {
		return []string{"No raw evidence extracted from graph."}
	}
	return evidence
}
