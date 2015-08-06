package main

import (
	"flag"
	"fmt"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
)

var (
	zookeeperAddr string
	frameworkName string
	port          int
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&frameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.IntVar(&port, "port", 0, "CEPMd Port")
	flag.Parse()
}

func main() {
	mgr := metamgr.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	c := cepm.NewCPMd(port, mgr)
	fmt.Println("CEPMd running on port: ", c.GetPort())
	c.Foreground()

}
