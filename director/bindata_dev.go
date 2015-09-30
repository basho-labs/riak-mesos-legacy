// +build dev

//go:generate go-bindata -o bindata_generated.go -pkg=main -prefix=../artifacts/data/ -debug ../artifacts/data/riak_mesos_director-bin.tar.gz ../artifacts/data/trusty.tar.gz

package main

import _ "github.com/jteeuwen/go-bindata"
