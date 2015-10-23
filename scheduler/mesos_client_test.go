package scheduler

// import (
// 	"code.google.com/p/go-uuid/uuid"
// 	"os"
// 	"testing"
//
// 	log "github.com/Sirupsen/logrus"
// 	// "github.com/basho-labs/riak-mesos/scheduler"
// 	// "github.com/stretchr/testify/assert"
// )

// func TestReserveResourceAndCreateVolume(t *testing.T) {
// 	fo, logErr := os.Create("test.log")
// 	if logErr != nil {
// 		panic(logErr)
// 	}
// 	log.SetOutput(fo)
//
// 	frameworkID := "frameworkID"
// 	offerID := "offerID"
// 	persistenceID := uuid.NewRandom().String()
// 	log.Infof("persistenceID: %+v", persistenceID)
// 	role := "role"
// 	principal := "principal"
// 	containerPath := "volume"
//
// 	client := NewMesosClient("http://master.mesos")
// 	acceptMessage, _ := client.ReserveResourceAndCreateVolume(frameworkID, offerID, 1.0, 8000, 20000, 5, role, principal, persistenceID, containerPath)
//
// 	log.Infof("resOpJson: %+v", acceptMessage)
// }
