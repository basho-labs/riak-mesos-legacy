# Riak Mesos HTTP API

The HTTP API component of the Riak Mesos Framework is primarily used by other
tools to create and modify a running Riak cluster. The API may also be used by
custom tools. Below is a specification of the provided endpoints.

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
