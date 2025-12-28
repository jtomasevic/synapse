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

func TestInMemoryEventNetwork_GetByIDs(t *testing.T) {
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
