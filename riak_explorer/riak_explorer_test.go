package riak_explorer

import (
	//"github.com/stretchr/testify/assert"
	"testing"
)

func TestNothing(t *testing.T) {
	re, err := NewRiakExplorer()
	if err == nil {
		defer re.TearDown()
	}
}
