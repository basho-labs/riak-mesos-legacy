package metadata_manager

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStub(t *testing.T) {
	assert.True(t, true, "This is good. Canary test passing")
}

func TestManager(t *testing.T) {
	manager := NewMetadataManager()
	manager.GetClusters()
}
