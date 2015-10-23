package scheduler

// https://github.com/apache/mesos/blob/master/docs/scheduler-http-api.md#accept
//TODO: add content type application/json to headers (Or try protobuf stuff)
//TODO: add Accepts application/json to headers
//TODO: add user / pass encoded as auth header
//TODO: need to fix all the disk stuff, diskInfo?

import (
	"bytes"
	json "encoding/json"
	"fmt"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"io"
	"io/ioutil"
	"net/http"
)

// MesosClient contains information common to all Mesos HTTP Server requests
type MesosClient struct {
	baseURL       string
	frameworkID   string
	refuseSeconds float64
}

// NewMesosClient creates a client struct to be used for future calls
func NewMesosClient(baseURL string, frameworkID string, refuseSeconds float64) *MesosClient {
	c := &MesosClient{
		baseURL:       baseURL,
		frameworkID:   frameworkID,
		refuseSeconds: refuseSeconds,
	}

	return c
}

//ReserveResourceAndCreateVolume attempts to reserve resources from an offer
func (client *MesosClient) ReserveResourceAndCreateVolume(acceptInfo *AcceptOfferInfo) (bool, error) {
	resources := getResourceRequest(acceptInfo)

	createOperation := getCreateOperation(resources)
	reserveOperation := getReserveOperation(resources)
	operations := []AnyOperation{reserveOperation, createOperation}

	acceptMessageObj := getAcceptMessage(acceptInfo, operations)

	acceptMessageBytes, _ := json.Marshal(acceptMessageObj)
	// acceptMessageJSON := string(acceptMessageBytes)

	code, body, err := client.doPostWithData("api/v1/scheduler", bytes.NewReader(acceptMessageBytes))

	if err != nil {
		return false, err
	}
	if code != 202 {
		return false, fmt.Errorf("Unable to reserve resources and create volume, code: %+v, response body: %+v", code, body)
	}
	return true, nil
}

func (client *MesosClient) doGet(path string) (int, string, error) {
	commandURL := fmt.Sprintf("%s/%s", client.BaseURL, path)
	resp, err := http.Get(commandURL)
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

func (client *MesosClient) doPostWithData(path string, data io.Reader) (int, string, error) {
	commandURL := fmt.Sprintf("%s/%s", client.BaseURL, path)
	resp, err := http.Post(commandURL, "", data)
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

func (client *MesosClient) doPost(path string) (int, string, error) {
	commandURL := fmt.Sprintf("%s/%s", client.BaseURL, path)
	resp, err := http.Post(commandURL, "", nil)
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

func getResourceRequest(acceptInfo *AcceptOfferInfo) []Resource {
	cpusObj := Resource{}
	cpusObj.Name = "cpus"
	cpusObj.Type = "SCALAR"
	cpusObj.Scalar = Scalar{}
	cpusObj.Scalar.Value = acceptInfo.cpus + acceptInfo.ExecCpus
	cpusObj.Role = acceptInfo.role
	cpusObj.Reservation.Principal = acceptInfo.principal
	memObj := Resource{}
	memObj.Name = "mem"
	memObj.Type = "SCALAR"
	memObj.Scalar = Scalar{}
	memObj.Scalar.Value = acceptInfo.mem + acceptInfo.execMem
	memObj.Role = acceptInfo.role
	memObj.Reservation.Principal = acceptInfo.principal
	diskObj := Resource{}
	diskObj.Name = "disk"
	diskObj.Type = "SCALAR"
	diskObj.Scalar = Scalar{}
	diskObj.Scalar.Value = acceptInfo.disk
	diskObj.Role = acceptInfo.role
	diskObj.Reservation.Principal = acceptInfo.principal
	diskObj.Disk = &Disk{}
	diskObj.Disk.Persistence = Persistence{}
	diskObj.Disk.Persistence.ID = acceptInfo.persistenceID
	diskObj.Disk.Volume = Volume{}
	diskObj.Disk.Volume.ContainerPath = acceptInfo.containerPath
	diskObj.Disk.Volume.Mode = "RW"
	resources := []Resource{cpusObj, memObj, diskObj}

	return resources
}

func getReserveOperation(resources []Resource) AnyOperation {
	reserveOperation := AnyOperation{}
	reserveOperation.Type = "RESERVE"
	reserveOperation.Reserve = &Operation{}
	reserveOperation.Reserve.Resources = resources

	return reserveOperation
}

func getCreateOperation(resources []Resource) AnyOperation {
	createOperation := AnyOperation{}
	createOperation.Type = "CREATE"
	createOperation.Create = &Operation{}
	createOperation.Create.Resources = resources

	return createOperation
}

func (client *MesosClient) getAcceptMessage(acceptInfo *AcceptOfferInfo, operations []*mesos.Offer_Operation) *mesos.Call {
	frameworkId := *mesos.FrameworkID{
		Value: client.frameworkID,
	}
	accept := *mesos.Call_Accept{
		OfferIds:   []*mesos.OfferID{acceptInfo.offerID},
		Operations: operations,
		Filters:    &mesos.Filters{RefuseSeconds: proto.Float64(client.refuseSeconds)},
	}
	message := *mesos.Call{
		FrameworkId: &frameworkId,
		Type:        &mesos.Call_ACCEPT,
		Accept:      &accept,
	}

	return message
}
