//+build native

package process_manager

import (
	"path/filepath"
)

func (pm *ProcessManager) maybeChroot(executablePath string, args []string, chroot *string, _ bool) {
	executablePath = filepath.Join(*chroot, executablePath)

	realArgs := []string{}
	realArgs = append([]string{executablePath}, args...)
	procattr := GetProcAttributes()
	pm.doStart(executablePath, realArgs, procattr)
}
