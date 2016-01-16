// +build rel

//go:generate go-bindata -ignore=Makefile|download.make -o bindata_generated.go -pkg=process_manager -prefix=schroot/data/ schroot/data/
package process_manager

import _ "github.com/jteeuwen/go-bindata"
