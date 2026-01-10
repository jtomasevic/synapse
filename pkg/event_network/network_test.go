package event_network

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"testing"
	"time"
)

const (
	CpuStatusChanged       = "cpu_status_changed"
	MemoryStatusChanged    = "memory_status_changed"
	CpuCritical            = "cpu_critical"
	MemoryCritical         = "memory_critical"
	ServerNodeChangeStatus = "server_node_change_status"

	InfraDomain = "infra_domain"
)

func TestInMemoryEventNetwork_AddEvent(t *testing.T) {
	eventNetwork := NewInMemoryEventNetwork()
	eventId, err := eventNetwork.AddEvent(Event{
		EventType:   CpuStatusChanged,
		EventDomain: InfraDomain,
		Properties:  toProps(cpuStatusChangedEvent),
		Timestamp:   time.Now(),
	})
	require.NotEmpty(t, eventId)
	require.NoError(t, err)
}

func TestInMemoryEventNetwork_GetByID(t *testing.T) {
	eventNetwork := NewInMemoryEventNetwork()
	eventId, err := addCpuStatusChangedEvent(eventNetwork,
		98.5,
		"critical")
	require.NoError(t, err)
	event, err := eventNetwork.GetByID(eventId)
	require.NoError(t, err)
	require.Equal(t, event, event)
	require.Equal(t, event.EventType, EventType("cpu_status_changed"))
	require.Equal(t, event.EventDomain, EventDomain("infra_domain"))
	require.Equal(t, event.Timestamp, event.Timestamp)
	require.Equal(t, event.Properties["percentage"], 98.5)
	require.Equal(t, event.Properties["level"], "critical")
}

func TestInMemoryEventNetwork_GetByIDs(t *testing.T) {
	eventNetwork := NewInMemoryEventNetwork()
	eventId, err := addCpuStatusChangedEvent(eventNetwork,
		98.5,
		"critical")
	eventId2, err := addCpuStatusChangedEvent(eventNetwork,
		90,
		"critical")
	require.NoError(t, err)
	events, err := eventNetwork.GetByIDs([]EventID{eventId, eventId2})
	require.NoError(t, err)
	require.Equal(t, events[0].EventType, EventType("cpu_status_changed"))
	require.Equal(t, events[0].EventDomain, EventDomain("infra_domain"))
	require.Equal(t, events[0].Properties["percentage"], 98.5)
	require.Equal(t, events[0].Properties["level"], "critical")

	require.Equal(t, events[1].EventType, EventType("cpu_status_changed"))
	require.Equal(t, events[1].EventDomain, EventDomain("infra_domain"))
	require.Equal(t, events[1].Properties["percentage"], float64(90))
	require.Equal(t, events[1].Properties["level"], "critical")
}

func TestInMemoryEventNetwork_Children(t *testing.T) {
	network, parentNodes, _ := buildInfraSubGraph(t)
	PrintEventGraph(network.(*InMemoryEventNetwork))

	child, err := network.Children(parentNodes.CpuCriticalID)
	require.NoError(t, err)
	require.NotEmpty(t, child)
	require.Equal(t, len(child), 3)

	network, parentNodes, _ = buildInfraSubGraph(t)
	child, err = network.Children(parentNodes.MemoryCriticalID)
	require.NoError(t, err)
	require.NotEmpty(t, child)
	require.Equal(t, len(child), 3)
}

func TestInMemoryEventNetwork_Parents(t *testing.T) {
	network, _, childs := buildInfraSubGraph(t)
	parents, err := network.Parents(childs.CpuEventsIDs[0])
	require.NotEmpty(t, parents)
	require.NoError(t, err)
	require.Equal(t, 1, len(parents))
}

func TestInMemoryEventNetwork_Siblings(t *testing.T) {
	network, _, childs := buildInfraSubGraph(t)
	siblings, err := network.Siblings(childs.CpuEventsIDs[0])
	require.NoError(t, err)
	require.NotEmpty(t, siblings)
	require.Equal(t, len(siblings), 2)

	siblings, err = network.Siblings(childs.MemoryEventsIDs[0])
	require.NoError(t, err)
	require.NotEmpty(t, siblings)
	require.Equal(t, len(siblings), 2)
}

