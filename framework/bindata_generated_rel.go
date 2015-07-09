// +build rel

package main

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go data/

import bindata "bindata_generated"

// Asset reads the file at the abs path given
func Asset(name string) ([]byte, error) {
	return bindata.Asset(name)
}
