# Riak Mesos Framework

## Development

For initial setup of development environment, please follow the directions in
[DEVELOPMENT.md](https://github.com/basho-labs/riak-mesos/tree/master/docs/DEVELOPMENT.md).

### Build

Download dependencies and build the platform specific executables

```
make build
```

### Usage

#### Mac OS X

```
make run
```

or when running scheduler on mac os x and Mesos on vagrant

```
FRAMEWORK_USER=vagrant FRAMEWORK_HOSTNAME=33.33.33.1 FRAMEWORK_NAME=riak-mesos-go3 make run
```

or

```
./bin/framework_darwin_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -name=riak-mesos-go3 \
    -user=vagrant \
    -ip=33.33.33.1 \
    -hostname=33.33.33.1
```

##### Find the Framework URL

```
./bin/tools_darwin_amd64 -zk=33.33.33.2:2181 -command=get-url -name=riak-mesos-go3
```

This should return something like `http://33.33.33.1:57139/`

##### Add a node to a new cluster

```
curl -XPOST http://33.33.33.1:57173/clusters/mycluster
curl -XPOST http://33.33.33.1:57173/clusters/mycluster/nodes
```


#### Vagrant / Linux

Navigate to the shared directory:

```
cd /riak-mesos/src/github.com/basho-labs/riak-mesos
```

Run the scheduler

```
FRAMEWORK_USER=vagrant ARCH=linux_amd64 make run
```

or

```
./bin/framework_linux_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -name=riak-mesos-go3 \
    -user=vagrant \
    -ip=localhost \
    -hostname=33.33.33.2 \

```

## Architecture

![Architecture](docs/RiakMesosFramework.png)

### Client Interaction

For information on how client applications can interact with the Riak Mesos Framework, read the [HTTP-API.md](https://github.com/basho-labs/riak-mesos/tree/master/docs/HTTP-API.md) document.
