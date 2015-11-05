package main

import (
	"fmt"
	"io"
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

func (client *SchedulerHTTPClient) GetClusterConfig(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/config", clusterName)
	return client.doGet(commandURI)
}

func (client *SchedulerHTTPClient) SetClusterConfig(clusterName string, file io.Reader) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/config", clusterName)
	return client.doPostWithData(commandURI, file)
}
func (client *SchedulerHTTPClient) GetClusterAdvancedConfig(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/advancedConfig", clusterName)
	return client.doGet(commandURI)
}

func (client *SchedulerHTTPClient) SetClusterAdvancedConfig(clusterName string, file io.Reader) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/advancedConfig", clusterName)
	return client.doPostWithData(commandURI, file)
}

// GetClusters issues a Get to the Scheduler HTTP Server clusters endpoint
func (client *SchedulerHTTPClient) GetClusters() (string, error) {
	return client.doGet("clusters")
}

// GetCluster issues a Get to the Scheduler HTTP Server clusters/{cluster} endpoint
func (client *SchedulerHTTPClient) GetCluster(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s", clusterName)
	return client.doGet(commandURI)
}

// CreateCluster issues a POST to the Scheduler HTTP Server clusters{cluster} endpoint
func (client *SchedulerHTTPClient) CreateCluster(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s", clusterName)
	return client.doPost(commandURI)
}

// DeleteCluster issues a DELETE to the Scheduler HTTP Server clusters/{cluster} endpoint
func (client *SchedulerHTTPClient) DeleteCluster(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s", clusterName)
	return client.doDelete(commandURI)
}

// DeleteNode issues a DELETE to the Scheduler HTTP Server clusters/{cluster} endpoint
func (client *SchedulerHTTPClient) DeleteNode(clusterName string, nodeName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/nodes/%s", clusterName, nodeName)
	return client.doDelete(commandURI)
}

// GetNodes issues a GET to the Scheduler HTTP Server clusters/{cluster}/nodes endpoint
func (client *SchedulerHTTPClient) GetNodes(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/nodes", clusterName)
	return client.doGet(commandURI)
}

// GetNodeHosts issues a GET to the Scheduler HTTP Server clusters/{cluster}/nodehosts endpoint
func (client *SchedulerHTTPClient) GetNodeHosts(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/nodehosts", clusterName)
	return client.doGet(commandURI)
}

// AddNode issues a POST to the Scheduler HTTP Server clusters/{cluster}/nodes endpoint
func (client *SchedulerHTTPClient) AddNode(clusterName string) (string, error) {
	commandURI := fmt.Sprintf("clusters/%s/nodes", clusterName)
	return client.doPost(commandURI)
}

func (client *SchedulerHTTPClient) doGet(path string) (string, error) {
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

func (client *SchedulerHTTPClient) doPostWithData(path string, data io.Reader) (string, error) {
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

func (client *SchedulerHTTPClient) doPost(path string) (string, error) {
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

func (client *SchedulerHTTPClient) doDelete(path string) (string, error) {
	commandURL := fmt.Sprintf("%s/api/v1/%s", client.BaseURL, path)

	req, _ := http.NewRequest("DELETE", commandURL, nil)
	resp, err := http.DefaultClient.Do(req)

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
