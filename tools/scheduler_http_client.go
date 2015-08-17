package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// SchedulerHTTPClient contains information common to all Scheduler HTTP Server requests
type SchedulerHTTPClient struct {
	BaseURL string
}

// NewSchedulerHTTPClient creates a client struct to be used for future calls
func NewSchedulerHTTPClient(baseURL string) *SchedulerHTTPClient {
	c := &SchedulerHTTPClient{
		BaseURL: baseURL,
	}

	return c
}

// GetNodes issues a GET to the Scheduler HTTP Server clusters/{cluster}/nodes endpoint
func (client *SchedulerHTTPClient) GetNodes() (string, error) {
	commandURI := "/api/v1/nodes"
	return client.doGet(commandURI)
}

// AddNode issues a POST to the Scheduler HTTP Server clusters/{cluster}/nodes endpoint
func (client *SchedulerHTTPClient) AddNode() (string, error) {
	commandURI := "/api/v1/nodes"
	return client.doPost(commandURI)
}

func (client *SchedulerHTTPClient) doGet(path string) (string, error) {
	commandURL := fmt.Sprintf("%s%s", client.BaseURL, path)
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

func (client *SchedulerHTTPClient) doPost(path string) (string, error) {
	commandURL := fmt.Sprintf("%s%s", client.BaseURL, path)
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
