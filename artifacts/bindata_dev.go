// +build !rel

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go -pkg=artifacts -prefix=data/ -debug data/riak_explorer-bin.tar.gz data/riak-2.1.1-bin.tar.gz data/trusty.tar.gz

package artifacts
