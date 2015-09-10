# Development Guide

## Quickstart

To build the framework and get it running in a Mesos vagrant environment on Mac OSX, follow these steps:

### Build

```
make
```

### Bring up Mesos

```
vagrant plugin install vagrant-hostmanager
vagrant up
vagrant reload
```

### Start the framework

```

```


## Environment

For vagrant or regular Ubuntu 14.04, go to [https://github.com/basho-labs/vagrant-riak-mesos](https://github.com/basho-labs/vagrant-riak-mesos) and follow the directions to set up a development environment.

## Build

Download dependencies and build the platform specific executables

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && make
```

## Usage

See [MESOS_USAGE.md](MESOS_USAGE.md) for information on how to use the binaries created in the `bin/` directory
