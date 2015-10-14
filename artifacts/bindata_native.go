// +build !dev,native

//go:generate go-bindata -o bindata_generated.go -pkg=artifacts -prefix=data/ data/riak-bin.tar.gz

package artifacts
