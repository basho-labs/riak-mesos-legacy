# Development Guide

## Environment

For vagrant or regular Ubuntu 14.04, go to [https://github.com/basho-labs/vagrant-riak-mesos](https://github.com/basho-labs/vagrant-riak-mesos) and follow the directions to set up a development environment.

## Build

Download dependencies and build the platform specific executables

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos
make
```

## Usage

### Add some nodes to the cluster

TODO UPDATE ARGS

```
./bin/tools_darwin_amd64 \
    -name=riak-mesos-go3 \
    -zk=33.33.33.2:2181 \
    -command=create-cluster \
    -cluster-name=mycluster
./bin/tools_darwin_amd64 \
    -name=riak-mesos-go3 \
    -zk=33.33.33.2:2181 \
    -command=add-nodes \
    -nodes=3 \
    -cluster-name=mycluster
```

### Run the framework

TODO UPDATE ARGS

```
./bin/framework_linux_amd64 -id=riak-mesos-go3
```
