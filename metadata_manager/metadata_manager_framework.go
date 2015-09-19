package metadata_manager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
)

func (mgr *MetadataManager) SetupFramework(URI string) {

	URIPath := makeSubSpace(mgr.namespace, "uri")
	_, err := mgr.zkConn.Create(URIPath.GetZKPath(), []byte(URI), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Panic(err)
	}

}
