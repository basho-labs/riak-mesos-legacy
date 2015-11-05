//+build !native

package main

import (
	"bytes"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
)

func decompress() {
	var err error
	var asset []byte
	if err := os.Mkdir("director", 0777); err != nil {
		log.Fatal("Unable to make director directory: ", err)
	}

	asset, err = Asset("trusty.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	if err = common.ExtractGZ("director", bytes.NewReader(asset)); err != nil {
		log.Fatal("Unable to extract trusty root: ", err)
	}

	asset, err = Asset("riak_mesos_director-bin.tar.gz")

	if err != nil {
		log.Fatal(err)
	}
	if err = common.ExtractGZ("director", bytes.NewReader(asset)); err != nil {
		log.Fatal("Unable to extract rex: ", err)
	}
}
