package riak_explorer

import (
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	ps "github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"io/ioutil"
	"github.com/basho-labs/riak-mesos/common"
)

// TODO: Fix test and decompress trusty into "root"
// It needs to manage the root itself
func TestREX(t *testing.T) {
	if os.Getenv("TRAVIS") == "true" {
		t.Skip("Unable to run test on Travis")
	}
	assert := assert.New(t)

	mgr := metamgr.NewMetadataManager("rmf5", []string{"127.0.0.1"})
	c := cepm.NewCPMd(7902, mgr)

	go c.Background()
	// Port number for testing
	dirname, err := ioutil.TempDir("", "root")
	defer os.RemoveAll(dirname)
	t.Log("Decompressing into: ", dirname)
	assert.Nil(err)
	//defer os.RemoveAll(dirname)

	f, err := os.Open("../artifacts/data/trusty.tar.gz")
	assert.Nil(err)
	assert.Nil(common.ExtractGZ(dirname, f))
	f, err = os.Open("../artifacts/data/riak_explorer-bin.tar.gz")
	assert.Nil(err)
	assert.Nil(common.ExtractGZ(dirname, f))


	re, err := NewRiakExplorer(7901, "rex@ubuntu.", c, dirname, true) // 998th  prime number.
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
