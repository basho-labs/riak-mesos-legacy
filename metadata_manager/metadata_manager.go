package metadata_manager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	// "github.com/golang/protobuf/proto"
	"strings"
	"sync"
	"time"
)


type ZkNode struct {
	mgr				*MetadataManager
	stat			*zk.Stat
	data			[]byte
	ns				Namespace
}
func (node *ZkNode) GetData() []byte {
	return node.data
}
func (node *ZkNode) SetData(data []byte) {
	var err error
	log.Info("Persisting data")
	node.stat, err = node.mgr.zkConn.Set(node.ns.GetZKPath(), data, node.stat.Version)
	if err != nil {
		log.Panic("Error persisting data: ", err)
	}
}
func (node *ZkNode) GetChildren() []*ZkNode {
	return node.mgr.getChildren(node.ns)
}

func (node *ZkNode) MakeChild(name string) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := &SubNamespace{
		parent:node.ns,
		components:[]string{name},
	}
	return node.mgr.makeNode(ns)
}

func (node *ZkNode) GetChild(name string) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := &SubNamespace{
		parent:node.ns,
		components:[]string{name},
	}
	return node.mgr.getNode(ns)
}

func (node *ZkNode) CreateChildIfNotExists(name string) {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := &SubNamespace{
		parent:node.ns,
		components:[]string{name},
	}
	node.mgr.CreateNSIfNotExists(ns)
}

type Namespace interface {
	GetComponents() []string
	GetZKPath() string
}
type baseNamespace struct {
}

// Base namespace should only ever return "" -- at least for Zookeeper
func (baseNamespace) GetComponents() []string {
	return []string{""}
}

// Base namespace should only ever return "" -- at least for Zookeeper
func (baseNamespace) GetZKPath() string {
	return "/"
}

type SubNamespace struct {
	parent     Namespace
	components []string
}

// Components are read-only, so not pointer-receiver
func (ns SubNamespace) GetComponents() []string {
	return append(ns.parent.GetComponents(), ns.components...)
}
func (ns SubNamespace) GetZKPath() string {
	return strings.Join(ns.GetComponents(), "/")
}

type MetadataManager struct {
	frameworkName string
	zkConn        *zk.Conn
	namespace     Namespace
	lock          *sync.Mutex
}

func (mgr *MetadataManager) setup() {
	mgr.CreateNSIfNotExists(mgr.namespace)
}

func NewMetadataManager(frameworkName string, zookeeperAddr string) *MetadataManager {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second*10)
	if err != nil {
		panic(err)
	}
	manager := &MetadataManager{
		lock:          &sync.Mutex{},
		frameworkName: frameworkName,
		zkConn:        conn,
		namespace: SubNamespace{
			parent:     baseNamespace{},
			components: []string{"bletchley", "frameworks", frameworkName},
		},
	}
	manager.setup()
	return manager
}
func (mgr *MetadataManager) createPathIfNotExists(path string) {
	splitString := strings.Split(path, "/")
	for idx := range splitString {
		if idx == 0 {
			continue
		}
		mgr.createIfNotExists(strings.Join(splitString[0:idx+1], "/"))
	}
}

func (mgr *MetadataManager) CreateNSIfNotExists(ns Namespace) {
	components := ns.GetComponents()
	for idx := range components {
		if idx == 0 {
			continue
		}
		mgr.createIfNotExists(strings.Join(components[0:idx+1], "/"))
	}
}
func (mgr *MetadataManager) createIfNotExists(path string) {
	exists, _, err := mgr.zkConn.Exists(path)
	if err != nil {
		log.Panic(err)
	}
	if !exists {
		_, err := mgr.zkConn.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			log.Panic(err)
		}
	}
}

// This subspaces the node in the "current working namespace"
func (mgr *MetadataManager) GetRootNode() *ZkNode {
	return mgr.getNode(mgr.namespace)
}

func (mgr *MetadataManager) getChildren(ns Namespace) []*ZkNode {
	children, _, err :=  mgr.zkConn.Children(ns.GetZKPath())
	if err != nil{
		log.Panic(err)
	}
	result := make([]*ZkNode, len(children))
	for idx, name := range children {
		result[idx] = mgr.getNode(SubNamespace{parent:ns, components:[]string{name}})
	}
	return result
}

func (mgr *MetadataManager) getNode(ns Namespace) *ZkNode {
	// Namespaces are also nodes
	data, stat, err := mgr.zkConn.Get(ns.GetZKPath())
	if err != nil{
		log.Panic(err)
	}
	node := &ZkNode{
		mgr:	mgr,
		data:	data,
		stat:	stat,
		ns:		ns,
	}
	return node
}

func (mgr *MetadataManager) makeNode(ns Namespace) *ZkNode {
	// Namespaces are also nodes
	log.Info("Making node")
	_, err := mgr.zkConn.Create(ns.GetZKPath(), nil, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Panic(err)
	}
	return mgr.getNode(ns)
}
