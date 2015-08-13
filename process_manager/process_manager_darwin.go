//build +darwin
package process_manager

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
	"syscall"
)

func (pm *ProcessManager) start(executablePath string, args []string, chroot *string) {

	sysprocattr := &syscall.SysProcAttr{
		Setpgid: true,
	}
	if chroot != nil {
		sysprocattr.Chroot = *chroot
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
		procattr.Dir = wd
		homevar := fmt.Sprintf("HOME=%s", wd)
		procattr.Env = append(os.Environ(), homevar)
	}
	pathDetected := false
	for idx, val := range procattr.Env {
		splitArray := strings.Split(val, "=")
		if splitArray[0] == "PATH" {
			splitPath := strings.Split(splitArray[1], ":")
			splitPath = append(splitPath, "/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin:/bin", "/usr/games", "/usr/local/games")
			procattr.Env[idx] = fmt.Sprintf("PATH=%s", strings.Join(splitPath, ":"))
			pathDetected = true
			break
		}
	}
	if !pathDetected {
		procattr.Env = append(procattr.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games")
	}
	var err error
	realArgs := append([]string{executablePath}, args...)
	pm.pid, err = syscall.ForkExec(executablePath, realArgs, procattr)
	if err != nil {
		log.Panicf("Error starting process %v", err)
	} else {
		log.Infof("Process Manager started to manage %v at PID: %v", executablePath, pm.pid)
	}

}
