package metadata_manager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	// "github.com/golang/protobuf/proto"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ZkNode struct {
	mgr  *MetadataManager
	stat *zk.Stat
	data []byte
	ns   Namespace
}

func (node *ZkNode) String() string {
	return fmt.Sprintf("<%s> -> %v", node.ns.GetZKPath(), node.data)
}
func (node *ZkNode) GetData() []byte {
	return node.data
}
func (node *ZkNode) GetLock() *zk.Lock {
	zkLock := zk.NewLock(node.mgr.zkConn, node.ns.GetZKPath(), zk.WorldACL(zk.PermAll))
	return zkLock
}
func (node *ZkNode) SetData(data []byte) {
	var err error
	log.Info("Persisting data")
	if node.stat != nil {
		node.stat, err = node.mgr.zkConn.Set(node.ns.GetZKPath(), data, node.stat.Version)
		if err != nil {
			log.Panic("Error persisting data: ", err)
		}
	} else {
		_, err = node.mgr.zkConn.Create(node.ns.GetZKPath(), data, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			log.Panic("Error persisting data: ", err)
		}
		node.data, node.stat, err = node.mgr.zkConn.Get(node.ns.GetZKPath())
		if err != nil {
			log.Panic("Error persisting data: ", err)
		}
	}
}
func (node *ZkNode) GetChildren() []*ZkNode {
	return node.mgr.getChildren(node.ns)
}

func (node *ZkNode) GetChildrenW() ([]*ZkNode, <-chan zk.Event) {
	return node.mgr.getChildrenW(node.ns)
}

func (node *ZkNode) MakeEmptyChild(name string) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	newNode := &ZkNode{
		mgr: node.mgr,
		ns:  ns,
	}
	return newNode
}
func (node *ZkNode) MakeChild(name string, ephemeral bool) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	return node.mgr.makeNode(ns, ephemeral)
}

func (node *ZkNode) MakeChildWithData(name string, data []byte, ephemeral bool) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	return node.mgr.makeNodeWithData(ns, data, ephemeral)
}

func (node *ZkNode) GetChild(name string) *ZkNode {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	return node.mgr.getNode(ns)
}

func (node *ZkNode) CreateChildIfNotExists(name string) {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	node.mgr.CreateNSIfNotExists(ns, false)
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
	parent    Namespace
	component string
}

// Components are read-only, so not pointer-receiver
func (ns SubNamespace) GetComponents() []string {
	return append(ns.parent.GetComponents(), ns.component)
}
func (ns SubNamespace) GetZKPath() string {
	return strings.Join(ns.GetComponents(), "/")
}
func makeSubSpace(ns Namespace, subSpaceName string) Namespace {
	return SubNamespace{parent: ns, component: subSpaceName}
}

type MetadataManager struct {
	framework     MetadataManagerFramework
	frameworkName string
	zkConn        *zk.Conn
	namespace     Namespace
	lock          *sync.Mutex
	zkLock        zk.Lock
}

func (mgr *MetadataManager) setup() {
	mgr.CreateNSIfNotExists(mgr.namespace, false)
}

func NewMetadataManager(frameworkName string, zookeepers []string) *MetadataManager {
	conn, _, err := zk.Connect(zookeepers, time.Second)
	if err != nil {
		log.Panic(err)
	}
	bns := baseNamespace{}
	ns := makeSubSpace(makeSubSpace(makeSubSpace(bns, "riak"), "frameworks"), frameworkName)
	lockPath := makeSubSpace(ns, "lock")
	zkLock := zk.NewLock(conn, lockPath.GetZKPath(), zk.WorldACL(zk.PermAll))

	manager := &MetadataManager{
		lock:          &sync.Mutex{},
		frameworkName: frameworkName,
		zkConn:        conn,
		namespace:     ns,
		zkLock:        *zkLock,
	}

	manager.setup()
	return manager
}
func (mgr *MetadataManager) createPathIfNotExists(path string, ephemeral bool) {
	splitString := strings.Split(path, "/")
	for idx := range splitString {
		if idx == 0 {
			continue
		}
		mgr.createIfNotExists(strings.Join(splitString[0:idx+1], "/"), ephemeral)
	}
}

func (mgr *MetadataManager) CreateNSIfNotExists(ns Namespace, ephemeral bool) {
	components := ns.GetComponents()
	for idx := range components {
		if idx == 0 {
			continue
		}
		mgr.createIfNotExists(strings.Join(components[0:idx+1], "/"), ephemeral)
	}
}
func (mgr *MetadataManager) createIfNotExists(path string, ephemeral bool) {
	exists, _, err := mgr.zkConn.Exists(path)
	if err != nil {
		log.Panic(err)
	}
	if !exists {
		var err error
		if ephemeral {
			_, err = mgr.zkConn.Create(path, nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		} else {
			_, err = mgr.zkConn.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
		}
		if err != nil {
			log.Panic(err)
		}
	}
}

// This subspaces the node in the "current working namespace"
func (mgr *MetadataManager) GetRootNode() *ZkNode {
	return mgr.getNode(mgr.namespace)
}

func (mgr *MetadataManager) getChildrenW(ns Namespace) ([]*ZkNode, <-chan zk.Event) {
	children, _, watchChan, err := mgr.zkConn.ChildrenW(ns.GetZKPath())
	if err != nil {
		log.Panic(err)
	}
	result := make([]*ZkNode, len(children))
	for idx, name := range children {
		result[idx] = mgr.getNode(makeSubSpace(ns, name))
	}
	return result, watchChan
}
func (mgr *MetadataManager) getChildren(ns Namespace) []*ZkNode {
	children, _, err := mgr.zkConn.Children(ns.GetZKPath())
	if err != nil {
		log.Panic(err)
	}
	result := make([]*ZkNode, len(children))
	for idx, name := range children {
		result[idx] = mgr.getNode(makeSubSpace(ns, name))
	}
	return result
}

func (mgr *MetadataManager) getNode(ns Namespace) *ZkNode {
	// Namespaces are also nodes
	data, stat, err := mgr.zkConn.Get(ns.GetZKPath())
	if err != nil {
		log.Panic(err)
	}
	node := &ZkNode{
		mgr:  mgr,
		data: data,
		stat: stat,
		ns:   ns,
	}
	return node
}

func (mgr *MetadataManager) makeNode(ns Namespace, ephemeral bool) *ZkNode {
	var flags int32
	if ephemeral {
		flags = zk.FlagEphemeral
	} else {
		flags = 0
	}
	// Namespaces are also nodes
	log.Info("Making node")
	_, err := mgr.zkConn.Create(ns.GetZKPath(), nil, flags, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Panic(err)
	}
	return mgr.getNode(ns)
}

func (mgr *MetadataManager) makeNodeWithData(ns Namespace, data []byte, ephemeral bool) *ZkNode {
	var flags int32
	if ephemeral {
		flags = zk.FlagEphemeral
	} else {
		flags = 0
	}
	// Namespaces are also nodes
	log.Info("Making node")
	_, err := mgr.zkConn.Create(ns.GetZKPath(), data, flags, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Panic(err)
	}
	return mgr.getNode(ns)
}
