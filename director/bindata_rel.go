// +build !dev,!native

//go:generate go-bindata -o bindata_generated.go -pkg=main -prefix=../artifacts/data/ ../artifacts/data/riak_mesos_director-bin.tar.gz ../artifacts/data/trusty.tar.gz

package main
