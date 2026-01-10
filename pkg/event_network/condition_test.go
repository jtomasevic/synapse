package event_network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

/*
========================
Basic compilation
========================
*/

func TestConditionSpec_Compile_SimpleHasChild(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	spec := NewCondition().
		HasChild(CpuStatusChanged, Conditions{})

	compiler := NewConditionCompiler(net)

	anchor, err := net.GetByID(parents.CpuCriticalID)
	require.NoError(t, err)

	expr, err := compiler.Compile(spec, &anchor)
	require.NoError(t, err)
	require.NotNil(t, expr)

	ok, contributors, err := expr.Eval()
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, contributors, 3)
}

/*
========================
AND / OR chaining
========================
*/

func TestConditionSpec_Compile_AndChain(t *testing.T) {
	net, _, childs := buildInfraSubGraph(t)

	spec := NewCondition().
		HasDescendants(CpuCritical, Conditions{}).
		And().
		HasSiblings(CpuStatusChanged, Conditions{
			Counter: &Counter{HowMany: 2, HowManyOrMore: true},
		})

	compiler := NewConditionCompiler(net)
	anchor, _ := net.GetByID(childs.CpuEventsIDs[0])

	expr, err := compiler.Compile(spec, &anchor)
	require.NoError(t, err)

	ok, _, err := expr.Eval()
	require.NoError(t, err)
	require.True(t, ok)
}

/*
========================
Grouping / precedence
========================
*/

func TestConditionSpec_Compile_GroupingPrecedence(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	spec := NewCondition().
		Group().
		HasChild(CpuCritical, Conditions{MaxDepth: 1}).
		Or().
		HasChild(MemoryCritical, Conditions{MaxDepth: 1}).
		Ungroup().
		And().
		IsTypeOf(ServerNodeChangeStatus, Conditions{})

	compiler := NewConditionCompiler(net)
	anchor, _ := net.GetByID(parents.ServerNodeChangeStatusID)

	expr, err := compiler.Compile(spec, &anchor)
	require.NoError(t, err)

	ok, contributors, err := expr.Eval()
	require.NoError(t, err)
	require.True(t, ok)

	// cpu_critical + memory_critical should be contributors
	require.Len(t, contributors, 2)
}

/*
========================
Cousin semantics
========================
*/

func TestConditionSpec_Compile_HasCousin(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	spec := NewCondition().
		HasCousin(MemoryCritical, Conditions{MaxDepth: 1})

	compiler := NewConditionCompiler(net)
	anchor, _ := net.GetByID(parents.CpuCriticalID)

	expr, err := compiler.Compile(spec, &anchor)
	require.NoError(t, err)

	ok, contributors, err := expr.Eval()
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, contributors, 1)
}

/*
========================
Negative compilation cases
========================
*/

func TestConditionCompiler_NilSpec(t *testing.T) {
	net, parents, _ := buildInfraSubGraph(t)

	compiler := NewConditionCompiler(net)
	anchor, _ := net.GetByID(parents.CpuCriticalID)

	expr, err := compiler.Compile(nil, &anchor)
	require.Error(t, err)
	require.Nil(t, expr)
}

func TestConditionCompiler_NilAnchor(t *testing.T) {
	net, _, _ := buildInfraSubGraph(t)

	spec := NewCondition().
		HasChild(CpuStatusChanged, Conditions{})

	compiler := NewConditionCompiler(net)

	expr, err := compiler.Compile(spec, nil)
	require.Error(t, err)
	require.Nil(t, expr)
}
