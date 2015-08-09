# Development Guide

For first time dev environment setup, visit [DEVELOPMENT-SETUP.md](DEVELOPMENT-SETUP.md)

### Build

Download dependencies and build the platform specific executables

```
make dev
```

To build a complete framework package with embedded executor and riak packages:

```
FARC="linux_amd64" FGARC="linux/amd64" make rel
```

### Usage

#### Mac OS X

```
make run
```

or when running scheduler on mac os x and Mesos on vagrant

```
FUSR=vagrant FHST=33.33.33.1 FNAM=riak-mesos-go3 make run
```

or

```
./bin/framework_darwin_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -id=riak-mesos-go3 \
    -user=vagrant \
    -ip=33.33.33.1 \
    -hostname=33.33.33.1
```

##### Add some nodes to the cluster

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

#### Vagrant / Linux

Navigate to the shared directory:

```
cd /vagrant
```

Run the scheduler

```
FUSR=vagrant ARCH=linux_amd64 make run
```

or

```
./bin/framework_linux_amd64 \
    -master=zk://33.33.33.2:2181/mesos \
    -zk=33.33.33.2:2181 \
    -id=riak-mesos-go3 \
    -user=vagrant \
    -ip=localhost \
    -hostname=33.33.33.2 \
```
