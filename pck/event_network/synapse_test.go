package event_network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WithComplexRules(t *testing.T) {
	synapse := NewSynapse(nil)

	synapse.RegisterRule(CpuStatusChanged, NewDeriveEventRule("cpu_status_critical",
		NewCondition().HasPeers(CpuStatusChanged, Conditions{
			Counter: &Counter{
				HowMany:       2,
				HowManyOrMore: false,
			},
		}), EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 3,
			},
		},
	))

	synapse.RegisterRule(MemoryStatusChanged, NewDeriveEventRule("node_critical1",
		NewCondition().HasPeers(MemoryStatusChanged, Conditions{
			Counter: &Counter{
				HowMany:       2,
				HowManyOrMore: false,
			},
		}), EventTemplate{
			EventType:   MemoryCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 3,
			},
		},
	))

	synapse.RegisterRule(MemoryCritical, NewDeriveEventRule("node_critical2",
		NewCondition().HasPeers(CpuCritical,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
			}), EventTemplate{
			EventType:   ServerNodeChangeStatus,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	))

	synapse.RegisterRule(CpuCritical, NewDeriveEventRule("node_critical2",
		NewCondition().HasPeers(MemoryCritical,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
			}), EventTemplate{
			EventType:   ServerNodeChangeStatus,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	))

	_, err := synapse.Ingest(createCpuStatusChangedEvent(92, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.1, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.2, "critical"))

	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.1, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.2, "critical"))

	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.3, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.4, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.2, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.5, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(80.2, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(75.2, "critical"))

	require.NoError(t, err)

	PrintEventGraph(synapse.GetNetwork())
}
