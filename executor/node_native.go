//+build native

package main

import (
	log "github.com/Sirupsen/logrus"
)

func (riakNode *RiakNode) decompress() {
	log.Info("Native build, no need to get trusty root")
}
