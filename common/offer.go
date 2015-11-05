package common

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"math/rand"
)

type ResourceGroup struct {
	Cpus  float64
	Mem   float64
	Disk  float64
	Ports []int64
}

type OfferHelper struct {
	MesosOffer          *mesos.Offer
	OfferIDStr          string
	PersistenceIDs      []string
	ReservedResources   *ResourceGroup
	UnreservedResources *ResourceGroup
	ResourcesToReserve  []*mesos.Resource
	ResourcesToUneserve []*mesos.Resource
	VolumesToCreate     []*mesos.Resource
	VolumesToDestroy    []*mesos.Resource
	TasksToLaunch       []*mesos.TaskInfo
}

func NewOfferHelper(mesosOffer *mesos.Offer) *OfferHelper {
	unreservedCpus, unreservedMem, unreservedDisk, unreservedPorts := getUnreservedResources(mesosOffer.Resources)
	reservedCpus, reservedMem, reservedDisk, reservedPorts, persistenceIDs := getReservedResources(mesosOffer.Resources)

	return &OfferHelper{
		MesosOffer:     mesosOffer,
		OfferIDStr:     mesosOffer.Id.GetValue(),
		PersistenceIDs: persistenceIDs,
		ReservedResources: &ResourceGroup{
			Cpus:  reservedCpus,
			Mem:   reservedMem,
			Disk:  reservedDisk,
			Ports: reservedPorts,
		},
		UnreservedResources: &ResourceGroup{
			Cpus:  unreservedCpus,
			Mem:   unreservedMem,
			Disk:  unreservedDisk,
			Ports: unreservedPorts,
		},
		ResourcesToReserve:  []*mesos.Resource{},
		ResourcesToUneserve: []*mesos.Resource{},
		VolumesToCreate:     []*mesos.Resource{},
		VolumesToDestroy:    []*mesos.Resource{},
		TasksToLaunch:       []*mesos.TaskInfo{},
	}
}

func (offerHelper *OfferHelper) String() string {
	return fmt.Sprintf("Reserved: (cpus:%v, mem: %v, disk: %v, ports: %v, persistenceIds: %+v), "+
		"Unreserved: (cpus:%v, mem: %v, disk: %v, ports: %+v)",
		offerHelper.ReservedResources.Cpus, offerHelper.ReservedResources.Mem,
		offerHelper.ReservedResources.Disk, len(offerHelper.ReservedResources.Ports), len(offerHelper.PersistenceIDs),
		offerHelper.UnreservedResources.Cpus, offerHelper.UnreservedResources.Mem,
		offerHelper.UnreservedResources.Disk, len(offerHelper.UnreservedResources.Ports))
}

func (offerHelper *OfferHelper) OfferHasReservations() bool {
	return offerHelper.ReservedResources.Cpus > 0 ||
		offerHelper.ReservedResources.Mem > 0 ||
		offerHelper.ReservedResources.Disk > 0 ||
		len(offerHelper.ReservedResources.Ports) > 0
}

func (offerHelper *OfferHelper) OfferHasVolumes() bool {
	return len(offerHelper.PersistenceIDs) > 0
}

func (offerHelper *OfferHelper) MaybeUnreserve() {
	if len(offerHelper.TasksToLaunch) > 0 {
		return
	}

	if offerHelper.OfferHasReservations() {
		log.Warnf("An offer has reserved resources, but no nodes can use it. Unreserving resources for OfferID: %+v", offerHelper.OfferIDStr)
		offerHelper.ResourcesToUneserve = append(offerHelper.ResourcesToUneserve, CopyReservedResources(offerHelper.MesosOffer.Resources)...)
	}

	if offerHelper.OfferHasVolumes() {
		log.Warnf("An offer has persisted volumes, but no nodes can use it. Destroying volumes for OfferID: %+v", offerHelper.OfferIDStr)
		offerHelper.VolumesToDestroy = append(offerHelper.VolumesToDestroy, CopyReservedVolumes(offerHelper.MesosOffer.Resources)...)
	}
}

func (offerHelper *OfferHelper) MakeReservation(cpus float64, mem float64, disk float64, ports int,
	principal string, role string) {
	reservation := offerHelper.apply(offerHelper.UnreservedResources, cpus, mem, disk, ports, principal, role, "", "")
	offerHelper.ResourcesToReserve = append(offerHelper.ResourcesToReserve, reservation...)
}

func (offerHelper *OfferHelper) MakeVolume(disk float64, principal string, role string,
	persistenceID string, containerPath string) {
	volume := offerHelper.apply(offerHelper.UnreservedResources, 0, 0, disk, 0, principal, role, persistenceID, containerPath)
	// Add the disk back since it was likely already removed in the reservation
	offerHelper.UnreservedResources.Disk = offerHelper.UnreservedResources.Disk + disk
	offerHelper.VolumesToCreate = append(offerHelper.VolumesToCreate, volume...)
}

func (offerHelper *OfferHelper) ApplyReserved(cpus float64, mem float64, disk float64, ports int,
	principal string, role string, persistenceID string, containerPath string) []*mesos.Resource {
	return offerHelper.apply(offerHelper.ReservedResources, cpus, mem, disk, ports, principal, role, persistenceID, containerPath)
}

func (offerHelper *OfferHelper) ApplyUnreserved(cpus float64, mem float64, disk float64, ports int) []*mesos.Resource {
	return offerHelper.apply(offerHelper.UnreservedResources, cpus, mem, disk, ports, "", "", "", "")
}

func (offerHelper *OfferHelper) CanFitReserved(cpus float64, mem float64, disk float64, ports int) bool {
	return offerHelper.ReservedResources.Cpus >= cpus &&
		offerHelper.ReservedResources.Mem >= mem &&
		offerHelper.ReservedResources.Disk >= disk &&
		len(offerHelper.ReservedResources.Ports) >= ports
}

