package main

import (
	"flag"
	"fmt"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
)

var (
	riakLibDir    string
	zookeeperAddr string
	frameworkID   string
	port          int
)

func init() {

	flag.StringVar(&zookeeperAddr, "zk", "master.mesos:2181", "Zookeeper")
	flag.StringVar(&frameworkID, "name", "riak", "Framework Instance Name")
	flag.IntVar(&port, "port", 0, "CEPMd Port")
	flag.BoolVar(&install, "install", false, "When supplied, will install beam files to a given directory (riak_lib_dir)")
	flag.StringVar(&riakLibDir, "riak_lib_dir", "root/riak/lib", "Riak lib dir")
	flag.Parse()
}

func main() {
	mgr := metamgr.NewMetadataManager(frameworkID, []string{zookeeperAddr})
	c := cepm.NewCPMd(port, mgr)
	cepm.InstallInto(riakLibDir, c.GetPort())
	fmt.Println("CEPMd running on port: ", c.GetPort())
	c.Foreground()
}
