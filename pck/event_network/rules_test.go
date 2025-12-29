package event_network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDeriveEventRule(t *testing.T) {
	condition := NewCondition().IsTypeOf(CpuStatusChanged, Conditions{})
	template := EventTemplate{
		EventType:   CpuCritical,
		EventDomain: InfraDomain,
		EventProps:  map[string]any{"occurs": 1},
	}

	rule := NewDeriveEventRule(condition, template)

	require.NotNil(t, rule)
	require.Equal(t, rule.ActionType, DeriveNode)
	require.Equal(t, rule.Condition, condition)
	require.Equal(t, rule.EventTemplate, template)
}

func TestDeriveEventRule_GetActionType(t *testing.T) {
	rule := NewDeriveEventRule(
		NewCondition().IsTypeOf(CpuStatusChanged, Conditions{}),
		EventTemplate{EventType: CpuCritical, EventDomain: InfraDomain},
	)

	require.Equal(t, rule.GetActionType(), DeriveNode)
}

func TestDeriveEventRule_GetActionTemplate(t *testing.T) {
	template := EventTemplate{
		EventType:   CpuCritical,
		EventDomain: InfraDomain,
		EventProps:  map[string]any{"occurs": 5},
	}
	rule := NewDeriveEventRule(
		NewCondition().IsTypeOf(CpuStatusChanged, Conditions{}),
		template,
	)

	require.Equal(t, rule.GetActionTemplate(), template)
}

func TestDeriveEventRule_BindNetwork(t *testing.T) {
	network := NewInMemoryEventNetwork()
	rule := NewDeriveEventRule(
		NewCondition().IsTypeOf(CpuStatusChanged, Conditions{}),
		EventTemplate{EventType: CpuCritical, EventDomain: InfraDomain},
	)

	require.Nil(t, rule.Network)
	rule.BindNetwork(network)
	require.NotNil(t, rule.Network)
	require.Equal(t, rule.Network, network)
	require.NotNil(t, rule.conditionCompiler)
}

func TestDeriveEventRule_Process_WithConditionSuccess(t *testing.T) {
	network := NewInMemoryEventNetwork()
	// Create a simple rule that checks if event is of type CpuStatusChanged
	rule := NewDeriveEventRule(
		NewCondition().IsTypeOf(CpuStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	)
	rule.BindNetwork(network)

	// Create and process a matching event
	eventID, err := addCpuStatusChangedEvent(network, 95.5, "critical")
	require.NoError(t, err)
	event, err := network.GetByID(eventID)
	require.NoError(t, err)

	ok, events, err := rule.Process(event)
	require.NoError(t, err)
	require.True(t, ok)
	// Simple IsTypeOf conditions return no events; that's expected
	require.NotNil(t, events)
}

func TestDeriveEventRule_Process_WithConditionFailure(t *testing.T) {
	network := NewInMemoryEventNetwork()
	// Create a rule that checks for MemoryStatusChanged
	rule := NewDeriveEventRule(
		NewCondition().IsTypeOf(MemoryStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	)
	rule.BindNetwork(network)

	// Create a CpuStatusChanged event (does not match the rule condition)
	eventID, err := addCpuStatusChangedEvent(network, 95.5, "critical")
	require.NoError(t, err)
	event, err := network.GetByID(eventID)
	require.NoError(t, err)

	ok, events, err := rule.Process(event)
	require.Error(t, err)
	require.False(t, ok)
	require.Nil(t, events)
	require.Equal(t, err.Error(), "expression is not satisfied")
}

func TestDeriveEventRule_Process_WithHasSiblingsCondition(t *testing.T) {
	network := NewInMemoryEventNetwork()
	// Rule requires exactly 2 CpuStatusChanged siblings
	rule := NewDeriveEventRule(
		NewCondition().HasSiblings(CpuStatusChanged, Conditions{
			Counter: &Counter{
				HowMany:       2,
				HowManyOrMore: false,
			},
		}),
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 2,
			},
		},
	)
	rule.BindNetwork(network)

	// Add first event
	eventID1, _ := addCpuStatusChangedEvent(network, 98.3, "critical")
	event1, _ := network.GetByID(eventID1)
	ok, events, err := rule.Process(event1)
	require.Error(t, err) // Should fail - only 1 event, no siblings

	// Add second event - now they are siblings
	eventID2, _ := addCpuStatusChangedEvent(network, 95.2, "critical")
	event2, _ := network.GetByID(eventID2)
	ok, events, err = rule.Process(event2)
	require.Error(t, err) // Still fails - 1 sibling, need 2

	// Add third event - now we have 2 siblings for event2
	eventID3, err := addCpuStatusChangedEvent(network, 91.11, "critical")
	event3, _ := network.GetByID(eventID3)
	ok, events, err = rule.Process(event3)
	require.NoError(t, err) // Should succeed - event3 has 2 siblings (event1 and event2)
	require.True(t, ok)
	require.NotEmpty(t, events)
}

