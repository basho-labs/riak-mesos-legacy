package process_manager

import (
	log "github.com/Sirupsen/logrus"
	ps "github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
)

func TestTeardown(t *testing.T) {
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1<<20)
		for {
			<-sigs
			runtime.Stack(buf, true)
			log.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf)
		}
	}()
	assert := assert.New(t)

	re, err := NewProcessManager(func() { return }, "/bin/sleep", []string{"100"}, func() error { return nil }, nil, true)

	assert.Nil(err)
	re.TearDown()
	procs, err := ps.Processes()
	if err != nil {
		t.Fatal("Could not get OS processes")
	}
	pid := os.Getpid()
	for _, proc := range procs {
		if proc.PPid() == pid {
			assert.Fail("There are children proccesses leftover")
		}
	}
}

func TestNotify(t *testing.T) {
	assert := assert.New(t)
	re, err := NewProcessManager(func() { return }, "/bin/sleep", []string{"1"}, func() error { return nil }, nil, true)
	assert.Nil(err)
	status := <-re.Listen()
	assert.Equal(status, 0)
}
