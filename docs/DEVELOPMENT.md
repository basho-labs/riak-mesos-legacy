# Development Guide

## Quickstart

To build the framework and get it running in a Mesos vagrant environment on Mac OSX, follow these steps:

### Vagrant Dev Environment

Bring up the build environment with a running Mesos and ssh in

```
vagrant plugin install vagrant-hostmanager
vagrant up
vagrant reload
vagrant ssh
```

### Set up Dependencies

```
cd /vagrant && ./setup-env.sh
```

### Build the Framework

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && make
```

or for a faster build with lower RAM requirements:

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && TAGS=dev make
```

### Creating Native / Platform Specific Builds

By default, the build process will include a Debian Ubuntu image in the binary to support multiple platforms using chroot. To build the framework without relying on chroot, everything can be built natively:

```
# Rebuild all Erlang artifacts
cd $GOPATH/src/github.com/basho-labs/riak-mesos && TAGS='"rel native"' make rebuild_all_native
# Recompile the framework only
cd $GOPATH/src/github.com/basho-labs/riak-mesos && TAGS='"rel native"' make
# Create packages in the `_build/` directory
cd $GOPATH/src/github.com/basho-labs/riak-mesos && make package
```

### Start the Framework

```
./framework_linux_amd64 \
    -master=zk://localhost:2181/mesos \
    -zk=localhost:2181 \
    -name=riak \
    -user=root \
    -role=*
```

### Create a Cluster

```
./tools_linux_amd64 \
    -name=riak \
    -zk=localhost:2181 \
    -cluster-name=mycluster \
    -command="create-cluster"
```

Add Riak nodes

```
./tools_linux_amd64 \
    -name=riak \
    -zk=localhost:2181 \
    -cluster-name=mycluster \
    -command="add-nodes" \
    -nodes=1
```

### Start the Director

```
DIRECTOR_CLUSTER=mycluster DIRECTOR_FRAMEWORK=riak DIRECTOR_ZK=localhost:2181 ./director_linux_amd64
```

### Endpoints for Testing

* Director API: [http://192.168.0.30:9000/](http://192.168.0.30:9000/)
* Riak HTTP (Director Proxy): [http://192.168.0.30:8098/](http://192.168.0.30:8098/)
* Riak PB (Director Proxy) [http://192.168.0.30:8087/](http://192.168.0.30:8087/)
* Framework API: Dynamically assigned, check output of framework_linux_amd64

### Writing and Reading Data

```
curl -v -XPUT http://192.168.0.30:8098/buckets/test/keys/one -d "hello"
curl -v http://192.168.0.30:8098/buckets/test/keys/one
```

## Further Usage

See [MESOS_USAGE.md](MESOS_USAGE.md) for more information on how to use the binaries created in the `bin/` directory
