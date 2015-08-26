# Riak Framework on Mesos

## Installation

The Riak Mesos Framework can be configured in a few different ways depending on the restraints of the Mesos cluster.

### Marathon Usage

Sample Riak Mesos Framework `marathon.json`: [../mararthon.json](../marathon.json).
Sample Riak Mesos Director `marathon.json`: [../director.mararthon.json](../marathon.json).

### Manual Usage

#### Start the framework

Download and extract the Riak Mesos Framework (link in [../README.md](../README.md)), and start the framework with an incantation similar to this:

```
./framework_linux_amd64 \
    -master=zk://master.mesos:2181/mesos \
    -zk=master.mesos:2181 \
    -name=riak \
    -user=root \
    -role=*
```

Included with the framework tarball is a CLI tool named `tools_linux_amd64` which can be used to perform a variety of tasks on a running Riak Mesos Framework instance. Following are some usage instructions.

Configure a few environment variables matching your setup for convenience.

#### Create a cluster

```
./tools_linux_amd64 \
    -name=riak \
    -zk=master.mesos:2181 \
    -cluster-name=mycluster \
    -command="create-cluster"
```

Add Riak nodes

```
./tools_linux_amd64 \
    -name=riak \
    -zk=master.mesos:2181 \
    -cluster-name=mycluster \
    -command="add-nodes" \
    -nodes=5
```

Get the base URL for the Riak Mesos Framework [HTTP API](docs/HTTP-API.md) endpoints for more ways to interact with the framework.

```
./tools_linux_amd64 -name=$NAME -zk=$ZK -command="get-url"
```

#### Start the director

Download and extract the Riak Mesos Director (link in [../README.md](../README.md)), and start the framework with an incantation similar to this:

```
FRAMEWORK_HOST=master.mesos FRAMEWORK_PORT=9090 DIRECTOR_CLUSTER=mycluster DIRECTOR_FRAMEWORK=riak DIRECTOR_ZK=master.mesos:2181 ./director_linux_amd64
```

Starting the director should give you access to a number of endpoints:

* Balanced Riak HTTP [http://master.mesos:8098](http://master.mesos:8098)
* Balanced Riak Protobuf [http://master.mesos:8087](http://master.mesos:8087)
* Director HTTP API [http://master.mesos:9000](http://master.mesos:9000)
* Riak Explorer Web UI and API [http://master.mesos:9999](http://master.mesos:9999)

These ports are the defaults, and will be dynamically assigned when using Marathon or DCOS.
