// +build rel

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go -pkg=process_manager -prefix=schroot/data/ schroot/data/
package process_manager