func (offerHelper *OfferHelper) CanFitUnreserved(cpus float64, mem float64, disk float64, ports int) bool {
	return offerHelper.UnreservedResources.Cpus >= cpus &&
		offerHelper.UnreservedResources.Mem >= mem &&
		offerHelper.UnreservedResources.Disk >= disk &&
		len(offerHelper.UnreservedResources.Ports) >= ports
}

func (offerHelper *OfferHelper) HasPersistenceId(persistenceId string) bool {
	for _, offerPersistenceId := range offerHelper.PersistenceIDs {
		if offerPersistenceId == persistenceId {
			return true
		}
	}
	return false
}

func (offerHelper *OfferHelper) Operations() []*mesos.Offer_Operation {
	operations := []*mesos.Offer_Operation{}
	if len(offerHelper.TasksToLaunch) > 0 {
		operations = append(operations, util.NewLaunchOperation(offerHelper.TasksToLaunch))
	}
	if len(offerHelper.VolumesToDestroy) > 0 {
		operations = append(operations, util.NewDestroyOperation(offerHelper.VolumesToDestroy))
	}
	if len(offerHelper.ResourcesToUneserve) > 0 {
		operations = append(operations, util.NewUnreserveOperation(offerHelper.ResourcesToUneserve))
	}
	if len(offerHelper.ResourcesToReserve) > 0 {
		operations = append(operations, util.NewReserveOperation(offerHelper.ResourcesToReserve))
	}
	if len(offerHelper.VolumesToCreate) > 0 {
		operations = append(operations, util.NewCreateOperation(offerHelper.VolumesToCreate))
	}
	return operations
}

// --- Util ---

func getUnreservedResources(resources []*mesos.Resource) (float64, float64, float64, []int64) {
	return getResource("cpus", resources, false),
		getResource("mem", resources, false),
		getResource("disk", resources, false),
		getPorts(resources, false)
}

func getReservedResources(resources []*mesos.Resource) (float64, float64, float64, []int64, []string) {
	return getResource("cpus", resources, true),
		getResource("mem", resources, true),
		getResource("disk", resources, true),
		getPorts(resources, true),
		getPersistenceIds(resources)
}

func getPersistenceIds(resources []*mesos.Resource) []string {
	filtered := util.FilterResources(resources, func(res *mesos.Resource) bool {
		return res.GetName() == "disk" && res.Reservation != nil && res.Disk != nil
	})
	val := []string{}
	for _, res := range filtered {
		val = append(val, res.Disk.Persistence.GetId())
	}
	return val
}

func getPorts(resources []*mesos.Resource, withReservation bool) []int64 {
	filtered := util.FilterResources(resources, func(res *mesos.Resource) bool {
		if withReservation {
			return res.GetName() == "ports" && res.Reservation != nil
		}
		return res.GetName() == "ports" && res.Reservation == nil
	})
	val := []int64{}
	for _, res := range filtered {
		val = append(val, RangesToArray(res.GetRanges().GetRange())...)
	}
	return val
}

func getResource(name string, resources []*mesos.Resource, withReservation bool) float64 {
	filtered := util.FilterResources(resources, func(res *mesos.Resource) bool {
		if withReservation {
			return res.GetName() == name && res.Reservation != nil
		}
		return res.GetName() == name && res.Reservation == nil
	})
	val := 0.0
	for _, res := range filtered {
		val += res.GetScalar().GetValue()
	}
	return val
}

func (offerHelper *OfferHelper) apply(against *ResourceGroup, cpus float64, mem float64, disk float64, ports int,
	principal string, role string, persistenceID string, containerPath string) []*mesos.Resource {

	ask := []*mesos.Resource{}

	if cpus > 0 {
		against.Cpus = against.Cpus - cpus
		if principal != "" && role != "" {
			ask = append(ask, util.NewScalarResourceWithReservation("cpus", cpus, principal, role))
		} else {
			ask = append(ask, util.NewScalarResource("cpus", cpus))
		}
	}

	if mem > 0 {
		against.Mem = against.Mem - mem
		if principal != "" && role != "" {
			ask = append(ask, util.NewScalarResourceWithReservation("mem", mem, principal, role))
		} else {
			ask = append(ask, util.NewScalarResource("mem", mem))
		}
	}

	if disk > 0 {
		against.Disk = against.Disk - disk
		if principal != "" && role != "" && containerPath != "" && persistenceID != "" {
			ask = append(ask, util.NewVolumeResourceWithReservation(disk, containerPath, persistenceID, mesos.Volume_RW.Enum(), principal, role))
		} else if principal != "" && role != "" {
			ask = append(ask, util.NewScalarResourceWithReservation("disk", disk, principal, role))
		} else {
			ask = append(ask, util.NewScalarResource("disk", disk))
		}
	}

	if ports > 0 {
		sliceLoc := 0
		if len(against.Ports)-ports > 0 {
			sliceLoc = rand.Intn(len(against.Ports) - ports)
		}
		takingPorts := make([]int64, ports)
		copy(takingPorts, against.Ports[sliceLoc:(sliceLoc+ports)])
		leavingPorts := make([]int64, len(against.Ports)-ports)
		copy(leavingPorts, against.Ports[:sliceLoc])
		copy(leavingPorts[sliceLoc:], against.Ports[(sliceLoc+ports):])

		against.Ports = leavingPorts
		if principal != "" && role != "" {
			ask = append(ask, util.AddResourceReservation(util.NewRangesResource("ports", ArrayToRanges(takingPorts)), principal, role))
		} else {
			ask = append(ask, util.NewRangesResource("ports", ArrayToRanges(takingPorts)))
		}
	}

	return ask
}
