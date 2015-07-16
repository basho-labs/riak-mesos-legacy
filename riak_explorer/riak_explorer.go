package riak_explorer

import (
	"archive/tar"
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type RiakExplorer struct {
	tempdir  string
	exe      *exec.Cmd
	waitChan chan interface{}
	teardown chan chan interface{}
}

func startExplorer(retChan chan *RiakExplorer) {

	signals := make(chan os.Signal, 3)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	defer close(retChan)

	tempDirChan := make(chan string)
	go decompress(tempDirChan)

	// TODO: Put fault-tolerance around decompression failure
	tempdir := <-tempDirChan

	waitChan := make(chan interface{})
	re := &RiakExplorer{
		tempdir:  tempdir,
		teardown: make(chan chan interface{}),
		waitChan: waitChan,
	}

	defer close(re.teardown)

	re.configure()
	re.start()

	go func() {
		log.Info("Wait starting")
		re.exe.Wait()
		log.Info("Wait ended")
		waitChan <- nil
		close(waitChan)
	}()

	for i := 0; i < 1000; i++ {
		select {
		case <-signals:
			{
				log.Info("Tearing down at signal")
				re.killProcess()
				re.deleteData()
				return
			}
		case <-re.waitChan:
			{
				log.Info("Deleting data after process died")
				re.killProcess()
				re.deleteData()
				return
			}
		case tearDownChan := <-re.teardown:
			{
				log.Info("Tearing down")
				re.killProcess()
				re.deleteData()
				tearDownChan <- nil
				return
			}
		case <-time.After(100 * time.Millisecond):
			{
				// Try pinging Riak Explorer
				_, err := NewRiakExplorerClient(fmt.Sprintf("localhost:9000")).Ping()
				if err == nil {
					retChan <- re
					// re.background() should never return
					re.background(signals)
					return
				} else {
					log.Info("Rex status: ", err)
				}
			}
		}
	}
}
func (re *RiakExplorer) TearDown() {
	log.Infof("RE: %+v", re)
	replyChan := make(chan interface{})
	log.Info("Teardown: ", re.teardown)
	re.teardown <- replyChan
	<-replyChan
	return
}

func (re *RiakExplorer) background(signals chan os.Signal) {
	select {
	case <-signals:
		{
			log.Info("Tearing down at signal")
			re.deleteData()
			re.killProcess()
		}
	case <-re.waitChan:
		{
			log.Info("Deleting data after process died")
			re.killProcess()
			re.deleteData()
		}
	case tearDownChan := <-re.teardown:
		{
			log.Info("Tearing down")
			re.killProcess()
			re.deleteData()
			tearDownChan <- nil
			return
		}
	}
}
func (re *RiakExplorer) killProcess() {
	re.exe.Process.Signal(syscall.SIGTERM)

	// Wait 5 seconds for the process to try to exit
	for i := 0; i < 50; i++ {
		if re.exe.Process.Signal(syscall.Signal(0)) == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Otherwise, kill it:
	re.exe.Process.Signal(syscall.SIGKILL)
}

func (re *RiakExplorer) deleteData() {
	log.Info("Deleting all data in: ", re.tempdir)
	err := os.RemoveAll(re.tempdir)
	if err != nil {
		log.Error(err)
	}
}
func decompress(ret chan string) {
	defer close(ret)
	assetPath := fmt.Sprintf("riak_explorer_%s_%s.tar", runtime.GOOS, runtime.GOARCH)
	asset, err := Asset(assetPath)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewReader(asset)
	tr := tar.NewReader(r)
	tempdir, err := ioutil.TempDir("./", "riak_explorer.")
	if err != nil {
		log.Fatal(err)
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		} else if err != nil {
			log.Fatalln(err)
		}
		filename := filepath.Join(".", tempdir, hdr.Name)
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
			io.Copy(file, tr)
			if err != nil {
				log.Fatal(err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				log.Fatal(err)
			}
			file.Close()
		} else if hdr.Typeflag == tar.TypeDir {
			err := os.Mkdir(filename, 0777)
			if err != nil {
				log.Fatalln(err)
			}
		} else if hdr.Typeflag == tar.TypeSymlink {
			if err := os.Symlink(hdr.Linkname, filename); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("Experienced unknown tar file type: ", hdr.Typeflag)
		}
	}
	ret <- tempdir
}
func (re *RiakExplorer) configure() {
	// TODO: Add dynamic port configuration
	configpath := filepath.Join(".", re.tempdir, "riak_explorer", "etc", "riak_explorer.conf")
	configfile, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		log.Fatal(err)
	}
	configAsset, err := Asset("riak_explorer.conf")
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(configfile, bytes.NewReader(configAsset))
}
func NewRiakExplorer(port int) (*RiakExplorer, error) {
	retFuture := make(chan *RiakExplorer)
	go startExplorer(retFuture)
	retVal := <-retFuture
	log.Info("Retval: ", retVal)
	if retVal == nil {
		err := fmt.Errorf("Unknown Error")
		return retVal, err
	} else {
		return retVal, nil
	}
}
