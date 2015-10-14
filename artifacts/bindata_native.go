// +build !dev,native

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ data/riak-2.1.1-bin.tar.gz

package artifacts
