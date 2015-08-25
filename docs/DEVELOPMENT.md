# Development Guide

## Environment

For vagrant or regular Ubuntu 14.04, go to [https://github.com/basho-labs/vagrant-riak-mesos](https://github.com/basho-labs/vagrant-riak-mesos) and follow the directions to set up a development environment.

## Build

Download dependencies and build the platform specific executables

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && make
```

## Usage

### Add some nodes to the cluster

```
./bin/tools_darwin_amd64 \
    -name=riak \
    -zk=localhost:2181 \
    -command=create-cluster \
    -cluster-name=mycluster
./bin/tools_darwin_amd64 \
    -name=riak \
    -zk=localhost:2181 \
    -command=add-nodes \
    -nodes=1 \
    -cluster-name=mycluster
```

### Run the framework

```
./bin/framework_linux_amd64 \
    -master=zk://localhost:2181/mesos \
    -name=riak \
    -user=root \
    -zk=localhost:2181
```
