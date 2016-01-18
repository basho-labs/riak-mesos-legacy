package cepm

import (
	"bufio"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type CEPM struct {
	mgr       *metamgr.MetadataManager
	cepmdNode *metamgr.ZkNode
	ln        net.Listener
	lock      *sync.Mutex
	hostname  string
}

func (c *CEPM) handleConn(conn net.Conn) {
	log.Info("Received connection: ", conn.RemoteAddr())
	defer conn.Close()
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Info("Connection error: ", err)
		return
	}
	line = strings.Trim(line, "\r\n")
	log.Infof("Received line: %s", line)
	commandAndMore := strings.Split(line, " ")
	command := commandAndMore[0]
	log.Infof("Command '%s'", command)
	if command == "REGISTER" {
		log.Infof("Received register node: %s at port: %s", commandAndMore[1], commandAndMore[2])
		var port int
		port, err = strconv.Atoi(commandAndMore[2])
		if err != nil {
			log.Info("Error parsing CEPMd command: ", err)
			return
		}
		disterlData := common.DisterlData{
			NodeName:    commandAndMore[1],
			DisterlPort: port,
		}
		nodeByteData, err := disterlData.Serialize()
		if err != nil {
			log.Info("Error serializing data: ", err)
			return
		}

		// Right now, if this fails, we blow up the entire process... Which I'm perfectly okay with...
		node, err := c.cepmdNode.MakeChildWithData(commandAndMore[1], nodeByteData, true)
		if err != nil {
			conn.Write([]byte("ERROR\n"))
			log.Info("Failed to register node in Zookeeper: ", err)
			return
		}

		conn.Write([]byte("OK\n"))
		defer node.Delete()
		// Basically waits forever, as soon as it returns, we close out
		emptyBuf := make([]byte, 1)
		conn.Read(emptyBuf)
		log.Info("Deregistering node name: ", commandAndMore[1])
	} else if command == "PORT_PLEASE" {
		log.Info("Received port_please: ", commandAndMore[1])
		zkNode, err := c.cepmdNode.GetChild(commandAndMore[1])
		if err != nil {
			log.Info("Failed to get node from Zookeeper: ", err)
			conn.Write([]byte("NOTFOUND\n"))
		} else {
			disterlData, err := common.DeserializeDisterlData(zkNode.GetData())
			if err != nil {
				log.Info("Failed to deserialize data: ", err)
			}
			conn.Write([]byte(fmt.Sprintf("%d\n", disterlData.DisterlPort)))
		}

		log.Printf("zkNode: %+v", zkNode)
	} else {
		log.Errorf("Unknown command: '%s'", command)
		return
	}
}
func (c *CEPM) GetPort() int {
	_, strPort, err := net.SplitHostPort(c.ln.Addr().String())
	if err != nil {
		log.Panic(err)
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		log.Panic(err)
	}
	return port
}
func (c *CEPM) acceptLoop() {
	defer c.ln.Close()
	log.Info("Listening on: ", c.ln.Addr())
	for {
		conn, err := c.ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go c.handleConn(conn)
	}
}
func (c *CEPM) Foreground() {
	c.acceptLoop()

}
func (c *CEPM) Background() {
	go c.acceptLoop()

}

func setupCPMd(port int) *CEPM {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))

	if err != nil {
		log.Panic("Failed to bind to port: ", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Panic("Failed resolve hostname: ", err)
	}

	c := &CEPM{
		ln:       ln,
		lock:     &sync.Mutex{},
		hostname: hostname,
	}
	return c
}
func NewCPMd(port int, mgr *metamgr.MetadataManager) *CEPM {
	c := setupCPMd(port)

	var err error
	c.mgr = mgr
	mgr.GetRootNode().CreateChildIfNotExists("cepmd")
	c.cepmdNode, err = mgr.GetRootNode().GetChild("cepmd")
	if err != nil {
		log.Panic("Could not get cepmd child")
	}

	return c
}

// Drops the ERL files into the given directory
func InstallInto(dir string) error {
	var err error

	err = RestoreAssets(dir, "")
	if err != nil {
		return err
	}

	return nil
}

func InstallIntoCli(dir string, port int) error {
	var err error

	kernelDirs, err := filepath.Glob(fmt.Sprint(dir, "/kernel*"))
	if err != nil {
		log.Fatal("Could not find kernel directory")
	}

	log.Infof("Found kernel dirs: %v", kernelDirs)

	err = RestoreAssets(fmt.Sprint(kernelDirs[0], "/ebin"), "")

	if err != nil {
		log.Panic(err)
	}
	if err := common.KillEPMD("root/riak"); err != nil {
		log.Fatal("Could not kill EPMd: ", err)
	}
	os.MkdirAll(fmt.Sprint(kernelDirs[0], "/priv"), 0777)
	ioutil.WriteFile(fmt.Sprint(kernelDirs[0], "/priv/cepmd_port"), []byte(fmt.Sprintf("%d.", port)), 0777)

	log.Infof("Port written to %v", fmt.Sprint(kernelDirs[0], "/priv/cepmd_port"))

	return nil
}