func TestDeriveEventRule_Process_WithHasChildCondition(t *testing.T) {
	network := NewInMemoryEventNetwork()

	// Create parent and child events
	parentID, err := addCpuCriticalEvent(network, TimeFrame{}, 1)
	require.NoError(t, err)

	childID, err := addCpuStatusChangedEvent(network, 95.5, "critical")
	require.NoError(t, err)

	// Add edge from child to parent
	err = network.AddEdge(childID, parentID, "trigger")
	require.NoError(t, err)

	// Rule checks if event has a CpuStatusChanged child
	rule := NewDeriveEventRule(
		NewCondition().HasChild(CpuStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   ServerNodeChangeStatus,
			EventDomain: InfraDomain,
			EventProps:  map[string]any{},
		},
	)
	rule.BindNetwork(network)

	parent, err := network.GetByID(parentID)
	require.NoError(t, err)

	ok, events, err := rule.Process(parent)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, events)
}

func TestDeriveEventRule_Process_WithComplexCondition(t *testing.T) {
	network := NewInMemoryEventNetwork()

	// Create a complex condition with AND operator
	rule := NewDeriveEventRule(
		NewCondition().
			IsTypeOf(CpuStatusChanged, Conditions{}).
			And().
			IsTypeOf(CpuStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps:  map[string]any{"occurs": 1},
		},
	)
	rule.BindNetwork(network)

	eventID, err := addCpuStatusChangedEvent(network, 90.0, "critical")
	require.NoError(t, err)
	event, err := network.GetByID(eventID)
	require.NoError(t, err)

	ok, events, err := rule.Process(event)
	require.NoError(t, err)
	require.True(t, ok)
	// Complex conditions with logical operators return empty events list
	require.NotNil(t, events)
}

func TestDeriveEventRule_Process_WithNilCondition(t *testing.T) {
	network := NewInMemoryEventNetwork()

	rule := NewDeriveEventRule(
		nil,
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps:  map[string]any{},
		},
	)
	rule.BindNetwork(network)

	eventID, err := addCpuStatusChangedEvent(network, 90.0, "critical")
	require.NoError(t, err)
	event, err := network.GetByID(eventID)
	require.NoError(t, err)

	// Should handle nil condition gracefully
	_, _, err = rule.Process(event)
	require.Error(t, err)
}

func TestDeriveEventRule_EventTemplateProperties(t *testing.T) {
	condition := NewCondition().IsTypeOf(CpuStatusChanged, Conditions{})
	props := map[string]any{
		"percentage": 95.0,
		"level":      "critical",
		"timestamp":  "2025-12-29",
	}
	template := EventTemplate{
		EventType:   CpuCritical,
		EventDomain: InfraDomain,
		EventProps:  props,
	}

	rule := NewDeriveEventRule(condition, template)
	require.Equal(t, rule.GetActionTemplate().EventProps, props)
	require.Len(t, rule.GetActionTemplate().EventProps, 3)
}

func TestDeriveEventRule_MultipleRulesWithDifferentConditions(t *testing.T) {
	network := NewInMemoryEventNetwork()

	// Create rule 1
	rule1 := NewDeriveEventRule(
		NewCondition().IsTypeOf(CpuStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps:  map[string]any{"source": "cpu"},
		},
	)
	rule1.BindNetwork(network)

	// Create rule 2
	rule2 := NewDeriveEventRule(
		NewCondition().IsTypeOf(MemoryStatusChanged, Conditions{}),
		EventTemplate{
			EventType:   MemoryCritical,
			EventDomain: InfraDomain,
			EventProps:  map[string]any{"source": "memory"},
		},
	)
	rule2.BindNetwork(network)

	// Test rule 1
	cpuEventID, _ := addCpuStatusChangedEvent(network, 95.0, "critical")
	cpuEvent, _ := network.GetByID(cpuEventID)
	ok1, _, err1 := rule1.Process(cpuEvent)
	require.NoError(t, err1)
	require.True(t, ok1)

	// Rule 1 should fail on memory event
	memoryEventID, _ := addMemoryStatusChangedEvent(network, 85.0, "critical")
	memoryEvent, _ := network.GetByID(memoryEventID)
	_, _, err2 := rule1.Process(memoryEvent)
	require.Error(t, err2)

	// Rule 2 should succeed on memory event
	ok2, _, err3 := rule2.Process(memoryEvent)
	require.NoError(t, err3)
	require.True(t, ok2)
}
