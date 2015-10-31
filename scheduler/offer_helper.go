package scheduler

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	"github.com/basho-labs/riak-mesos/common"
)

func (sc *SchedulerCore) createOperationsForOffers(offers []*mesos.Offer) map[string][]*mesos.Offer_Operation {
	var applySuccess bool
	operations := make(map[string][]*mesos.Offer_Operation)
	nodesToLaunch, nodesToReserve := sc.getNodesToBeScheduled()

	// All nodes are elligible for launch in compatibilityMode
	if sc.compatibilityMode {
		nodesToLaunch = append(nodesToLaunch, nodesToReserve...)
		nodesToReserve = []*FrameworkRiakNode{}
	}

	// Populate operations
	for _, offer := range offers {
		log.Infof("Got offer with these resources: %s", common.PrettyStringForScalarResources(offer.Resources))
		operations[*offer.Id.Value] = []*mesos.Offer_Operation{}
		launchTasks := []*mesos.TaskInfo{}
		createResources := []*mesos.Resource{}
		destroyResources := []*mesos.Resource{}
		reserveResources := []*mesos.Resource{}
		unreserveResources := []*mesos.Resource{}

		// Launch all elligible nodes
		for _, riakNode := range nodesToLaunch {
			// Need to check again because we don't want to double book a node
			if riakNode.CanBeScheduled() && (sc.compatibilityMode || riakNode.OfferCompatible(offer)) {
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					log.Infof("Found an offer for a launchable node. OfferID: %+v, NodeID: %+v", *offer.Id.Value, riakNode.CurrentID())
					taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc)
					launchTasks = append(launchTasks, taskInfo)
					sc.schedulerState.Persist()
				}
			}
		}

		// The offer has reservations, but noone can use them
		if len(common.FilterReservedResources(offer.Resources)) > 0 && len(launchTasks) == 0 && len(nodesToLaunch) == 0 {
			log.Warnf("An offer has reserved resources, but no nodes can use it. Unreserving resources for OfferID: %+v", *offer.Id.Value)
			unreserveResources = append(unreserveResources, common.CopyReservedResources(offer.Resources)...)
		}
		if len(common.FilterReservedVolumes(offer.Resources)) > 0 && len(launchTasks) == 0 && len(nodesToLaunch) == 0 {
			log.Warnf("An offer has persisted volumes, but no nodes can use it. Destroying volumes for OfferID: %+v", *offer.Id.Value)
			destroyResources = append(destroyResources, common.CopyReservedVolumes(offer.Resources)...)

		}
		if len(common.FilterReservedResources(offer.Resources)) == 0 && len(common.FilterReservedVolumes(offer.Resources)) == 0 && len(nodesToLaunch) > 0 && len(launchTasks) == 0 {
			for _, riakNode := range nodesToLaunch {
				log.Infof("Unable to launch node that had reserved resources, unreserving. NodeID: %+v", riakNode.CurrentID())
				riakNode.Unreserve()
			}
			sc.schedulerState.Persist()
		}

		// Reserve all elligible nodes
		for _, riakNode := range nodesToReserve {
			// Need to check again because we don't want to double book a node
			if !riakNode.HasRequestedReservation() {
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					log.Infof("Found a new offer for a node. OfferID: %+v, NodeID: %+v", *offer.Id.Value, riakNode.CurrentID())
					createResources = append(createResources, riakNode.GetResourcesToCreate()...)
					reserveResources = append(reserveResources, riakNode.GetResourcesToReserve()...)
					sc.schedulerState.Persist()
				}
			}
		}

		operations[*offer.Id.Value] = append(operations[*offer.Id.Value], buildLaunchOperation(launchTasks)...)
		operations[*offer.Id.Value] = append(operations[*offer.Id.Value], buildDestroyOperation(destroyResources)...)
		operations[*offer.Id.Value] = append(operations[*offer.Id.Value], buildUnreserveOperation(unreserveResources)...)
		operations[*offer.Id.Value] = append(operations[*offer.Id.Value], buildReserveOperation(reserveResources)...)
		operations[*offer.Id.Value] = append(operations[*offer.Id.Value], buildCreateOperation(createResources)...)
	}

	return operations
}