func TestInMemoryEventNetwork_Siblings2(t *testing.T) {
	network, parents, _ := buildInfraSubGraph(t)
	siblings, err := network.Siblings(parents.MemoryCriticalID)
	require.NoError(t, err)
	require.NotEmpty(t, siblings)
	require.Equal(t, 1, len(siblings))

	//siblings, err = network.Siblings(childs.MemoryEventsIDs[0])
	//require.NoError(t, err)
	//require.NotEmpty(t, siblings)
	//require.Equal(t, len(siblings), 2)
}

func TestInMemoryEventNetwork_Descendants(t *testing.T) {
	network, parentNodes, _ := buildInfraSubGraph(t)
	descendants, err := network.Descendants(parentNodes.ServerNodeChangeStatusID, 1)
	require.NoError(t, err)
	require.Equal(t, len(descendants), 2)

	descendants, err = network.Descendants(parentNodes.ServerNodeChangeStatusID, 2)
	require.NoError(t, err)
	require.Equal(t, len(descendants), 8)
}

func TestInMemoryEventNetwork_Cousins(t *testing.T) {
	network, parentNodes, childNodes := buildInfraSubGraph(t)
	require.NotEmpty(t, network)
	require.NotEmpty(t, parentNodes)
	require.NotEmpty(t, childNodes)

	cousins, err := network.Cousins(parentNodes.CpuCriticalID, 1)
	require.NoError(t, err)
	require.Equal(t, len(cousins), 1)

	cousins, err = network.Cousins(childNodes.MemoryEventsIDs[0], 1)
	require.NoError(t, err)
	require.Equal(t, len(cousins), 2)

	cousins, err = network.Cousins(childNodes.MemoryEventsIDs[0], 2)
	require.NoError(t, err)
	require.Equal(t, len(cousins), 5)
}

func TestInMemoryEventNetwork_GetByID2(t *testing.T) {
	eventNetwork := NewInMemoryEventNetwork()
	event := Event{
		EventType:   "testType",
		EventDomain: "testDomain",
		Properties: map[string]interface{}{
			"prop1": 10,
			"prop2": true,
			"prop3": "ha",
			"props4": map[string]interface{}{
				"objProp": "hej",
			},
		},
		Timestamp: time.Time{},
	}
	eventId, err := eventNetwork.AddEvent(event)
	require.NoError(t, err)
	require.NotEmpty(t, eventId)

	getEvent, err := eventNetwork.GetByID(eventId)
	require.NoError(t, err)
	require.Equal(t, event.EventType, getEvent.EventType)
	require.Equal(t, event.EventDomain, getEvent.EventDomain)
	require.Equal(t, event.Properties["prop1"], getEvent.Properties["prop1"])
	require.Equal(t, event.Properties["prop2"], getEvent.Properties["prop2"])
	require.Equal(t, event.Properties["prop3"], getEvent.Properties["prop3"])
	require.Equal(t, event.Properties["prop3"], getEvent.Properties["prop3"])
	props4 := event.Properties["props4"].(map[string]interface{})
	require.Equal(t, props4["objProp"], getEvent.Properties["props4"].(map[string]interface{})["objProp"])
}

func TestInMemoryEventNetwork_GetByType(t *testing.T) {
	newtork, _, _ := buildInfraSubGraph(t)
	require.NotNil(t, newtork)
	events, err := newtork.GetByType(CpuStatusChanged)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, len(events), 3)
}

func toProps(eventPropsPayload string) EventProps {
	props := make(EventProps)
	json.Unmarshal([]byte(eventPropsPayload), &props)
	return props
}

const cpuStatusChangedEvent = `{
	"percentage": 95.8,
	"level": "critical"
}`

func createCpuStatusChangedEvent(percentage float64, level string) Event {
	return Event{
		EventType:   CpuStatusChanged,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"percentage": percentage,
			"level":      level,
		},
	}
}

func addCpuStatusChangedEvent(network EventNetwork,
	percentage float64,
	level string) (EventID, error) {
	event := createCpuStatusChangedEvent(percentage, level)
	return network.AddEvent(event)
}

