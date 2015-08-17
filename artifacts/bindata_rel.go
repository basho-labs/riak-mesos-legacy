// +build rel

//go:generate go-bindata -ignore=Makefile|deploy_key|deploy_keypub  -o bindata_generated.go -pkg=artifacts -prefix=data/ data/

package artifacts
