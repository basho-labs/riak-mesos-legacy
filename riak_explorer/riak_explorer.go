package riak_explorer

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/artifacts"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	"github.com/basho-labs/riak-mesos/common"
	"github.com/basho-labs/riak-mesos/process_manager"

	"os"
	"path/filepath"
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

func decompress() {
	log.Info("Decompressing Riak Explorer")

	var err error
	if err := os.Mkdir("riak_explorer", 0777); err != nil {
		log.Fatal("Unable to make rex directory: ", err)
	}

	asset, err := artifacts.Asset("trusty.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	if err = common.ExtractGZ("riak_explorer", bytes.NewReader(asset)); err != nil {
		log.Fatal("Unable to extract trusty root: ", err)
	}
	asset, err = artifacts.Asset("riak_explorer-bin.tar.gz")

	if err != nil {
		log.Fatal(err)
	}
	if err = common.ExtractGZ("riak_explorer", bytes.NewReader(asset)); err != nil {
		log.Fatal("Unable to extract rex: ", err)
	}

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
	configpath := filepath.Join(".", "riak_explorer", "riak_explorer", "etc", "riak_explorer.conf")
	file, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	defer file.Close()
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
	configpath := filepath.Join(".", "riak_explorer", "riak_explorer", "etc", "advanced.config")
	file, err := os.OpenFile(configpath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	defer file.Close()
	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func NewRiakExplorer(port int64, nodename string, c *cepm.CEPM) (*RiakExplorer, error) {
	decompress()
	exepath := "/riak_explorer/bin/riak_explorer"

	var err error

	args := []string{"console", "-noinput"}
	healthCheckFun := func() error {
		log.Info("Running healthcheck: ", port)
		_, err := NewRiakExplorerClient(fmt.Sprintf("localhost:%d", port)).Ping()
		log.Info("Healthcheck result ", err)
		return err
	}
	tearDownFun := func() {
		log.Info("Tearing down riak explorer")
		//err := os.RemoveAll("riak_explorer")
		//if err != nil {
		///		log.Error(err)
		//}
	}
	re := &RiakExplorer{
		port: port,
	}
	if c != nil {
		// This is gross -- we're passing "hidden" state by passing it through the unix environment variables.
		// Fix it -- we should convert the NewRiakExplorer into using a fluent pattern?
		libpath := filepath.Join(".", "riak_explorer", "riak_explorer", "lib", "basho-patches")
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

	chroot := filepath.Join(".", "riak_explorer")
	re.pm, err = process_manager.NewProcessManager(tearDownFun, exepath, args, healthCheckFun, &chroot)
	if err != nil {
		log.Error("Could not start Riak Explorer: ", err)
	}

	return re, err
}
