package main

import log "github.com/Sirupsen/logrus"

func main() {
	log.SetLevel(log.DebugLevel)

	directorNode := NewDirectorNode()
	directorNode.Run()
}
