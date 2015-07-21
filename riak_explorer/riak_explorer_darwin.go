// +build darwin

package riak_explorer

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"fmt"
)

func (re *RiakExplorer) start() {
	exepath := filepath.Join(".", re.tempdir, "riak_explorer", "bin", "riak_explorer")
	exe := exec.Command(exepath, "console", "-noinput")
	exe.Stdout = os.Stdout
	exe.Stderr = os.Stderr
	//TODO: Add for Linux
	//exe.SysProcAttr.Pdeathsig = syscall.SIGTERM
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current working directory")
	}
	home := filepath.Join(wd, re.tempdir)
	homevar := fmt.Sprintf("HOME=%s", home)
	exe.Env = append(os.Environ(), homevar)

	err = exe.Start()
	if err != nil {
		log.Panic("Error starting explorer")
	} // TODO:
	re.exe = exe
}
