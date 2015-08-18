// +build !rel

//go:generate go-bindata -o bindata_generated.go -pkg=process_manager -prefix=schroot/data/ -debug schroot/data/
package process_manager
