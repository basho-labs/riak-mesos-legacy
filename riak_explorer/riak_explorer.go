package riak_explorer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
)

type RiakExplorer struct {
	tempdir  string
	exe      *exec.Cmd
	waitChan chan interface{}
	teardown chan chan interface{}
	port	 int64
}

type templateData struct {
	HTTPPort int64
}

func startExplorer(port int64, retChan chan *RiakExplorer) {

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
		port:	  port,
	}

	defer close(re.teardown)

	re.configure(port)
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
				_, err := NewRiakExplorerClient(fmt.Sprintf("localhost:%d", port)).Ping()
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
func (re *RiakExplorer) NewRiakExplorerClient() *RiakExplorerClient {
	return NewRiakExplorerClient(fmt.Sprintf("localhost:%d", re.port))
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
	assetPath := fmt.Sprintf("riak_explorer_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	asset, err := Asset(assetPath)
	if err != nil {
		log.Fatal(err)
	}
	bytesReader := bytes.NewReader(asset)

	gzReader, err := gzip.NewReader(bytesReader)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tr := tar.NewReader(gzReader)
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
func (re *RiakExplorer) configure(port int64) {
	data, err := Asset("riak_explorer.conf")
	if err != nil {
		log.Panic("Got error", err)
	}
	tmpl, err := template.New("test").Parse(string(data))

	if err != nil {
		log.Panic(err)
	}

	// Populate template data from the MesosTask
	vars := templateData{}


	vars.HTTPPort = port
	configpath := filepath.Join(".", re.tempdir, "riak_explorer", "etc", "riak_explorer.conf")
	file, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func NewRiakExplorer(port int64) (*RiakExplorer, error) {
	retFuture := make(chan *RiakExplorer)
	go startExplorer(port, retFuture)
	retVal := <-retFuture
	log.Info("Retval: ", retVal)
	if retVal == nil {
		err := fmt.Errorf("Unknown Error")
		return retVal, err
	} else {
		return retVal, nil
	}
}
