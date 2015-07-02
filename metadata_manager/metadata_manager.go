package metadata_manager

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
	"github.com/satori/go.uuid"
)

type MetadataManager struct {
	frameworkName           string
	setTaskUUIDChan 		chan setTaskUUIDRequest
	getTaskUUIDChan  	    chan getTaskUUIDRequest
	addClusterChan			chan addClusterRequest
	zkConn                  *zk.Conn
}

type addClusterRequest struct {
	clusterName				string
	replyChannel			chan bool
}

func (msg *addClusterRequest) Reply(response bool) {
	msg.replyChannel <- response
}

type setTaskUUIDRequest struct {
	oldUUID			string
	taskName     	string
	replyChannel	chan string
}

func (msg *setTaskUUIDRequest) Reply(response string) {
	msg.replyChannel <- response
}

type getTaskUUIDRequest struct {
	taskName     	string
	replyChannel	chan string
}


func (msg *getTaskUUIDRequest) Reply(response string) {
	msg.replyChannel <- response
}

func NewMetadataManager(frameworkName string, zookeeperAddr string) *MetadataManager {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second*10)
	if err != nil {
		panic(err)
	}
	manager := &MetadataManager{
		frameworkName:           frameworkName,
		setTaskUUIDChan:         make(chan setTaskUUIDRequest, 1),
		getTaskUUIDChan:         make(chan getTaskUUIDRequest, 1),
		addClusterChan:			 make(chan addClusterRequest, 1),
		zkConn:                  conn,
	}

	go manager.loop()
	return manager
}
func (mgr *MetadataManager) createPathIfNotExists(path string) {
	splitString := strings.Split(path, "/")
	for idx := range splitString {
		if idx == 0 { continue }
		mgr.createIfNotExists(strings.Join(splitString[0:idx+1], "/"))
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
func (mgr *MetadataManager) loop() {
	defer close(mgr.setTaskUUIDChan)
	defer close(mgr.getTaskUUIDChan)
	tasksPath := fmt.Sprintf("/bletchley/frameworks/%s/tasks", mgr.frameworkName)
	mgr.createPathIfNotExists(tasksPath)
	clustersPath := fmt.Sprintf("/bletchley/frameworks/%s/clusters", mgr.frameworkName)
	mgr.createPathIfNotExists(clustersPath)
	children, _, clusterEventChannel, err := mgr.zkConn.ChildrenW("/bletchley/frameworks/%s/clusters")

	if err != nil {
		log.Panic(err)
	}
	for child := range children {
		log.Info("Saw child: ", child)
	}

	for {
		select {
		case rq := <-mgr.setTaskUUIDChan: mgr.setTaskUUID(rq)
		case rq := <-mgr.getTaskUUIDChan: mgr.getTaskUUID(rq)
		case event := <- clusterEventChannel: { log.Info("Got cluster event: ", event) }
		case rq := <-mgr.addClusterChan: { log.Panic("not yet implemented: ", rq) }
		}
	}
}

func (mgr *MetadataManager) getTaskUUID(rq getTaskUUIDRequest) {
	defer close(rq.replyChannel)

	path := fmt.Sprintf("/bletchley/frameworks/%s/tasks/%s/uuid", mgr.frameworkName, rq.taskName)

	exists, _, err := mgr.zkConn.Exists(path)
	if err != nil {
		log.Panic(err)
	}

	if exists {
		data, _, err := mgr.zkConn.Get(path)
		if err != nil {
			log.Panic(err)
		}
		zkUUID, err := uuid.FromBytes(data)
		if err != nil {
			log.Panic(err)
		}
		rq.Reply(zkUUID.String())
	} else {
		taskBasePath := fmt.Sprintf("/bletchley/frameworks/%s/tasks/%s", mgr.frameworkName, rq.taskName)
		mgr.createPathIfNotExists(taskBasePath)
		uuid := uuid.NewV4()
		_, err := mgr.zkConn.Create(path, uuid.Bytes(), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			log.Panic(err)
		}
		rq.Reply(uuid.String())
	}
}

func (mgr *MetadataManager) setTaskUUID(rq setTaskUUIDRequest) {
	defer close(rq.replyChannel)
	path := fmt.Sprintf("/bletchley/frameworks/%s/tasks/%s/uuid", mgr.frameworkName, rq.taskName)
	data, stat, err := mgr.zkConn.Get(path)
	if err != nil {
		log.Panic(err)
	}
	oldZKUUID, err := uuid.FromBytes(data)
	if err != nil {
		log.Panic(err)
	}
	oldTaskUUID, err := uuid.FromString(rq.oldUUID)
	if err != nil {
		log.Panic(err)
	}

	if !uuid.Equal(oldZKUUID, oldTaskUUID) { log.Panic("UUIDs not equal") }

	newUUID := uuid.NewV4()
	_, err = mgr.zkConn.Set(path, newUUID.Bytes(), stat.Version)
	if err != nil {
		log.Panic(err)
	}
	rq.Reply(newUUID.String())
}

func (mgr *MetadataManager) GetTaskUUID(taskName string) string {
	rq := getTaskUUIDRequest{
		replyChannel: make(chan string),
		taskName:     taskName,
	}
	mgr.getTaskUUIDChan <- rq
	retval := <-rq.replyChannel
	return retval
}

func (mgr *MetadataManager) SetTaskUUID(taskName string, oldUUID string) string {
	rq := setTaskUUIDRequest{
		replyChannel: make(chan string),
		taskName:     taskName,
		oldUUID:	  oldUUID,
	}
	mgr.setTaskUUIDChan <- rq
	retval := <-rq.replyChannel
	return retval
}



func (mgr *MetadataManager) AddCluster(clusterName string) bool {
	rq := addClusterRequest{
		replyChannel: 	make(chan bool),
		clusterName:	clusterName,
	}
	mgr.addClusterChan <- rq
	retval := <-rq.replyChannel
	return retval
}