package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadConfig(t *testing.T) {
	file := "../testdata/test_readConfig.yaml"
	c, err := ReadConfig(file)
	assert.NoError(t, err)
	assert.Len(t, c.Nodes, 2)
	n1 := c.Nodes[0]
	assert.Equal(t, n1.Id, 1)
	assert.Equal(t, n1.Address, "123")
	assert.Equal(t, n1.Port, "14")
	n2 := c.Nodes[1]
	assert.Equal(t, n2.Id, 2)
	assert.Equal(t, n2.Address, "123")
	assert.Equal(t, n2.Port, "15")
}
