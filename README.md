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

##### Mac OS X

```
make run
```

or

```
./bin/framework_darwin_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -name=riak-mesos-go3 \
    -ip=33.33.33.1 \
    -hostname=33.33.33.1
```

##### Vagrant / Linux

Navigate to the shared directory:

```
cd /riak-mesos/src/github.com/basho-labs/riak-mesos
```

Run the scheduler

```
ARCH=linux_amd64 make run
```

or

```
./bin/framework_linux_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -name=riak-mesos-go3 \
    -hostname=33.33.33.2 \
    -ip=localhost
```

## Architecture

![Architecture](docs/RiakMesosFramework.png)

### Client Interaction

For information on how client applications can interact with the Riak Mesos Framework, read the [HTTP-API.md](https://github.com/basho-labs/riak-mesos/tree/master/docs/HTTP-API.md) document.
