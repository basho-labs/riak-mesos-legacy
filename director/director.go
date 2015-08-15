package main

//go:generate go-bindata -o bindata_generated.go  -prefix=data/ data/

import log "github.com/Sirupsen/logrus"

func main() {
	log.SetLevel(log.DebugLevel)

	directorNode := NewDirectorNode()
	directorNode.Run()
}
