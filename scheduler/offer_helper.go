package scheduler

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	"github.com/basho-labs/riak-mesos/common"
)

func (sc *SchedulerCore) createOperationsForOffers(offers []*mesos.Offer) map[string][]*mesos.Offer_Operation {
	operations := make(map[string][]*mesos.Offer_Operation)
	var applySuccess bool
	nodesToLaunch, nodesToReserve := sc.getNodesToBeScheduled()

	// All nodes are elligible for launch in compatibilityMode
	if sc.compatibilityMode {
		nodesToLaunch = append(nodesToLaunch, nodesToReserve...)
		nodesToReserve = []*FrameworkRiakNode{}
	}

	// Populate operations
	for _, offer := range offers {
		if operations[*offer.Id.Value] == nil {
			operations[*offer.Id.Value] = []*mesos.Offer_Operation{}
		}

		// Launch all elligible nodes
		for _, riakNode := range nodesToLaunch {
			// Need to check again because we don't want to double book a node
			if riakNode.NeedsToBeScheduled() && (sc.compatibilityMode || riakNode.OfferCompatible(offer)) {
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					log.Infof("Found an offer for a launchable node. OfferID: %+v, NodeID: %+v", *offer.Id.Value, riakNode.CurrentID())
					launchOperation := sc.createLaunchNodeOperation(riakNode)
					operations[*offer.Id.Value] = append(operations[*offer.Id.Value], launchOperation)
					sc.schedulerState.Persist()
				}
			}
		}

		// The offer has reservations, but no node can use them, unreserve
		reservedResources := common.FilterReservedResources(offer.Resources)
		if len(reservedResources) > 0 && len(operations[*offer.Id.Value]) == 0 {
			log.Warnf("An offer has reservations, but no nodes can use it. Destroying any volumes and unreserving resources for OfferID: %+v", *offer.Id.Value)
			unreserveOperations := createUnreserveOfferOperations(offer)
			operations[*offer.Id.Value] = append(operations[*offer.Id.Value], unreserveOperations...)
		}

		// Reserve all elligible nodes
		for _, riakNode := range nodesToReserve {
			// Need to check again because we don't want to double book a node
			if !riakNode.HasRequestedReservation() {
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					log.Infof("Found a new offer for a node. OfferID: %+v, NodeID: %+v", *offer.Id.Value, riakNode.CurrentID())
					reserveOperations := createReserveNodeOperations(riakNode)
					operations[*offer.Id.Value] = append(operations[*offer.Id.Value], reserveOperations...)
					sc.schedulerState.Persist()
				}
			}
		}
	}

	return operations
}

func (sc *SchedulerCore) getNodesToBeScheduled() ([]*FrameworkRiakNode, []*FrameworkRiakNode) {
	// Find nodes which still need to be scheduled
	nodesWithReservations := []*FrameworkRiakNode{}
	nodesWithoutReservations := []*FrameworkRiakNode{}
	for _, cluster := range sc.schedulerState.Clusters {
		for _, riakNode := range cluster.Nodes {
			if riakNode.NeedsToBeScheduled() && riakNode.HasRequestedReservation() {
				log.Infof("Adding Riak node for scheduling (has reservations): %+v", riakNode.CurrentID())
				nodesWithReservations = append(nodesWithReservations, riakNode)
			} else if riakNode.NeedsToBeScheduled() && !riakNode.HasRequestedReservation() {
				log.Infof("Adding Riak node for scheduling (no reservations): %+v", riakNode.CurrentID())
				nodesWithoutReservations = append(nodesWithoutReservations, riakNode)
			}
		}
	}

	return nodesWithReservations, nodesWithoutReservations
}

func createUnreserveOfferOperations(offer *mesos.Offer) []*mesos.Offer_Operation {
	destroy := &mesos.Offer_Operation_Destroy{
		Volumes: offer.Resources,
	}
	destroyType := mesos.Offer_Operation_DESTROY
	destroyOperation := &mesos.Offer_Operation{
		Type:    &destroyType,
		Destroy: destroy,
	}
	unreserve := &mesos.Offer_Operation_Unreserve{
		Resources: offer.Resources,
	}
	unreserveType := mesos.Offer_Operation_UNRESERVE
	unreserveOperation := &mesos.Offer_Operation{
		Type:      &unreserveType,
		Unreserve: unreserve,
	}

	return []*mesos.Offer_Operation{destroyOperation, unreserveOperation}
}

func createReserveNodeOperations(riakNode *FrameworkRiakNode) []*mesos.Offer_Operation {
	reserveResources := riakNode.GetResourcesToReserve()
	createResources := riakNode.GetResourcesToCreate()

	reserve := &mesos.Offer_Operation_Reserve{
		Resources: reserveResources,
	}
	reserveType := mesos.Offer_Operation_RESERVE
	reserveOperation := &mesos.Offer_Operation{
		Type:    &reserveType,
		Reserve: reserve,
	}
	create := &mesos.Offer_Operation_Create{
		Volumes: createResources,
	}
	createType := mesos.Offer_Operation_CREATE
	createOperation := &mesos.Offer_Operation{
		Type:   &createType,
		Create: create,
	}

	return []*mesos.Offer_Operation{reserveOperation, createOperation}
}

func (sc *SchedulerCore) createLaunchNodeOperation(riakNode *FrameworkRiakNode) *mesos.Offer_Operation {
	log.Infof("Using offerId: %+v, for riakNode.CurrentID(): %+v", *riakNode.LastOfferUsed.Id.Value, riakNode.CurrentID())

	taskInfos := []*mesos.TaskInfo{}
	taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc)
	taskInfos = append(taskInfos, taskInfo)
	sc.frnDict[riakNode.CurrentID()] = riakNode

	launch := &mesos.Offer_Operation_Launch{
		TaskInfos: taskInfos,
	}
	operationType := mesos.Offer_Operation_LAUNCH
	launchOperation := &mesos.Offer_Operation{
		Type:   &operationType,
		Launch: launch,
	}

	return launchOperation
}
