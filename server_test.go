package server

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/raft/config"
	"github.com/stretchr/testify/assert"
)

func Test_NewServer(t *testing.T) {
	cfile := "./testdata/config.yaml"
	conf, err := config.ReadConfig(cfile)
	assert.NoError(t, err)
	s, err := NewServer(1, conf)
	assert.NoError(t, err)

	go s.Listen()

	output, err := exec.Command("nc", "-vz", "127.0.0.1", "2000").CombinedOutput()
	fmt.Println(string(output))

	assert.NoError(t, s.Close())
}
