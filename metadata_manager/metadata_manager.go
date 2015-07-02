package metadata_manager

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
	"github.com/satori/go.uuid"
	// "flag"
)

// var (
// 	zookeeper	string
// )

// func init() {
// 	flag.StringVar(&zookeeper, "zk", "ubuntu:2181", "Zookeeper")
// 	flag.Parse()
// }

type MetadataManager struct {
	frameworkName           string
	setTaskUUIDChan 		chan setTaskUUIDRequest
	getTaskUUIDChan  	    chan getTaskUUIDRequest
	zkConn                  *zk.Conn
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
	path := fmt.Sprintf("/bletchley/frameworks/%s/tasks", mgr.frameworkName)
	mgr.createPathIfNotExists(path)
	for {
		select {
		case rq := <-mgr.setTaskUUIDChan: mgr.setTaskUUID(rq)
		case rq := <-mgr.getTaskUUIDChan: mgr.getTaskUUID(rq)

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
