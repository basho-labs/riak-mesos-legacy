package scheduler

// https://github.com/apache/mesos/blob/master/docs/scheduler-http-api.md#accept
//TODO: add content type application/json to headers (Or try protobuf stuff)
//TODO: add Accepts application/json to headers
//TODO: add user / pass encoded as auth header
//TODO: need to fix all the disk stuff, diskInfo?

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"io"
	"io/ioutil"
	"net/http"
)

// MesosClient contains information common to all Mesos HTTP Server requests
type MesosClient struct {
	baseURL       string
	frameworkID   *string
	refuseSeconds float64
}

// NewMesosClient creates a client struct to be used for future calls
func NewMesosClient(baseURL string, frameworkID *string, refuseSeconds float64) *MesosClient {
	c := &MesosClient{
		baseURL:       baseURL,
		frameworkID:   frameworkID,
		refuseSeconds: refuseSeconds,
	}

	return c
}

//UnreserveResourceAndDestroyVolume attempts to unreserve resources from an offer
func (client *MesosClient) UnreserveResourceAndDestroyVolume(offer *mesos.Offer) error {
	destroyOperation := getDestroyOperation(offer.Resources)
	unreserveOperation := getUnreserveOperation(offer.Resources)

	operations := []*mesos.Offer_Operation{destroyOperation, unreserveOperation}

	acceptMessageObj := client.getAcceptMessage(offer.Id, operations)
	acceptMessageBytes, _ := proto.Marshal(acceptMessageObj)

	log.Infof("Sending destroy / unreserve operations. Client: %+v, Message: %+v.", client, acceptMessageObj)

	code, body, err := client.doPostWithData("api/v1/scheduler", bytes.NewReader(acceptMessageBytes))

	if err != nil {
		return err
	}
	if code != 202 {
		return fmt.Errorf("Unable to destroy volume and unreserve resources, code: %+v, response body: %+v", code, body)
	}
	return nil
}

//ReserveResourceAndCreateVolume attempts to reserve resources from an offer
func (client *MesosClient) ReserveResourceAndCreateVolume(offerID *mesos.OfferID, reserveResources []*mesos.Resource, createResources []*mesos.Resource) error {
	reserveOperation := getReserveOperation(reserveResources)
	createOperation := getCreateOperation(createResources)

	operations := []*mesos.Offer_Operation{reserveOperation, createOperation}

	acceptMessageObj := client.getAcceptMessage(offerID, operations)
	acceptMessageBytes, _ := proto.Marshal(acceptMessageObj)

	log.Infof("Sending reserve / create operations. Client: %+v, Message: %+v.", client, acceptMessageObj)

	code, body, err := client.doPostWithData("api/v1/scheduler", bytes.NewReader(acceptMessageBytes))

	if err != nil {
		return err
	}
	if code != 202 {
		return fmt.Errorf("Unable to reserve resources and create volume, code: %+v, response body: %+v", code, body)
	}
	return nil
}

func getReserveOperation(resources []*mesos.Resource) *mesos.Offer_Operation {
	// Make sure we're only sending what can be reserved
	var reservations []*mesos.Resource
	for _, resource := range util.FilterResources(resources, func(res *mesos.Resource) bool { return res.Reservation != nil }) {
		if resource.GetName() == "disk" {
			resource.Disk = nil
		}
		reservations = append(reservations, resource)
	}

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

func getUnreserveOperation(reservations []*mesos.Resource) *mesos.Offer_Operation {
	// Make sure we're only sending what can be unreserved
	var unreservations []*mesos.Resource
	for _, resource := range util.FilterResources(reservations, func(res *mesos.Resource) bool { return res.Reservation != nil }) {
		if resource.GetName() == "disk" {
			resource.Disk = nil
		}
		unreservations = append(unreservations, resource)
	}

	unreserve := &mesos.Offer_Operation_Unreserve{
		Resources: unreservations,
	}
	operationType := mesos.Offer_Operation_UNRESERVE
	operation := &mesos.Offer_Operation{
		Type:      &operationType,
		Unreserve: unreserve,
	}

	return operation
}

func getCreateOperation(resources []*mesos.Resource) *mesos.Offer_Operation {
	// Make sure we're only sending the disk portion
	var volumes []*mesos.Resource
	for _, resource := range util.FilterResources(resources, func(res *mesos.Resource) bool { return res.GetName() == "disk" }) {
		volumes = append(volumes, resource)
	}

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

func getDestroyOperation(resources []*mesos.Resource) *mesos.Offer_Operation {
	// Make sure we're only sending the disk portion
	var volumes []*mesos.Resource
	for _, resource := range util.FilterResources(resources, func(res *mesos.Resource) bool { return res.Reservation != nil && res.GetName() == "disk" }) {
		volumes = append(volumes, resource)
	}

	destroy := &mesos.Offer_Operation_Destroy{
		Volumes: volumes,
	}
	operationType := mesos.Offer_Operation_DESTROY
	operation := &mesos.Offer_Operation{
		Type:    &operationType,
		Destroy: destroy,
	}

	return operation
}

func (client *MesosClient) getAcceptMessage(offerID *mesos.OfferID, operations []*mesos.Offer_Operation) *mesos.Call {
	frameworkID := &mesos.FrameworkID{
		Value: client.frameworkID,
	}
	accept := &mesos.Call_Accept{
		OfferIds:   []*mesos.OfferID{offerID},
		Operations: operations,
		Filters:    &mesos.Filters{RefuseSeconds: proto.Float64(client.refuseSeconds)},
	}
	callType := mesos.Call_ACCEPT
	message := &mesos.Call{
		FrameworkId: frameworkID,
		Type:        &callType,
		Accept:      accept,
	}

	return message
}

func (client *MesosClient) doPostWithData(path string, data io.Reader) (int, string, error) {
	commandURL := fmt.Sprintf("http://%s/%s", client.baseURL, path)

	resp, err := http.Post(commandURL, "application/x-protobuf", data)
	if err != nil {
		return 0, "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(body[:]), nil
}

// func (client *MesosClient) doGet(path string) (int, string, error) {
// 	commandURL := fmt.Sprintf("%s/%s", client.BaseURL, path)
// 	resp, err := http.Get(commandURL)
// 	if err != nil {
// 		return 0, "", err
// 	}
// 	body, err := ioutil.ReadAll(resp.Body)
// 	defer resp.Body.Close()
// 	if err != nil {
// 		return 0, "", err
// 	}
// 	return resp.StatusCode, string(body[:]), nil
// }

// func (client *MesosClient) doPost(path string) (int, string, error) {
// 	commandURL := fmt.Sprintf("%s/%s", client.BaseURL, path)
// 	resp, err := http.Post(commandURL, "", nil)
// 	if err != nil {
// 		return 0, "", err
// 	}
// 	body, err := ioutil.ReadAll(resp.Body)
// 	defer resp.Body.Close()
// 	if err != nil {
// 		return 0, "", err
// 	}
// 	return resp.StatusCode, string(body[:]), nil
// }

// func getReserveOperation(resources []Resource) AnyOperation {
// 	reserveOperation := AnyOperation{}
// 	reserveOperation.Type = "RESERVE"
// 	reserveOperation.Reserve = &Operation{}
// 	reserveOperation.Reserve.Resources = resources
//
// 	return reserveOperation
// }
//
// func getCreateOperation(resources []Resource) AnyOperation {
// 	createOperation := AnyOperation{}
// 	createOperation.Type = "CREATE"
// 	createOperation.Create = &Operation{}
// 	createOperation.Create.Resources = resources
//
// 	return createOperation
// }
