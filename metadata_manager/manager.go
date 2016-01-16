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

// TODO: Convert ZKNode functions to all work around MetadataNode interface for better testing
type MetadataNode interface {
}
type ZkNode struct {
	mgr  *MetadataManager
	stat *zk.Stat
	data []byte
	ns   Namespace
}

func (node *ZkNode) Delete() {
	node.mgr.DeleteChildren(node.ns.GetZKPath())
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
func (node *ZkNode) SetData(data []byte) error {
	return node.SetDataWithRetry(data, 0, 10)
}
func (node *ZkNode) SetDataWithRetry(data []byte, currentRetry int, retry int) error {
	var err error
	log.Info("Persisting data")
	if node.stat != nil {
		node.data, node.stat, err = node.mgr.zkConn.Get(node.ns.GetZKPath())
		if err == nil {
			node.stat, err = node.mgr.zkConn.Set(node.ns.GetZKPath(), data, node.stat.Version)
		}
	} else {
		_, err = node.mgr.zkConn.Create(node.ns.GetZKPath(), data, 0, zk.WorldACL(zk.PermAll))
		if err == nil {
			node.data, node.stat, err = node.mgr.zkConn.Get(node.ns.GetZKPath())
		}
	}

	if err != nil && currentRetry >= retry {
		log.Panic("Error persisting data: ", err)
		return err
	}
	if err != nil {
		log.Warning("Error persisting data, retrying: ", err)
		node.mgr.CreateConnection()
		return node.SetDataWithRetry(data, currentRetry + 1, retry)
	}

	return nil
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
func (node *ZkNode) MakeChild(name string, ephemeral bool) (*ZkNode, error) {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	return node.mgr.makeNode(ns, ephemeral)
}

func (node *ZkNode) MakeChildWithData(name string, data []byte, ephemeral bool) (*ZkNode, error) {
	if strings.Contains(name, "/") {
		panic("Error, name of subnode cannot contain /")
	}
	ns := makeSubSpace(node.ns, name)
	return node.mgr.makeNodeWithData(ns, data, ephemeral)
}

func (node *ZkNode) GetChild(name string) (*ZkNode, error) {
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
	frameworkID string
	zkConn      *zk.Conn
	namespace   Namespace
	lock        *sync.Mutex
	zkLock      zk.Lock
	zookeepers  []string
}

func (mgr *MetadataManager) setup() {
	mgr.CreateNSIfNotExists(mgr.namespace, false)
}

func NewMetadataManager(frameworkID string, zookeepers []string) *MetadataManager {
	manager := &MetadataManager{
		lock:        &sync.Mutex{},
		frameworkID: frameworkID,
	  zookeepers:  zookeepers,
	}

	manager.CreateConnection()

	manager.setup()
	return manager
}

func (mgr *MetadataManager) DeleteChildren(path string) {
	mgr.DeleteChildrenWithRetry(path, 0, 10)
}
func (mgr *MetadataManager) DeleteChildrenWithRetry(path string, currentRetry int, retry int) {
	children, _, _ := mgr.zkConn.Children(path)

	// Leaf
	if len(children) == 0 {
		fmt.Println("Deleting ", path)
		err := mgr.zkConn.Delete(path, -1)
		if err != nil && currentRetry >= retry {
			log.Panic(err)
		}
		if err != nil {
			log.Warning(err)
			mgr.DeleteChildrenWithRetry(path, currentRetry + 1, retry)
		} else {
			return
		}
	}

	// Branches
	for _, name := range children {
		mgr.DeleteChildrenWithRetry(path + "/" + name, currentRetry, retry)
	}

	return
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

func (mgr *MetadataManager) CreateConnection() {
	conn, _, err := zk.Connect(mgr.zookeepers, time.Second)
	if err != nil {
		log.Panic(err)
	}
	bns := baseNamespace{}
	ns := makeSubSpace(makeSubSpace(makeSubSpace(bns, "riak"), "frameworks"), mgr.frameworkID)
	lockPath := makeSubSpace(ns, "lock")
	zkLock := zk.NewLock(conn, lockPath.GetZKPath(), zk.WorldACL(zk.PermAll))

	mgr.zkConn = conn
	mgr.namespace = ns
	mgr.zkLock = *zkLock
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
	mgr.createIfNotExistsWithRetry(path, ephemeral, 0, 10)
}
func (mgr *MetadataManager) createIfNotExistsWithRetry(path string, ephemeral bool, currentRetry int, retry int) {
	exists, _, err := mgr.zkConn.Exists(path)
	if err == nil {
		if !exists {
			if ephemeral {
				_, err = mgr.zkConn.Create(path, nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			} else {
				_, err = mgr.zkConn.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
			}
		}
	}

	if err != nil && currentRetry >= retry {
		log.Panic(err)
	}
	if err != nil {
		log.Warning(err)
		mgr.createIfNotExistsWithRetry(path, ephemeral, currentRetry + 1, retry)
	}
}

// This subspaces the node in the "current working namespace"
func (mgr *MetadataManager) GetRootNode() *ZkNode {
	return mgr.GetRootNodeWithRetry(0, 10)
}
func (mgr *MetadataManager) GetRootNodeWithRetry(currentRetry int, retry int) *ZkNode {
	node, err := mgr.getNode(mgr.namespace)
	if err != nil && currentRetry >= retry {
		log.Panic("Could not get Root node: ", err)
	}
	if err != nil {
		log.Warning("Could not get Root node: ", err)
		return mgr.GetRootNodeWithRetry(currentRetry + 1, retry)
	} else {
		return node
	}
}

func (mgr *MetadataManager) getChildrenW(ns Namespace) ([]*ZkNode, <-chan zk.Event) {
	return mgr.getChildrenWWithRetry(ns, 0, 10)
}
func (mgr *MetadataManager) getChildrenWWithRetry(ns Namespace, currentRetry int, retry int) ([]*ZkNode, <-chan zk.Event) {
	children, _, watchChan, err := mgr.zkConn.ChildrenW(ns.GetZKPath())
	var result []*ZkNode
	if err == nil {
		result = make([]*ZkNode, len(children))
		for idx, name := range children {
			result[idx], err = mgr.getNode(makeSubSpace(ns, name))
			if err != nil {
				break
			}
		}
	}

	if err != nil && currentRetry >= retry {
		log.Panic(err)
	}
	if err != nil {
		log.Warning(err)
		return mgr.getChildrenWWithRetry(ns, currentRetry + 1, retry)
	} else {
		return result, watchChan
	}
}

func (mgr *MetadataManager) getChildren(ns Namespace) []*ZkNode {
	return mgr.getChildrenWithRetry(ns, 0, 10)
}
func (mgr *MetadataManager) getChildrenWithRetry(ns Namespace, currentRetry int, retry int) []*ZkNode {
	children, _, err := mgr.zkConn.Children(ns.GetZKPath())
	var result []*ZkNode
	if err == nil {
		result = make([]*ZkNode, len(children))
		for idx, name := range children {
			result[idx], err = mgr.getNode(makeSubSpace(ns, name))
			if err != nil {
				break
			}
		}
	}
	if err != nil && currentRetry >= retry {
		log.Panic(err)
	}
	if err != nil {
		log.Warning(err)
		return mgr.getChildrenWithRetry(ns, currentRetry + 1, retry)
	} else {
		return result
	}
}

func (mgr *MetadataManager) getNode(ns Namespace) (*ZkNode, error) {
	return mgr.getNodeWithRetry(ns, 0, 10)
}

func (mgr *MetadataManager) getNodeWithRetry(ns Namespace, currentRetry int, retry int) (*ZkNode, error) {
	// Namespaces are also nodes
	data, stat, err := mgr.zkConn.Get(ns.GetZKPath())
	if err != nil && currentRetry >= retry {
		return nil, err
	}
	if err != nil {
		mgr.CreateConnection()
		log.Warning(err)
		return mgr.getNodeWithRetry(ns, currentRetry + 1, retry)
	} else {
		node := &ZkNode{
			mgr:  mgr,
			data: data,
			stat: stat,
			ns:   ns,
		}
		return node, nil
	}
}

func (mgr *MetadataManager) makeNode(ns Namespace, ephemeral bool) (*ZkNode, error) {
	return mgr.makeNodeWithRetry(ns, ephemeral, 0, 10)
}

func (mgr *MetadataManager) makeNodeWithRetry(ns Namespace, ephemeral bool, currentRetry int, retry int) (*ZkNode, error) {
	var flags int32
	if ephemeral {
		flags = zk.FlagEphemeral
	} else {
		flags = 0
	}
	// Namespaces are also nodes
	log.Info("Making node")
	_, err := mgr.zkConn.Create(ns.GetZKPath(), nil, flags, zk.WorldACL(zk.PermAll))
	if err != nil && currentRetry >= retry {
		log.Panic(err)
	}
	if err != nil {
		mgr.CreateConnection()
		log.Warning(err)
		return mgr.makeNodeWithRetry(ns, ephemeral, currentRetry + 1, retry)
	} else {
		return mgr.getNode(ns)
	}
}

func (mgr *MetadataManager) makeNodeWithData(ns Namespace, data []byte, ephemeral bool) (*ZkNode, error) {
	return mgr.makeNodeWithDataWithRetry(ns, data, ephemeral, 0, 10)
}

func (mgr *MetadataManager) makeNodeWithDataWithRetry(ns Namespace, data []byte, ephemeral bool, currentRetry int, retry int) (*ZkNode, error) {
	var flags int32
	if ephemeral {
		flags = zk.FlagEphemeral
	} else {
		flags = 0
	}
	// Namespaces are also nodes
	log.Info("Making node")
	_, err := mgr.zkConn.Create(ns.GetZKPath(), data, flags, zk.WorldACL(zk.PermAll))
	if err != nil && currentRetry >= retry {
		log.Panic(err)
	}
	if err != nil {
		mgr.CreateConnection()
		log.Warning(err)
		return mgr.makeNodeWithDataWithRetry(ns, data, ephemeral, currentRetry + 1, retry)
	} else {
		return mgr.getNode(ns)
	}
}
