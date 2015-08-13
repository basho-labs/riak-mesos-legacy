//build +linux
package process_manager

import (
	"syscall"
	"os"
	log "github.com/Sirupsen/logrus"
	"fmt"
)

func (pm *ProcessManager) start(executablePath string, args []string, chroot *string) {

	sysprocattr := &syscall.SysProcAttr{
		Setpgid: true,
	}
	if chroot != nil {
		sysprocattr.Chroot = *chroot
		if os.Getuid() != 0 {
			//Namespace tricks
			sysprocattr.Cloneflags = syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS
			sysprocattr.UidMappings = []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			}
		}
	}
	env := os.Environ()

	procattr := &syscall.ProcAttr{
		Sys:   sysprocattr,
		Env:   env,
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	if os.Getenv("HOME") == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Could not get current working directory")
		}
		if chroot != nil {
			procattr.Dir = "/"
		} else {
			procattr.Dir = wd
		}
		homevar := fmt.Sprintf("HOME=%s", wd)
		procattr.Env = append(os.Environ(), homevar)
	}
	var err error
	realArgs := append([]string{executablePath}, args...)

	log.Infof("Getting Ready to start process: %v with args: %v and ProcAttr: %+v and %+v", executablePath, realArgs, procattr, sysprocattr)

	pm.pid, err = syscall.ForkExec(executablePath, realArgs, procattr)
	if err != nil {
		log.Panicf("Error starting process %v", err)
	} else {
		log.Infof("Process Manager started to manage %v at PID: %v", executablePath, pm.pid)
	}

}