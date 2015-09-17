//build +darwin

// This file is merely here so we can run tests in a darwin test environment.
// Process manager is never meant to run on a Mac OS X system. If you try to
// run this in darwin
package process_manager

func (pm *ProcessManager) start(executablePath string, args []string, chroot *string, useSuperChroot bool) {
	panic("Not implemented")
}