func (sc *SchedulerCore) getNodesToBeScheduled() ([]*FrameworkRiakNode, []*FrameworkRiakNode) {
	// Find nodes which still need to be scheduled
	nodesWithReservations := []*FrameworkRiakNode{}
	nodesWithoutReservations := []*FrameworkRiakNode{}
	for _, cluster := range sc.schedulerState.Clusters {
		for _, riakNode := range cluster.Nodes {
			if riakNode.CanBeScheduled() && riakNode.HasRequestedReservation() {
				log.Infof("Adding Riak node for scheduling (has reservations): %+v", riakNode.CurrentID())
				nodesWithReservations = append(nodesWithReservations, riakNode)
			} else if riakNode.CanBeScheduled() && !riakNode.HasRequestedReservation() {
				log.Infof("Adding Riak node for scheduling (no reservations): %+v", riakNode.CurrentID())
				nodesWithoutReservations = append(nodesWithoutReservations, riakNode)
			}
		}
	}

	return nodesWithReservations, nodesWithoutReservations
}

func buildDestroyOperation(resources []*mesos.Resource) []*mesos.Offer_Operation {
	if len(resources) == 0 {
		return []*mesos.Offer_Operation{}
	}

	destroy := &mesos.Offer_Operation_Destroy{
		Volumes: resources,
	}
	destroyType := mesos.Offer_Operation_DESTROY
	destroyOperation := &mesos.Offer_Operation{
		Type:    &destroyType,
		Destroy: destroy,
	}

	return []*mesos.Offer_Operation{destroyOperation}
}

func buildUnreserveOperation(resources []*mesos.Resource) []*mesos.Offer_Operation {
	if len(resources) == 0 {
		return []*mesos.Offer_Operation{}
	}

	unreserve := &mesos.Offer_Operation_Unreserve{
		Resources: resources,
	}
	unreserveType := mesos.Offer_Operation_UNRESERVE
	unreserveOperation := &mesos.Offer_Operation{
		Type:      &unreserveType,
		Unreserve: unreserve,
	}

	return []*mesos.Offer_Operation{unreserveOperation}
}

func buildCreateOperation(resources []*mesos.Resource) []*mesos.Offer_Operation {
	if len(resources) == 0 {
		return []*mesos.Offer_Operation{}
	}

	create := &mesos.Offer_Operation_Create{
		Volumes: resources,
	}
	createType := mesos.Offer_Operation_CREATE
	createOperation := &mesos.Offer_Operation{
		Type:   &createType,
		Create: create,
	}

	return []*mesos.Offer_Operation{createOperation}
}

func buildReserveOperation(resources []*mesos.Resource) []*mesos.Offer_Operation {
	if len(resources) == 0 {
		return []*mesos.Offer_Operation{}
	}

	reserve := &mesos.Offer_Operation_Reserve{
		Resources: resources,
	}
	reserveType := mesos.Offer_Operation_RESERVE
	reserveOperation := &mesos.Offer_Operation{
		Type:    &reserveType,
		Reserve: reserve,
	}

	return []*mesos.Offer_Operation{reserveOperation}
}

func buildLaunchOperation(tasks []*mesos.TaskInfo) []*mesos.Offer_Operation {
	if len(tasks) == 0 {
		return []*mesos.Offer_Operation{}
	}

	launch := &mesos.Offer_Operation_Launch{
		TaskInfos: tasks,
	}
	operationType := mesos.Offer_Operation_LAUNCH
	launchOperation := &mesos.Offer_Operation{
		Type:   &operationType,
		Launch: launch,
	}

	return []*mesos.Offer_Operation{launchOperation}
}
