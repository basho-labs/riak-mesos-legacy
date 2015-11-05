//+build !native

package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
	"net/http"
)

func (riakNode *RiakNode) decompress() {
	var err error
	var fetchURI string
	var resp *http.Response

	fetchURI = fmt.Sprintf("%s/static2/trusty.tar.gz", riakNode.taskData.URI)
	log.Info("Preparing to fetch trusty_root from: ", fetchURI)
	resp, err = http.Get(fetchURI)
	if err != nil {
		log.Panic("Unable to fetch trusty root: ", err)
	}
	err = common.ExtractGZ("root", resp.Body)
	if err != nil {
		log.Panic("Unable to extract trusty root: ", err)
	}
}
