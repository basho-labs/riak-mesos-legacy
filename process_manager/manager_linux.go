//+build linux

package process_manager

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

const openFilesLimit uint64 = 65536

func GetProcAttributes() *syscall.ProcAttr {
	env := os.Environ()

	sysprocattr := &syscall.SysProcAttr{
		Setpgid: true,
	}

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

	return procattr
}

func (pm *ProcessManager) start(executablePath string, args []string, chroot *string, useSuperChroot bool) {
	pm.maybeChroot(executablePath, args, chroot, useSuperChroot)
}

// setOpenFilesLimit sets the open file limit in the kernel
// cur is the soft limit, max is the ceiling (or hard limit) for that limit
func (pm *ProcessManager) setOpenFilesLimit(cur, max uint64) error {
	var rLimit syscall.Rlimit
	// First check if the limits are already what we want
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	// If the current values are less than we want, set them
	if rLimit.Cur < cur || rLimit.Max < max {
		if cur > rLimit.Cur {
			rLimit.Cur = cur
		}
		if max > rLimit.Max {
			rLimit.Max = max
		}

		log.Infof("Setting open files limit (soft, hard) to (%v, %v)", rLimit.Cur, rLimit.Max)
		err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *ProcessManager) doStart(executablePath string, args []string, procattr *syscall.ProcAttr) {
	var err error
	log.Infof("Getting Ready to start process: %v with args: %v and ProcAttr: %+v", executablePath, args, procattr)
	err = pm.setOpenFilesLimit(openFilesLimit, openFilesLimit)
	if err != nil {
		log.Error("Error setting open files limit", err)
	}

	pm.pid, err = syscall.ForkExec(executablePath, args, procattr)
	if err != nil {
		log.Panicf("Error starting process %v", err)
	} else {
		log.Infof("Process Manager started to manage %v at PID: %v", executablePath, pm.pid)
	}
}