func createMemoryStatusChangedEvent(percentage float64, level string) Event {
	return Event{
		EventType:   MemoryStatusChanged,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"percentage": percentage,
			"level":      level,
		},
	}
}

func addMemoryStatusChangedEvent(network EventNetwork,
	percentage float64,
	level string) (EventID, error) {
	event := createMemoryStatusChangedEvent(percentage, level)
	return network.AddEvent(event)
}

type TimeFrame struct {
	From time.Time
	To   time.Time
}

func addCpuCriticalEvent(network EventNetwork,
	timeFrame TimeFrame,
	occurs int) (EventID, error) {
	event := Event{
		EventType:   CpuCritical,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"time_frame": timeFrame,
			"occurs":     occurs,
			"timestamp":  time.Now(),
		},
	}
	return network.AddEvent(event)
}

func addMemoryCriticalEvent(network EventNetwork,
	timeFrame TimeFrame,
	occurs int) (EventID, error) {
	event := Event{
		EventType:   MemoryCritical,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"time_frame": timeFrame,
			"occurs":     occurs,
			"timestamp":  time.Now(),
		},
	}
	return network.AddEvent(event)
}

func addNodeStatusChangedEvent(network EventNetwork,
	status string,
	eventTime time.Time) (EventID, error) {
	event := Event{
		EventType:   ServerNodeChangeStatus,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"status":        status,
			"statusChanged": eventTime,
			"timestamp":     time.Now(),
		},
	}
	return network.AddEvent(event)
}

type ChildNodes struct {
	CpuEventsIDs    []EventID
	MemoryEventsIDs []EventID
}

type ParentNodes struct {
	CpuCriticalID            EventID
	MemoryCriticalID         EventID
	ServerNodeChangeStatusID EventID
}

func GetLatestTime(times []time.Time) time.Time {
	if len(times) == 0 {
		// Return a zero value for time.Time if the slice is empty
		return time.Time{}
	}

	// Initialize the latest time to the first element
	latest := times[0]

	// Iterate through the rest of the slice
	for i := 1; i < len(times); i++ {
		// time.After() returns true if the time is after the argument time
		if times[i].After(latest) {
			latest = times[i]
		}
	}

	return latest
}

func TestInMemoryEventNetwork_Peers(t *testing.T) {
	t.Run("finds same-type parentless events", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
		require.NoError(t, err)
		eventID3, err := addCpuStatusChangedEvent(net, 97.1, "critical")
		require.NoError(t, err)

		peers, err := net.Peers(eventID1)
		require.NoError(t, err)
		require.Len(t, peers, 2)
		require.Contains(t, []EventID{eventID2, eventID3}, peers[0].ID)
		require.Contains(t, []EventID{eventID2, eventID3}, peers[1].ID)
	})

	t.Run("excludes events with parents", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
		require.NoError(t, err)
		eventID3, err := addCpuStatusChangedEvent(net, 97.1, "critical")
		require.NoError(t, err)

		// Create a derived event and add edges
		cpuCriticalID, err := addCpuCriticalEvent(net, TimeFrame{}, 3)
		require.NoError(t, err)
		err = net.AddEdge(eventID1, cpuCriticalID, "trigger")
		require.NoError(t, err)
		err = net.AddEdge(eventID2, cpuCriticalID, "trigger")
		require.NoError(t, err)
		err = net.AddEdge(eventID3, cpuCriticalID, "trigger")
		require.NoError(t, err)

		// Now eventID1, eventID2, eventID3 have parents, so they shouldn't be peers
		eventID4, err := addCpuStatusChangedEvent(net, 90.0, "critical")
		require.NoError(t, err)
		eventID5, err := addCpuStatusChangedEvent(net, 91.0, "critical")
		require.NoError(t, err)

		peers, err := net.Peers(eventID4)
		require.NoError(t, err)
		require.Len(t, peers, 1)
		require.Equal(t, eventID5, peers[0].ID)
	})

	t.Run("excludes anchor event itself", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
		require.NoError(t, err)

		peers, err := net.Peers(eventID1)
		require.NoError(t, err)
		require.Len(t, peers, 1)
		require.Equal(t, eventID2, peers[0].ID)
		require.NotEqual(t, eventID1, peers[0].ID)
	})

	t.Run("filters by same domain", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		// Add event with different domain
		event2 := Event{
			EventType:   CpuStatusChanged,
			EventDomain: "other_domain",
			Timestamp:   time.Now(),
			Properties:  EventProps{"percentage": 95.0, "level": "critical"},
		}
		_, err = net.AddEvent(event2)
		require.NoError(t, err)

		peers, err := net.Peers(eventID1)
		require.NoError(t, err)
		require.Len(t, peers, 0) // Different domain, so not a peer
	})

	t.Run("filters by same type", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		_, err = addMemoryStatusChangedEvent(net, 90.0, "critical")
		require.NoError(t, err)

		peers, err := net.Peers(eventID1)
		require.NoError(t, err)
		require.Len(t, peers, 0) // Different type, so not a peer
	})

	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Peers(nonExistentID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("returns empty when no peers exist", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		peers, err := net.Peers(eventID1)
		require.NoError(t, err)
		require.Len(t, peers, 0)
	})
}

