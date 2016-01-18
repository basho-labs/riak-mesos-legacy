// +build !erlang

package scheduler

//ExecutorArgs specifies the arg list for golang executor
func ExecutorArgs(currentID string) []string {
	return []string{ExecutorValue(), "-logtostderr=true", "-taskinfo", currentID}
}

//ExecutorShell specifies the shell setting for golang executor
func ExecutorShell() bool {
	return false
}

//ExecutorValue specifies the executable for golang executor
func ExecutorValue() string {
	return "./executor_linux_amd64"
}
