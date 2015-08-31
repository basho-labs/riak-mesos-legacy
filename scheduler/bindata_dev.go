// +build !rel

//go:generate go-bindata -ignore=.gitkeep -o bindata_generated.go -pkg=scheduler -prefix=data/ -debug data/

package scheduler
import _ "github.com/jteeuwen/go-bindata"
