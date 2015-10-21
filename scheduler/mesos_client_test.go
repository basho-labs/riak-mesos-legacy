package scheduler

import (
	"encoding/json"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	// "github.com/stretchr/testify/assert"
)

const (
	CPUS_PER_TASK = 0.1
	MEM_PER_TASK  = 32
	DISK_PER_TASK = 16
)

func TestReserve(t *testing.T) {
	fo, logErr := os.Create("test.log")
	if logErr != nil {
		panic(logErr)
	}
	log.SetOutput(fo)

	role := "string"
	principal := "principal"
	persistenceId := "persistenceId"
	volume := "volume"
	reservation := taskReservations(&role, &principal, &persistenceId, &volume)
	reservationJsonBytes, _ := json.Marshal(reservation)
	reservationJson := string(reservationJsonBytes)
	log.Infof("Reservation: %+v", reservationJson)
}

func shardInitialResources(role *string) []*mesos.Resource {
	cpus := util.NewScalarResource("cpus", 0.1)
	cpus.Role = role
	mem := util.NewScalarResource("mem", 32)
	mem.Role = role
	disk := util.NewScalarResource("disk", 16)
	disk.Role = role

	var resources []*mesos.Resource
	resources = make([]*mesos.Resource, 3)
	resources[0] = cpus
	resources[1] = mem
	resources[2] = disk

	return resources
}

func taskPersistentVolume(persistenceID *string, containerPath *string) *mesos.Resource {

	mode := mesos.Volume_RW
	volume := &mesos.Volume{
		ContainerPath: containerPath,
		Mode:          &mode,
	}

	persistence := &mesos.Resource_DiskInfo_Persistence{
		Id: persistenceID,
	}

	info := &mesos.Resource_DiskInfo{
		Persistence: persistence,
		Volume:      volume,
	}

	resource := util.NewScalarResource("disk", DISK_PER_TASK)
	resource.Disk = info

	return resource
}

func taskReservations(role *string, principal *string, persistenceId *string, containerPath *string) []*mesos.Resource {
	log.Infof(*role, *principal, *persistenceId, *containerPath)
	resources := []*mesos.Resource{}

	// reservation := &mesos.Resource_ReservationInfo{
	// 	Principal: principal,
	// }

	cpus := util.NewScalarResource("cpus", CPUS_PER_TASK)
	cpus.Role = role
	// cpus.Reservation = reservation

	mem := util.NewScalarResource("mem", MEM_PER_TASK)
	mem.Role = role
	// mem.Reservation = reservation

	disk := taskPersistentVolume(persistenceId, containerPath)
	disk.Role = role
	// disk.Reservation = reservation

	resources = append(resources, cpus)
	resources = append(resources, mem)
	resources = append(resources, disk)

	return resources
}

func reserveOperation(reservations []*mesos.Resource) *mesos.Offer_Operation {
	reserve := &mesos.Offer_Operation_Reserve{
		Resources: reservations,
	}
	operationType := mesos.Offer_Operation_RESERVE
	operation := &mesos.Offer_Operation{
		Type:    &operationType,
		Reserve: reserve,
	}

	return operation
}

func createOperation(volumes []*mesos.Resource) *mesos.Offer_Operation {
	create := &mesos.Offer_Operation_Create{
		Volumes: volumes,
	}
	operationType := mesos.Offer_Operation_CREATE
	operation := &mesos.Offer_Operation{
		Type:   &operationType,
		Create: create,
	}

	return operation
}

func launchOperation(tasks []*mesos.TaskInfo) *mesos.Offer_Operation {
	launch := &mesos.Offer_Operation_Launch{
		TaskInfos: tasks,
	}
	operationType := mesos.Offer_Operation_LAUNCH
	operation := &mesos.Offer_Operation{
		Type:   &operationType,
		Launch: launch,
	}

	return operation
}
