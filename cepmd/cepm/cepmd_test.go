package cepm


import (
	"testing"
	"github.com/stretchr/testify/assert"
	"net"
)

func TestListen(t *testing.T) {
	assert := assert.New(t)
	_ = assert
	cpmd := setupCPMd(0)
	t.Log("Listening: ", cpmd.ln.Addr())
   	_, err := net.Dial("tcp", cpmd.ln.Addr().String())
	assert.Nil(err)
}
