// +build rel

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ data/executor_linux_amd64 data/riak.conf data/advanced.config data/riak-bin.tar.gz

package artifacts
