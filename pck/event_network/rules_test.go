package event_network

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeriveEventRule_Process(t *testing.T) {
	network := NewInMemoryEventNetwork()
	// Definition or ruls
	rule := NewDeriveEventRule(NewCondition().HasSiblings(CpuStatusChanged, Conditions{
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
	})
	rule.BindNetwork(network)
	// Adding nodes and evaluate rule
	eventID1, _ := addCpuStatusChangedEvent(network, 98.3, "critical")
	event1, _ := network.GetByID(eventID1)
	err := rule.Process(event1)
	require.Error(t, err)

	eventID2, _ := addCpuStatusChangedEvent(network, 95.2, "critical")
	event2, _ := network.GetByID(eventID2)
	err = rule.Process(event2)
	require.Error(t, err)

	eventID3, err := addCpuStatusChangedEvent(network, 91.11, "critical")
	event3, _ := network.GetByID(eventID3)
	err = rule.Process(event3)
	require.NoError(t, err) // condition met

}
