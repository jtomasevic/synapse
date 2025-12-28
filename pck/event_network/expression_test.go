package event_network

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExpression_IsTypeOf(t *testing.T) {
	net, _, childs := buildInfraSubGraph(t)

	ev, _ := net.GetByID(childs.CpuEventsIDs[0])
	ok, _, err := NewExpression(net, &ev).
		IsTypeOf(CpuStatusChanged, Conditions{}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_InDomain(t *testing.T) {
	net, _, childs := buildInfraSubGraph(t)

	ev, _ := net.GetByID(childs.MemoryEventsIDs[0])
	ok, _, err := NewExpression(net, &ev).
		InDomain(InfraDomain).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasChild(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasChild(CpuStatusChanged, Conditions{}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasChild_WithCounterExact(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	fmt.Println(ev.EventType)
	ok, _, err := NewExpression(net, &ev).
		HasChild(CpuStatusChanged, Conditions{
			Counter: &Counter{HowMany: 3},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasChild_WithCounterOrMore(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasChild(CpuStatusChanged, Conditions{
			Counter: &Counter{HowMany: 1, HowManyOrMore: true},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasDescendants(t *testing.T) {
	net, parents, childs := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasDescendants(CpuStatusChanged, Conditions{}).
		Eval()

	require.NoError(t, err)
	require.False(t, ok)

	ev, _ = net.GetByID(childs.CpuEventsIDs[0])
	ok, _, err = NewExpression(net, &ev).
		HasDescendants(CpuCritical, Conditions{}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasDescendants_WithDepth(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasDescendants(ServerNodeChangeStatus, Conditions{MaxDepth: 1}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_HasDescendants_WithCount(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasDescendants(ServerNodeChangeStatus, Conditions{Counter: &Counter{
			HowMany:       1,
			HowManyOrMore: false,
		}}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_ChildrenContainsPredicate(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	// Derived event
	ev, _ := net.GetByID(parents.CpuCriticalID)

	ok, _, err := NewExpression(net, &ev).
		ChildrenContains(func(e *Event) bool {
			// Under Option A, children are semantic nodes returned by EventNetwork.Children
			return e.EventType == CpuCritical
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_TimeWindow(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.CpuCriticalID)
	ok, _, err := NewExpression(net, &ev).
		HasChild(CpuStatusChanged, Conditions{
			TimeWindow: &TimeWindow{
				Within:   10,
				TimeUnit: Minute,
			},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_PropertyFilter(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	ev, _ := net.GetByID(parents.MemoryCriticalID)

	ok, _, err := NewExpression(net, &ev).
		HasChild(MemoryStatusChanged, Conditions{
			PropertyValues: map[string]any{
				"level": "critical", // property exists on derived event
			},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_Siblings(t *testing.T) {
	net, _, childs := buildInfraSubGraph(t)

	ev, _ := net.GetByID(childs.CpuEventsIDs[0])
	ok, _, err := NewExpression(net, &ev).
		HasSiblings(CpuStatusChanged, Conditions{
			Counter: &Counter{HowMany: 2},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_Siblings_Without_Root(t *testing.T) {
	net := NewInMemoryEventNetwork()
	eventID1, err := addCpuStatusChangedEvent(net, 98.3, "critical")
	require.NotEmpty(t, eventID1)
	require.NoError(t, err)
	eventID2, err := addCpuStatusChangedEvent(net, 95.2, "critical")
	require.NotEmpty(t, eventID2)
	require.NoError(t, err)
	eventID3, err := addCpuStatusChangedEvent(net, 97.1, "critical")
	require.NotEmpty(t, eventID3)
	require.NoError(t, err)
	ev, _ := net.GetByID(eventID3)
	ok, _, err := NewExpression(net, &ev).
		HasSiblings(CpuStatusChanged, Conditions{
			Counter: &Counter{HowMany: 2},
		}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_Cousins(t *testing.T) {
	net, _, childs := buildInfraSubGraph(t)

	ev, _ := net.GetByID(childs.MemoryEventsIDs[0])
	ok, _, err := NewExpression(net, &ev).
		HasCousin(CpuStatusChanged, Conditions{MaxDepth: 2}).
		Eval()

	require.NoError(t, err)
	require.True(t, ok)
}

func TestExpression_GroupAndOrPrecedence(t *testing.T) {
	t.Run("simple grouping - simple - on group, one expression", func(t *testing.T) {
		net, parents, chlids := buildInfraSubGraph(t)
		fmt.Println(chlids)

		ev, _ := net.GetByID(parents.CpuCriticalID)

		ok, _, err := NewExpression(net, &ev).
			Group().
			HasChild(CpuStatusChanged, Conditions{MaxDepth: 2}).
			Ungroup().
			Eval()

		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("grouping - complex 1", func(t *testing.T) {
		net, _, chlids := buildInfraSubGraph(t)
		fmt.Println(chlids)

		ev, _ := net.GetByID(chlids.CpuEventsIDs[0])

		ok, _, err := NewExpression(net, &ev).
			Group().
			HasDescendants(CpuCritical, Conditions{}).
			And().
			HasSiblings(CpuStatusChanged, Conditions{Counter: &Counter{HowMany: 2}}).
			Ungroup().
			And().
			HasCousin(MemoryStatusChanged, Conditions{MaxDepth: 2}).
			Eval()

		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("grouping - complex 2", func(t *testing.T) {
		net, _, chlids := buildInfraSubGraph(t)
		fmt.Println(chlids)

		ev, _ := net.GetByID(chlids.CpuEventsIDs[0])

		ok, _, err := NewExpression(net, &ev).
			Group().
			HasDescendants(CpuCritical, Conditions{}).
			And().
			HasSiblings(CpuStatusChanged, Conditions{Counter: &Counter{HowMany: 2}}).
			Ungroup().
			And().
			HasCousin(MemoryCritical, Conditions{MaxDepth: 1}). // <- false
			Eval()

		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("grouping - complex 2", func(t *testing.T) {
		net, _, chlids := buildInfraSubGraph(t)
		fmt.Println(chlids)

		ev, _ := net.GetByID(chlids.CpuEventsIDs[0])

		ok, _, err := NewExpression(net, &ev).
			Group().
			HasDescendants(CpuCritical, Conditions{}).
			And().
			HasSiblings(CpuStatusChanged, Conditions{Counter: &Counter{HowMany: 2}}).
			Ungroup().
			Or().                                               // <- because of OR whatever is bellow it will pass, so result will be True
			HasCousin(MemoryCritical, Conditions{MaxDepth: 1}). // <- false
			Eval()

		require.NoError(t, err)
		require.True(t, ok)
	})
}
