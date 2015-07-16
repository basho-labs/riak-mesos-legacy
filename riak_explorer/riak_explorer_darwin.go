// +build darwin

package riak_explorer

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
)

func (re *RiakExplorer) start() {
	exepath := filepath.Join(".", re.tempdir, "riak_explorer", "bin", "riak_explorer")
	exe := exec.Command(exepath, "console", "-noinput")
	exe.Stdout = os.Stdout
	exe.Stderr = os.Stderr
	//TODO: Add for Linux
	//exe.SysProcAttr.Pdeathsig = syscall.SIGTERM
	err := exe.Start()
	if err != nil {
		log.Panic("Error starting explorer")
	} // TODO:
	re.exe = exe
}
