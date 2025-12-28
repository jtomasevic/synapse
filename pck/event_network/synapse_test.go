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

	err := synapse.Ingest(createCpuStatusChangedEvent(92, "critical"))
	require.NoError(t, err)
	err = synapse.Ingest(createCpuStatusChangedEvent(95, "critical"))
	require.NoError(t, err)
	err = synapse.Ingest(createCpuStatusChangedEvent(91, "critical"))
	require.NoError(t, err)

	//err = synapse.Ingest(createCpuStatusChangedEvent(90, "critical"))
	//require.NoError(t, err)
	//err = synapse.Ingest(createCpuStatusChangedEvent(92, "critical"))
	//require.NoError(t, err)
	//err = synapse.Ingest(createCpuStatusChangedEvent(89, "critical"))
	//require.NoError(t, err)

	err = synapse.Ingest(createMemoryStatusChangedEvent(70, "critical"))
	require.NoError(t, err)
	err = synapse.Ingest(createMemoryStatusChangedEvent(75, "critical"))
	require.NoError(t, err)
	err = synapse.Ingest(createMemoryStatusChangedEvent(80, "critical"))
	require.NoError(t, err)

	PrintEventGraph(synapse.GetNetwork())
}
