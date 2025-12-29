package event_network

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSynapseRuntime_Ingest(t *testing.T) {
	synapse := NewSynapse()

	synapse.RegisterRule(CpuStatusChanged, NewDeriveEventRule(
		NewCondition().HasSiblings(CpuStatusChanged, Conditions{
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

	synapse.RegisterRule(MemoryStatusChanged, NewDeriveEventRule(
		NewCondition().HasSiblings(MemoryStatusChanged, Conditions{
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

	synapse.RegisterRule(MemoryCritical, NewDeriveEventRule(
		NewCondition().HasSiblings(CpuCritical,
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

	synapse.RegisterRule(CpuCritical, NewDeriveEventRule(
		NewCondition().HasSiblings(MemoryCritical, Conditions{
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
	require.NoError(t, err)
	_, err = synapse.Ingest(createCpuStatusChangedEvent(95, "critical"))
	require.NoError(t, err)
	_, err = synapse.Ingest(createCpuStatusChangedEvent(91, "critical"))
	require.NoError(t, err)

	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70, "critical"))
	require.NoError(t, err)
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(75, "critical"))
	require.NoError(t, err)
	// break down fot eaisier dubugging.
	lastEvent := createMemoryStatusChangedEvent(80, "critical")
	_, err = synapse.Ingest(lastEvent)
	require.NoError(t, err)

	PrintEventGraph(synapse.GetNetwork())
}
