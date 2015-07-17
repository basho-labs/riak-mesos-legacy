// +build !rel

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go -pkg=scheduler -prefix=data/scheduler -debug data/

package scheduler
