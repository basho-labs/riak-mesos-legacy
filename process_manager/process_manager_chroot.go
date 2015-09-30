//+build !native

package process_manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

func (pm *ProcessManager) maybeChroot(executablePath string, args []string, chroot *string, useSuperChroot bool) {
	var err error
	procattr := GetProcAttributes()

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
