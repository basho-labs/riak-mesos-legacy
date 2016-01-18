// +build erlang

package scheduler

//ExecutorArgs specifies empty arg list for erlang executor
func ExecutorArgs(_currentID string) []string {
	return []string{}
}

//ExecutorShell specifies the shell setting for erlang executor
func ExecutorShell() bool {
	return true
}

//ExecutorValue specifies the executable for erlang executor
func ExecutorValue() string {
	return "./riak_mesos_executor/bin/ermf-executor"
}
