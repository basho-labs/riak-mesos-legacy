# Riak Mesos Framework

An [Apache Mesos](http://mesos.apache.org/) framework for [Riak KV](http://basho.com/products/riak-kv/), a distributed NoSQL key-value data store that offers high availability, fault tolerance,
operational simplicity, and scalability.

## Quick Links

* [Architecture](#architecture)
    * [Scheduler](#scheduler)
    * [Director](#director)
* [Usage](#usage)
    * [Installation](#installation)
        * [Marathon Setup](#marathon-setup)
        * [Manual Setup](#manual-setup)
    * [Riak Cluster Configuration](#riak-cluster-configuration)
    * [CLI Tool](#cli-tool)
    * [HTTP API](#http-api.md)

### DCOS Users

* [DCOS Resources](dcos/)

### Documentation

* [Development Setup](docs/DEVELOPMENT-SETUP.md)
* [Development Guide](docs/DEVELOPMENT.md)
* [Director](docs/DIRECTOR.md)
* [HTTP API](docs/HTTP-API.md)

## Architecture

### Scheduler

The Riak Mesos Framework scheduler will attempt to spread Riak nodes across as many different
mesos agents as possible to increase fault tolerance. If there are more nodes requested than
there are agents available, the scheduler will then start adding more Riak nodes to existing
agents.

![Architecture](docs/RiakMesosFramework.png)

### Director

Due to the nature of Apache Mesos and the potential for Riak nodes to come and
go on a regular basis, client applications using a Mesos based cluster must
be kept up to date on the cluster's current state. Instead of requiring this
intelligence to be built into Riak client libraries, a smart proxy application named
`Director` has been created which can run alongside client applications.

![Director](docs/RiakMesosControlFrame.png)

For installation and usage instructions related to the Riak Mesos Director, please read [docs/DIRECTOR.md](docs/DIRECTOR.md)

## Usage

### Installation

The Riak Mesos Framework can be configured in a few different ways depending on the restraints of
the Mesos cluster.

#### Marathon Setup

Sample `marathon.json`

```
{
  "id": "/riak",
  "instances": 1,
  "cpus": 0.5,
  "mem": 512,
  "ports": [0,0],
  "uris": [
      "http://riak-tools.s3.amazonaws.com/riak_mesos_framework_0.1.0_linux_amd64.tar.gz"
  ],
  "env": {},
  "args": [
      "framework_linux_amd64",
      "-master=zk://master.mesos:2181/mesos",
      "-zk=master.mesos:2181",
      "-id=riak-mesos-go",
      "-name=\"Riak Mesos Framework\"",
      "-role=*"],
  "healthChecks": [
    {
      "path": "/healthcheck",
      "portIndex": 0,
      "protocol": "HTTP",
      "gracePeriodSeconds": 300,
      "intervalSeconds": 60,
      "timeoutSeconds": 20,
      "maxConsecutiveFailures": 5,
      "ignoreHttp1xx": false
    }
  ]
}
```

#### Manual Setup

Download and extract [riak_mesos_framework_linux_amd64_0.1.0.tar.gz](http://riak-tools.s3.amazonaws.com/riak_mesos_framework_linux_amd64_0.1.0.tar.gz), and start the framework with an incantation similar to this:

```
./framework_linux_amd64 \
    -master=zk://master.mesos:2181/mesos \
    -zk=master.mesos:2181 \
    -id=riak-mesos-go \
    -user=centos \
    -role=* \
    -ip=master.mesos \
    -hostname=master.mesos
```

### Riak Cluster Configuration

#### CLI Tool

Included with the framework tarball is a CLI tool named `tools_linux_amd64` which can be used to
perform a variety of tasks on a running Riak Mesos Framework instance. Following are some usage
instructions.

Configure a few environment variables matching your setup for convenience.

```
NAME="riak-mesos-go"
ZK="master.mesos:2181"
CLUSTERNAME="mycluster"
```

Create a cluster

```
./tools_darwin_amd64 \
    -name=$NAME \
    -zk=$ZK \
    -cluster-name=$CLUSTERNAME \
    -command="create-cluster"
```

Set the initial node count

```
./tools_darwin_amd64 \
    -name=$NAME \
    -zk=$ZK \
    -cluster-name=$CLUSTERNAME \
    -command="add-nodes" \
    -nodes=5
```

Get the base URL for the Riak Mesos Framework [HTTP API](docs/HTTP-API.md) endpoints.

```
./tools_darwin_amd64 -name=$NAME -zk=$ZK -command="get-url"
```

#### HTTP API

Clusters may also be configured using the HTTP API exposed by the framework. For more information,
please read the [docs/HTTP-API.md](docs/HTTP-API.md) document.

## Development / Contributing

For initial setup of development environment, please follow the directions in
[docs/DEVELOPMENT-SETUP.md](docs/DEVELOPMENT-SETUP.md). For further build and testing information,
visit [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
