package event_network

import (
	"encoding/json"
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

func TestInMemoryEventNetwork_Children(t *testing.T) {
	network, parentNodes, _ := buildInfraSubGraph(t)
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

	cousins, err = network.Cousins(childNodes.MemoryEventsIDs[0], 2)
	require.NoError(t, err)
	require.Equal(t, len(cousins), 1)

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

func addCpuStatusChangedEvent(network EventNetwork,
	percentage float64,
	level string) (EventID, error) {
	event := Event{
		EventType:   CpuStatusChanged,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"percentage": percentage,
			"level":      level,
		},
	}
	return network.AddEvent(event)
}

func addMemoryStatusChangedEvent(network EventNetwork,
	percentage float64,
	level string) (EventID, error) {
	event := Event{
		EventType:   MemoryStatusChanged,
		EventDomain: InfraDomain,
		Timestamp:   time.Now(),
		Properties: EventProps{
			"percentage": percentage,
			"level":      level,
		},
	}
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

func buildInfraSubGraph(t *testing.T) (EventNetwork, ParentNodes, ChildNodes) {
	childNodes := ChildNodes{}
	network := NewInMemoryEventNetwork()
	eventID1, err := addCpuStatusChangedEvent(network, 98.3, "critical")
	require.NotEmpty(t, eventID1)
	require.NoError(t, err)
	eventID2, err := addCpuStatusChangedEvent(network, 95.2, "critical")
	require.NotEmpty(t, eventID2)
	require.NoError(t, err)
	eventID3, err := addCpuStatusChangedEvent(network, 97.1, "critical")
	require.NotEmpty(t, eventID3)
	require.NoError(t, err)
	event1, err := network.GetByID(eventID1)
	require.NoError(t, err)
	require.NotEmpty(t, event1)
	event2, err := network.GetByID(eventID2)
	require.NoError(t, err)
	require.NotEmpty(t, event2)
	event3, err := network.GetByID(eventID3)
	require.NoError(t, err)
	require.NotEmpty(t, event3)

	memoryEventID1, err := addMemoryStatusChangedEvent(network, 90.2, "critical")
	require.NotEmpty(t, memoryEventID1)
	require.NoError(t, err)
	memoryEventID2, err := addMemoryStatusChangedEvent(network, 88.5, "critical")
	require.NotEmpty(t, memoryEventID2)
	require.NoError(t, err)
	memoryEventID3, err := addMemoryStatusChangedEvent(network, 91.0, "critical")
	require.NotEmpty(t, memoryEventID3)
	require.NoError(t, err)
	memoryEvent1, err := network.GetByID(memoryEventID1)
	require.NoError(t, err)
	require.NotEmpty(t, memoryEvent1)
	memoryEvent2, err := network.GetByID(memoryEventID2)
	require.NoError(t, err)
	require.NotEmpty(t, memoryEvent2)
	memoryEvent3, err := network.GetByID(memoryEventID3)
	require.NoError(t, err)
	require.NotEmpty(t, memoryEvent3)

	cpuCriticalEventId, err := addCpuCriticalEvent(network, TimeFrame{
		From: event1.Timestamp,
		To:   event1.Timestamp,
	}, 3)
	require.NoError(t, err)
	require.NotEmpty(t, cpuCriticalEventId)
	// adding edges
	err = network.AddEdge(eventID1, cpuCriticalEventId, "trigger")
	require.NoError(t, err)
	err = network.AddEdge(eventID2, cpuCriticalEventId, "trigger")
	require.NoError(t, err)
	err = network.AddEdge(eventID3, cpuCriticalEventId, "trigger")
	cpuCriticalEvent, err := network.GetByID(cpuCriticalEventId)
	require.NoError(t, err)
	require.NotEmpty(t, cpuCriticalEvent)

	memoryCriticalEventId, err := addMemoryCriticalEvent(network, TimeFrame{
		From: memoryEvent1.Timestamp,
		To:   memoryEvent3.Timestamp,
	}, 3)
	require.NoError(t, err)
	require.NotEmpty(t, memoryCriticalEventId)
	err = network.AddEdge(memoryEventID1, memoryCriticalEventId, "trigger")
	require.NoError(t, err)
	err = network.AddEdge(memoryEventID2, memoryCriticalEventId, "trigger")
	require.NoError(t, err)
	err = network.AddEdge(memoryEventID3, memoryCriticalEventId, "trigger")
	require.NoError(t, err)
	memoryCriticalEvent, err := network.GetByID(memoryCriticalEventId)
	require.NoError(t, err)
	require.NotEmpty(t, memoryCriticalEvent)

	require.NoError(t, err)
	require.NotEmpty(t, network)
	childNodes.CpuEventsIDs = append(childNodes.CpuEventsIDs, eventID1, eventID2, eventID3)
	childNodes.MemoryEventsIDs = append(childNodes.MemoryEventsIDs, memoryEventID1, memoryEventID2, memoryEventID3)

	nodeCriticalDate := GetLatestTime([]time.Time{
		memoryEvent1.Timestamp,
		cpuCriticalEvent.Timestamp,
	})

	nodeCriticalEventId, err := addNodeStatusChangedEvent(network, "critical", nodeCriticalDate)
	require.NoError(t, err)
	require.NotEmpty(t, nodeCriticalEventId)

	err = network.AddEdge(memoryCriticalEventId, nodeCriticalEventId, "trigger")
	require.NoError(t, err)
	err = network.AddEdge(cpuCriticalEventId, nodeCriticalEventId, "trigger")
	require.NoError(t, err)

	return network, ParentNodes{
		CpuCriticalID:            cpuCriticalEventId,
		MemoryCriticalID:         memoryCriticalEventId,
		ServerNodeChangeStatusID: nodeCriticalEventId,
	}, childNodes
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
