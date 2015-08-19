package riak_explorer

import (
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	ps "github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNothing(t *testing.T) {
	assert := assert.New(t)

	mgr := metamgr.NewMetadataManager("rmf5", []string{"127.0.0.1"})
	c := cepm.NewCPMd(7902, mgr)

	go c.Background()
	// Port number for testing
	re, err := NewRiakExplorer(7901, "rex@ubuntu.", c) // 998th  prime number.
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
