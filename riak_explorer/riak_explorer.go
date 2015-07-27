package riak_explorer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
	"github.com/basho-labs/riak-mesos/process_manager"
	log "github.com/Sirupsen/logrus"
)

type RiakExplorer struct {
	tempdir  string
	port     int64
	pm		 *process_manager.ProcessManager

}

type templateData struct {
	HTTPPort int64
	NodeName string
}

func (re *RiakExplorer) NewRiakExplorerClient() *RiakExplorerClient {
	return NewRiakExplorerClient(fmt.Sprintf("localhost:%d", re.port))
}
func (re *RiakExplorer) TearDown() {
	re.pm.TearDown()
}

func decompress() string {
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
	return tempdir
}
func (re *RiakExplorer) configure(port int64, nodename string) {
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

	vars.NodeName = nodename
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

func NewRiakExplorer(port int64, nodename string) (*RiakExplorer, error) {
	tempdir := decompress()
	exepath := filepath.Join(".", tempdir, "riak_explorer", "bin", "riak_explorer")

	healthCheckFun := func() error {
		log.Info("Running healthcheck: ", port)
		_, err := NewRiakExplorerClient(fmt.Sprintf("localhost:%d", port)).Ping()
		log.Info("Healthcheck result ", err)
		return err
	}
	tearDownFun := func() {
		log.Info("Deleting all data in: ", tempdir)
		err := os.RemoveAll(tempdir)
		if err != nil {
			log.Error(err)
		}
	}
	re := &RiakExplorer{
		tempdir:  tempdir,
		port:     port,
	}
	re.configure(port, nodename)
	log.Debugf("Starting up Riak Exploer %v", exepath)
	var err error
	re.pm, err = process_manager.NewProcessManager(tearDownFun, exepath, []string{"console", "-noinput"}, healthCheckFun)
	if err != nil {
		log.Error("Could not start Riak Explorer: ", err)
	}

	return re, err
}
