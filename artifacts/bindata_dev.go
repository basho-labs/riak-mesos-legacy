// +build !rel

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ -debug data/riak_mesos_executor.tar.gz data/riak.conf data/advanced.config data/riak-bin.tar.gz data/cepmd_$(OS_ARCH)

package artifacts

import _ "github.com/jteeuwen/go-bindata"
