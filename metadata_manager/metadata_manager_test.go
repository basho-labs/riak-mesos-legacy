package metadata_manager

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStub(t *testing.T) {
	assert.True(t, true, "This is good. Canary test passing")
}
func TestNS(t *testing.T) {
	assert := assert.New(t)
	bns := baseNamespace{}
	namespace := makeSubSpace(makeSubSpace(makeSubSpace(bns, "bletchley"), "frameworks"), "fakeFramework")

	assert.Equal([]string{"", "bletchley", "frameworks", "fakeFramework"}, namespace.GetComponents())
}
