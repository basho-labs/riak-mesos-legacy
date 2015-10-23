package scheduler

// https://github.com/apache/mesos/blob/master/docs/scheduler-http-api.md#accept

import (
	"bytes"
	json "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type AcceptOfferInfo struct {
	frameworkID   string
	offerID       string
	cpus          float64
	mem           float64
	disk          float64
	refuseSeconds float64
	role          string
	principal     string
	persistenceID string
	containerPath string
}

// MesosClient contains information common to all Mesos HTTP Server requests
type MesosClient struct {
	BaseURL string
}

// NewMesosClient creates a client struct to be used for future calls
func NewMesosClient(baseURL string) *MesosClient {
	c := &MesosClient{
		BaseURL: baseURL,
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
	cpusObj.Scalar.Value = acceptInfo.cpus
	cpusObj.Role = acceptInfo.role
	cpusObj.Reservation.Principal = acceptInfo.principal
	memObj := Resource{}
	memObj.Name = "mem"
	memObj.Type = "SCALAR"
	memObj.Scalar = Scalar{}
	memObj.Scalar.Value = acceptInfo.mem
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

func getAcceptMessage(acceptInfo *AcceptOfferInfo, operations []AnyOperation) AcceptMessage {
	filterObj := Filter{}
	filterObj.RefuseSeconds = acceptInfo.refuseSeconds
	filters := []Filter{filterObj}

	offerIDObj := OfferID{}
	offerIDObj.Value = acceptInfo.offerID
	offerIDs := []OfferID{offerIDObj}

	acceptMessageObj := AcceptMessage{}
	acceptMessageObj.FrameworkID = FrameworkID{}
	acceptMessageObj.FrameworkID.Value = acceptInfo.frameworkID
	acceptMessageObj.Type = "ACCEPT"
	acceptMessageObj.Accept = Accept{}
	acceptMessageObj.Accept.OfferIDs = offerIDs
	acceptMessageObj.Accept.Operations = operations
	acceptMessageObj.Accept.Filters = filters

	return acceptMessageObj
}

//OfferID is the id for the offer being accepted
type OfferID struct {
	Value string `json:"value"`
}

//Scalar is a value type
type Scalar struct {
	Value float64 `json:"value"`
}

//Reservation contains the principal
type Reservation struct {
	Principal string `json:"principal"`
}

//Persistence is an ID to track a volume request
type Persistence struct {
	ID string `json:"id"`
}

//Volume contains the persistent volume path location and mode
type Volume struct {
	ContainerPath string `json:"container_path"`
	Mode          string `json:"mode"`
}

//Disk is an optional part of a resource
type Disk struct {
	Persistence Persistence `json:"persistence"`
	Volume      Volume      `json:"volume"`
}

//Resource is a cpu, mem, or disk definition
type Resource struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Scalar      Scalar      `json:"scalar"`
	Role        string      `json:"role"`
	Reservation Reservation `json:"reservation"`
	Disk        *Disk       `json:"disk"`
}

//Operation contains resources
type Operation struct {
	Resources []Resource `json:"resources"`
}

//AnyOperation lives in an AcceptMessage to reserve or create resources
type AnyOperation struct {
	Type    string     `json:"type"`
	Reserve *Operation `json:"reserve"`
	Create  *Operation `json:"create"`
}

//Filter lives in an AcceptMessage to give a refuse seconds argument
type Filter struct {
	RefuseSeconds float64 `json:"refuse_seconds"`
}

//Accept is the contents of an ACCEPT message
type Accept struct {
	OfferIDs   []OfferID      `json:"offer_ids"`
	Operations []AnyOperation `json:"operations"`
	Filters    []Filter       `json:"filters"`
}

//FrameworkID is a single value object
type FrameworkID struct {
	Value string `json:"value"`
}

//AcceptMessage is the top level message sent to reserve resources and create volumes
type AcceptMessage struct {
	FrameworkID FrameworkID `json:"framework_id"`
	Type        string      `json:"type"`
	Accept      Accept      `json:"accept"`
}
