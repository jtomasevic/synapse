package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForSynapse_ReturnsBoundClient(t *testing.T) {
	c := &Client{}
	syn := c.ForSynapse("syn-42")

	require.NotNil(t, syn)
	assert.Equal(t, "syn-42", syn.SynapseID())
}

func TestForSynapse_SharesParentClient(t *testing.T) {
	c := &Client{}
	syn1 := c.ForSynapse("syn-1")
	syn2 := c.ForSynapse("syn-2")

	assert.Same(t, c, syn1.client)
	assert.Same(t, c, syn2.client)
	assert.NotEqual(t, syn1.SynapseID(), syn2.SynapseID())
}

func TestForSynapse_MultipleSynapsesIndependent(t *testing.T) {
	c := &Client{}
	syn1 := c.ForSynapse("alpha")
	syn2 := c.ForSynapse("beta")

	assert.Equal(t, "alpha", syn1.SynapseID())
	assert.Equal(t, "beta", syn2.SynapseID())
}
