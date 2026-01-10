package event_network

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