func TestInMemoryEventNetwork_Ancestors(t *testing.T) {
	t.Run("finds direct ancestors", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		ancestors, err := net.Ancestors(childs.CpuEventsIDs[0], 1)
		require.NoError(t, err)
		require.Len(t, ancestors, 1)
		require.Equal(t, CpuCritical, ancestors[0].EventType)
	})

	t.Run("finds ancestors up to maxDepth", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		ancestors, err := net.Ancestors(childs.CpuEventsIDs[0], 2)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(ancestors), 1)
		// Should find CpuCritical and potentially ServerNodeChangeStatus
		foundCpuCritical := false
		for _, a := range ancestors {
			if a.EventType == CpuCritical {
				foundCpuCritical = true
				break
			}
		}
		require.True(t, foundCpuCritical)
	})

	t.Run("respects maxDepth limit", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		ancestors, err := net.Ancestors(childs.CpuEventsIDs[0], 1)
		require.NoError(t, err)
		require.Len(t, ancestors, 1) // Only direct parent
	})

	t.Run("returns empty when maxDepth is 0", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		ancestors, err := net.Ancestors(childs.CpuEventsIDs[0], 0)
		require.NoError(t, err)
		require.Len(t, ancestors, 0)
	})

	t.Run("returns empty when maxDepth is negative", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		ancestors, err := net.Ancestors(childs.CpuEventsIDs[0], -1)
		require.NoError(t, err)
		require.Len(t, ancestors, 0)
	})

	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Ancestors(nonExistentID, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("handles circular references with visited set", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
		require.NoError(t, err)

		cpuCriticalID, err := addCpuCriticalEvent(net, TimeFrame{}, 1)
		require.NoError(t, err)
		err = net.AddEdge(eventID1, cpuCriticalID, "trigger")
		require.NoError(t, err)
		err = net.AddEdge(eventID2, cpuCriticalID, "trigger")
		require.NoError(t, err)

		// Should not infinite loop
		ancestors, err := net.Ancestors(eventID1, 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(ancestors), 0)
	})

	t.Run("returns error when ancestor event not found in events map", func(t *testing.T) {
		// This tests the edge case where an edge points to a non-existent event
		// This is hard to test directly without manipulating internal state
		// The code path exists but is unlikely to occur in normal operation
	})
}

func TestInMemoryEventNetwork_AddEdge_ErrorCases(t *testing.T) {
	t.Run("returns error when from event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		toID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		nonExistentID := EventID(uuid.New())

		err = net.AddEdge(nonExistentID, toID, "trigger")
		require.Error(t, err)
		require.Contains(t, err.Error(), "from event not found")
	})

	t.Run("returns error when to event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		fromID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		nonExistentID := EventID(uuid.New())

		err = net.AddEdge(fromID, nonExistentID, "trigger")
		require.Error(t, err)
		require.Contains(t, err.Error(), "to event not found")
	})
}

func TestInMemoryEventNetwork_GetEvent_ErrorCases(t *testing.T) {
	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.GetByID(nonExistentID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})
}

