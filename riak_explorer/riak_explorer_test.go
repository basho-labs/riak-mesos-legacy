package riak_explorer

import (
	"github.com/stretchr/testify/assert"
	"testing"
	ps "github.com/mitchellh/go-ps"
	"os"
)

func TestNothing(t *testing.T) {
	assert := assert.New(t)

	// Port number for testing
	re, err := NewRiakExplorer(7901, "rex@127.0.0.1") // 998th  prime number.
	assert.Equal(nil, err)
	re.TearDown()
	_, err = re.NewRiakExplorerClient().Ping()
	assert.NotNil(err)
	procs, err := ps.Processes()
	if err != nil {
		t.Fatal("Could not get OS processes")
	}
	pid := os.Getpid()
	for _, proc := range procs {
		if proc.PPid() == pid {
			assert.Fail("There are children proccesses leftover")
		}
	}

}
