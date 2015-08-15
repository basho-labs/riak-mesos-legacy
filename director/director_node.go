package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/process_manager"

	"bytes"
	"errors"
)

type DirectorNode struct {
	finishChan chan interface{}
	pm         *process_manager.ProcessManager
	running    bool
}

func NewDirectorNode() *DirectorNode {
	decompress()
	return &DirectorNode{
		running: false,
	}
}

func (directorNode *DirectorNode) runLoop() {
	waitChan := directorNode.pm.Listen()
	select {
	case <-waitChan:
		{
			log.Error("Director Died, failing")
		}
	case <-directorNode.finishChan:
		{
			log.Info("Finish channel says to shut down Director")
			directorNode.pm.TearDown()
		}
	}
	time.Sleep(15 * time.Second)
	log.Info("Shutting down")
}

func decompress() string {
	assetPath := fmt.Sprintf("riak_mesos_director_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	log.Info("Decompressing Riak Mesos Director: ", assetPath)

	asset, err := Asset(assetPath)
	if err != nil {
		log.Fatal(err)
	}
	bytesReader := bytes.NewReader(asset)

	gzReader, err := gzip.NewReader(bytesReader)

	if err != nil {
		log.Fatal("Error encountered decompressing riak explorer: ", err)
		os.Exit(1)
	}

	tr := tar.NewReader(gzReader)
	tempdir := "./"

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
			// Hard link
		} else if hdr.Typeflag == tar.TypeLink {
			fmt.Printf("Encountered hardlink: %+v\n", hdr)
			linkdest := filepath.Join(".", tempdir, hdr.Linkname)
			if err := os.Link(linkdest, filename); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("Experienced unknown tar file type: ", hdr.Typeflag)
		}
	}
	return tempdir
}

func (directorNode *DirectorNode) Run() {

	var err error

	args := []string{"console", "-noinput"}

	wd, err := os.Getwd()
	if err != nil {
		log.Panic("Could not get wd: ", err)
	}
	chroot := filepath.Join(wd, "director_root")

	HealthCheckFun := func() error {
		log.Info("Checking is Director is started")
		data, err := ioutil.ReadFile("director_root/riak_mesos_director/log/console.log")
		if err != nil {
			if bytes.Contains(data, []byte("lager started on node")) {
				log.Info("Director started")
				return nil
			} else {
				return errors.New("Director not yet started")
			}
		} else {
			return err
		}
	}
	directorNode.pm, err = process_manager.NewProcessManager(func() { return }, "/riak_mesos_director/bin/director", args, HealthCheckFun, &chroot)

	if err != nil {
		log.Error("Could not start Director: ", err)
		time.Sleep(15 * time.Minute)
		log.Info("Shutting down due to GC, after failing to bring up Director node")
	} else {
		directorNode.running = true
		directorNode.runLoop()
	}
}
