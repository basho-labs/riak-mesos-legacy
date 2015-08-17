# Riak Framework on Mesos

## Installation

The Riak Mesos Framework can be configured in a few different ways depending on the restraints of
the Mesos cluster.

### Marathon Setup

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

### Manual Setup

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
