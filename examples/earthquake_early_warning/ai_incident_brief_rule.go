package earthquake_early_warning

import (
	"encoding/json"
	. "github.com/jtomasevic/synapse/pkg/event_network"
	"strings"
	"time"
)

const (
	// Layer 3 derived
	CrisisProtocolActivated = "crisis_protocol_activated"

	// AI consumer output (derived via a Rule)
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
	RawAnimalBehavior   = "raw_animal_behavior"
	// derive event
	MultipleAnimalUnexpectedBehavior = "multiple_animal_unexpected_behavior"

	// cross domain derived event
	PotentialNaturalCatastrophic = "potential_natural_catastrophic"
)

/*
Local “AI consumer” rule

Reads Synapse outputs (derived CrisisProtocolActivated), inspects the derivation graph,
and emits a deterministic incident brief JSON.

This keeps the full “Layer 0 -> Synapse (3+ derived layers) -> AI consumer reading Synapse outputs” story,
while remaining fully local and shareable (no API keys, no external services).
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
	Severity           string         `json:"severity"`
	Confidence         float64        `json:"confidence"`
	Summary            string         `json:"summary"`
	LikelyCauses       []string       `json:"likely_causes"`
	RecommendedActions []string       `json:"recommended_actions"`
	EvidenceCounts     map[string]int `json:"evidence_counts"`
	SignalsSample      []string       `json:"signals_sample"`
	GeneratedAt        string         `json:"generated_at"`
	Engine             string         `json:"engine"`
}

func (r *AIIncidentBriefRule) Process(event Event) (bool, []Event, error) {
	if event.EventType != CrisisProtocolActivated || r.net == nil {
		return false, nil, ErrNotSatisfied
	}

	evidence := r.collectEvidence(event.ID, 10)

	descAll, _ := r.net.Descendants(event.ID, 6)
	counts := map[EventType]int{
		MinorTremors:                     0,
		UnusualBirdBehavior:              0,
		ZebrasMigration:                  0,
		HighFrequencyOfMinorTremors:      0,
		MultipleAnimalUnexpectedBehavior: 0,
		PotentialNaturalCatastrophic:     0,
	}
	for _, d := range descAll {
		if _, ok := counts[d.EventType]; ok {
			counts[d.EventType]++
		}
	}

	// deterministic scoring ("ML-ish")
	score := 0.0
	score += 0.10 * float64(minInt(counts[HighFrequencyOfMinorTremors], 5))      // up to +0.5
	score += 0.10 * float64(minInt(counts[MultipleAnimalUnexpectedBehavior], 5)) // up to +0.5
	score += 0.20 * float64(minInt(counts[PotentialNaturalCatastrophic], 5))     // up to +1.0

	conf := score / 2.0
	if conf > 1.0 {
		conf = 1.0
	}

	severity := "sev3"
	switch {
	case conf >= 0.75:
		severity = "sev1"
	case conf >= 0.45:
		severity = "sev2"
	}

	brief := IncidentBrief{
		Severity:   severity,
		Confidence: round2(conf),
		Summary: "Cross-domain weak signals stabilized into a repeatable catastrophic-risk motif " +
			"(tremor bursts + animal anomalies). Crisis protocol triggered.",
		LikelyCauses: []string{
			"microtremor burst near active fault line",
			"precursor behavior changes correlated in time (birds / herd movement)",
		},
		RecommendedActions: []string{
			"verify seismic station health and increase monitoring frequency",
			"notify incident commander; activate comms draft and field checks",
			"cross-check with independent sensors and geological advisories",
		},
		EvidenceCounts: map[string]int{
			"minor_tremors": counts[MinorTremors],
			"unusual_bird":  counts[UnusualBirdBehavior],
			"zebras":        counts[ZebrasMigration],
			"hf_tremors":    counts[HighFrequencyOfMinorTremors],
			"animal_unexp":  counts[MultipleAnimalUnexpectedBehavior],
			"catastrophic":  counts[PotentialNaturalCatastrophic],
		},
		SignalsSample: evidence,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Engine:        "local_ml_consumer_v1",
	}

	b, _ := json.MarshalIndent(brief, "", "  ")
	out := strings.TrimSpace(string(b))

	r.lastTemplate = EventTemplate{
		EventType:   AIIncidentBrief,
		EventDomain: NaturalDisasterWarningSystem,
		EventProps: map[string]any{
			"brief_json": out,
			"source":     "local_ml_consumer_reading_synapse",
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func round2(x float64) float64 {
	return float64(int(x*100+0.5)) / 100
}
