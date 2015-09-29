//build +linux
package process_manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

func getProcAttributes() *syscall.ProcAttr {
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

func (pm *ProcessManager) start(executablePath string, args []string) {
	realArgs := []string{}
	realArgs = append([]string{executablePath}, args...)
	procattr := getProcAttributes()
	pm.doStart(executablePath, realArgs, procattr)
}

func (pm *ProcessManager) startChroot(executablePath string, args []string, chroot *string, useSuperChroot bool) {
	var err error
	procattr := getProcAttributes()

	cpResolvCmd := exec.Command("/bin/cp", "/etc/resolv.conf", *chroot+"/etc/resolv.conf")
	log.Info(cpResolvCmd.Args)
	err = cpResolvCmd.Run()
	if err != nil {
		log.Info("Non-zero exit from command")
	}
	cpHostsCmd := exec.Command("/bin/cp", "/etc/hosts", *chroot+"/etc/hosts")
	log.Info(cpHostsCmd.Args)
	err = cpHostsCmd.Run()
	if err != nil {
		log.Info("Non-zero exit from command")
	}

	if useSuperChroot {
		log.Info("Assets: ", AssetNames())
		err = RestoreAsset(*chroot, "super_chroot")
		if err != nil {
			log.Panic("Unable to decompress: ", err)
		}
		args = append([]string{*chroot, executablePath}, args...)
		executablePath = filepath.Join(*chroot, "super_chroot")
	} else {
		procattr.Sys.Chroot = *chroot
		procattr.Dir = "/"
		if os.Getuid() != 0 {
			//Namespace tricks
			procattr.Sys.Cloneflags = syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS
			procattr.Sys.UidMappings = []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			}
		}
	}

	realArgs := []string{}
	realArgs = append([]string{executablePath}, args...)

	pm.doStart(executablePath, realArgs, procattr)
}

func (pm *ProcessManager) doStart(executablePath string, args []string, procattr *syscall.ProcAttr) {
	var err error
	log.Infof("Getting Ready to start process: %v with args: %v and ProcAttr: %+v", executablePath, args, procattr)
	pm.pid, err = syscall.ForkExec(executablePath, args, procattr)
	if err != nil {
		log.Panicf("Error starting process %v", err)
	} else {
		log.Infof("Process Manager started to manage %v at PID: %v", executablePath, pm.pid)
	}
}
