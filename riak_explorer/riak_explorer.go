package riak_explorer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	"github.com/basho-labs/riak-mesos/process_manager"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

type RiakExplorer struct {
	tempdir string
	port    int64
	pm      *process_manager.ProcessManager
}

type templateData struct {
	HTTPPort int64
	NodeName string
}

type advancedTemplateData struct {
	CEPMDPort int
}

func (re *RiakExplorer) NewRiakExplorerClient() *RiakExplorerClient {
	return NewRiakExplorerClient(fmt.Sprintf("localhost:%d", re.port))
}
func (re *RiakExplorer) TearDown() {
	re.pm.TearDown()
}

func decompress() string {
	assetPath := fmt.Sprintf("riak_explorer_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	log.Info("Decompressing Riak Explorer: ", assetPath)

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
	configpath := filepath.Join(".", re.tempdir, "rex_root", "riak_explorer", "etc", "riak_explorer.conf")
	file, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}
func (re *RiakExplorer) configureAdvanced(cepmdPort int) {
	data, err := Asset("advanced.config")
	if err != nil {
		log.Panic("Got error", err)
	}
	tmpl, err := template.New("advanced").Parse(string(data))

	if err != nil {
		log.Panic(err)
	}

	// Populate template data from the MesosTask
	vars := advancedTemplateData{}
	vars.CEPMDPort = cepmdPort
	configpath := filepath.Join(".", re.tempdir, "rex_root", "riak_explorer", "etc", "advanced.config")
	file, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func NewRiakExplorer(port int64, nodename string, c *cepm.CEPM) (*RiakExplorer, error) {
	tempdir := decompress()
	exepath := filepath.Join(".", tempdir, "rex_root", "riak_explorer", "bin", "riak_explorer")

	args := []string{"console", "-noinput"}
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
		tempdir: tempdir,
		port:    port,
	}
	if c != nil {
		// This is gross -- we're passing "hidden" state by passing it through the unix environment variables.
		// Fix it -- we should convert the NewRiakExplorer into using a fluent pattern?
		libpath := filepath.Join(".", tempdir, "rex_root", "riak_explorer", "lib", "basho-patches")
		os.Mkdir(libpath, 0777)
		err := cepm.InstallInto(libpath)
		if err != nil {
			log.Panic(err)
		}
		args = append(args, "-no_epmd")
		re.configureAdvanced(c.GetPort())
	}
	re.configure(port, nodename)
	log.Debugf("Starting up Riak Explorer %v", exepath)
	var err error
	chroot := filepath.Join(".", tempdir, "rex_root", "riak_explorer")
	re.pm, err = process_manager.NewProcessManager(tearDownFun, exepath, args, healthCheckFun, &chroot)
	if err != nil {
		log.Error("Could not start Riak Explorer: ", err)
	}

	return re, err
}
