package scheduler

// https://github.com/apache/mesos/blob/master/docs/scheduler-http-api.md#accept

import (
	json "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

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

// Reserve attempts to reserve resources from an offer
func (client *MesosClient) OfferReserve(path string) (string, error) {

}

func (client *MesosClient) doGet(path string) (string, error) {
	commandURL := fmt.Sprintf("%s/api/v1/%s", client.BaseURL, path)
	resp, err := http.Get(commandURL)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body[:]), nil
}

func (client *MesosClient) doPostWithData(path string, data io.Reader) (string, error) {
	commandURL := fmt.Sprintf("%s/api/v1/%s", client.BaseURL, path)
	resp, err := http.Post(commandURL, "", data)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body[:]), nil
}

func (client *MesosClient) doPost(path string) (string, error) {
	commandURL := fmt.Sprintf("%s/api/v1/%s", client.BaseURL, path)
	resp, err := http.Post(commandURL, "", nil)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body[:]), nil
}
