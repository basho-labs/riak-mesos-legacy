# Riak Mesos HTTP API

## Overview

The HTTP API is a component within the Riak Mesos Framework. It's primary
responsibility is to receive input from users and act on that input by
interacting with other components in the Framework to control a Riak cluster
created by the Framework.

## Design

### Client Interaction

![Client Interaction](RiakMesosControlFrame.png)

Due to the nature of Apache Mesos and the potential for Riak nodes to come and
go on a regular basis, client applications using a Mesos based cluster must
be kept up to date on the cluster's current state. Instead of requiring this
intelligence to be built into Riak client libraries, a small `Director`
application can run alongside HAProxy as well as the client application.

The Director will communicate with the HTTP API via Zookeeper to keep up to date
with Riak cluster changes. The Director in turn will update a local HAProxy
configuration with the currently known Riak node ip addresses.

This way, the entire Riak cluster is accessible to client applications by
communicating through the local HAProxy.

### API Requirements

The HTTP API must provide an external API with the following functionality:

#### Cluster Management

Name | Method | Path | Description
--- | --- | --- | ---
serveClusters | **GET** | `/clusters` | Lists clusters known to the framework
createCluster | **PUT**, **POST** | `/clusters/{cluster}` | Creates a cluster and stores metadata
getCluster | **GET** | `/clusters/{cluster}` | Returns cluster information

#### Node Management

Name | Method | Path | Description
--- | --- | --- | ---
serveNodes | **GET** | `/clusters/{cluster}/nodes` | Lists nodes for a cluster
createNode | **POST** | `/clusters/{cluster}/nodes` | Adds a node to a cluster
