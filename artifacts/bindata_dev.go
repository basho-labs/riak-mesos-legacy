// +build !rel

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ -debug data/riak_mesos_executor.tar.gz

package artifacts

import _ "github.com/jteeuwen/go-bindata"
