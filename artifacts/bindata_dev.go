// +build !rel

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ -debug data/riak_explorer-bin.tar.gz data/riak-2.1.1-bin.tar.gz data/trusty.tar.gz

package artifacts
import _ "github.com/jteeuwen/go-bindata"