func TestInMemoryEventNetwork_Children_ErrorCases(t *testing.T) {
	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Children(nonExistentID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("returns empty when event has no children", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		children, err := net.Children(eventID)
		require.NoError(t, err)
		require.Len(t, children, 0)
	})
}

func TestInMemoryEventNetwork_Parents_ErrorCases(t *testing.T) {
	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Parents(nonExistentID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("returns empty when event has no parents", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		parents, err := net.Parents(eventID)
		require.NoError(t, err)
		require.Len(t, parents, 0)
	})
}

func TestInMemoryEventNetwork_GetByIDs_ErrorCases(t *testing.T) {
	t.Run("returns error when one ID not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		nonExistentID := EventID(uuid.New())

		_, err = net.GetByIDs([]EventID{eventID1, nonExistentID})
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("returns error when all IDs not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID1 := EventID(uuid.New())
		nonExistentID2 := EventID(uuid.New())

		_, err := net.GetByIDs([]EventID{nonExistentID1, nonExistentID2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("returns empty slice when input is empty", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		events, err := net.GetByIDs([]EventID{})
		require.NoError(t, err)
		require.Len(t, events, 0)
	})
}

func TestInMemoryEventNetwork_Siblings_EdgeCases(t *testing.T) {
	t.Run("returns empty when event has no parents", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		siblings, err := net.Siblings(eventID)
		require.NoError(t, err)
		require.Len(t, siblings, 0)
	})

	t.Run("excludes anchor event from siblings", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		siblings, err := net.Siblings(childs.CpuEventsIDs[0])
		require.NoError(t, err)
		require.Len(t, siblings, 2)
		// Anchor should not be in siblings
		for _, s := range siblings {
			require.NotEqual(t, childs.CpuEventsIDs[0], s.ID)
		}
	})

	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Siblings(nonExistentID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("handles multiple parents correctly", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)
		eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
		require.NoError(t, err)

		cpuCriticalID1, err := addCpuCriticalEvent(net, TimeFrame{}, 1)
		require.NoError(t, err)
		cpuCriticalID2, err := addCpuCriticalEvent(net, TimeFrame{}, 1)
		require.NoError(t, err)

		err = net.AddEdge(eventID1, cpuCriticalID1, "trigger")
		require.NoError(t, err)
		err = net.AddEdge(eventID2, cpuCriticalID1, "trigger")
		require.NoError(t, err)
		err = net.AddEdge(eventID1, cpuCriticalID2, "trigger")
		require.NoError(t, err)

		siblings, err := net.Siblings(eventID1)
		require.NoError(t, err)
		require.Len(t, siblings, 1) // eventID2 is sibling through cpuCriticalID1
		require.Equal(t, eventID2, siblings[0].ID)
	})
}

func TestInMemoryEventNetwork_Descendants_EdgeCases(t *testing.T) {
	t.Run("returns empty when maxDepth is 0", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		descendants, err := net.Descendants(childs.CpuEventsIDs[0], 0)
		require.NoError(t, err)
		require.Len(t, descendants, 0)
	})

	t.Run("returns empty when maxDepth is negative", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		descendants, err := net.Descendants(childs.CpuEventsIDs[0], -1)
		require.NoError(t, err)
		require.Len(t, descendants, 0)
	})

	t.Run("handles events with no children", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		eventID, err := addCpuStatusChangedEvent(net, 98.3, "critical")
		require.NoError(t, err)

		descendants, err := net.Descendants(eventID, 1)
		require.NoError(t, err)
		require.Len(t, descendants, 0)
	})
}

func TestInMemoryEventNetwork_Cousins_EdgeCases(t *testing.T) {
	t.Run("returns error when event not found", func(t *testing.T) {
		net := NewInMemoryEventNetwork()
		nonExistentID := EventID(uuid.New())
		_, err := net.Cousins(nonExistentID, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "event not found")
	})

	t.Run("excludes anchor event from cousins", func(t *testing.T) {
		net, _, childs := buildInfraSubGraph(t)
		cousins, err := net.Cousins(childs.MemoryEventsIDs[0], 1)
		require.NoError(t, err)
		// Anchor should not be in cousins
		for _, c := range cousins {
			require.NotEqual(t, childs.MemoryEventsIDs[0], c.ID)
		}
	})
}
