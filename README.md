# Riak Mesos Framework

An [Apache Mesos](http://mesos.apache.org/) framework for [Riak KV](http://basho.com/products/riak-kv/), a distributed NoSQL key-value data store that offers high availability, fault tolerance, operational simplicity, and scalability.

## Quick Links

* [Installation](#installation)
    * [Setup DCOS CLI](#setup-dcos-cli)
    * [Create a Configuration File](#create-a-configuration-file)
    * [Install the Riak Service](#install-the-riak-service)
    * [CLI Usage](#cli-usage)
    * [Add Riak Nodes](#add-riak-nodes)
    * [Accessing Your Riak Nodes](#accessing-your-riak-nodes)
    * [Uninstalling](#uninstalling)
* [Architecture](#architecture)
    * [Scheduler](#scheduler)
    * [Director](#director)
* [DCOS Resources](#mesos-users)
* [Development / Contributing](#development--contributing)

### Other Documentation

* [Development Setup](docs/DEVELOPMENT-SETUP.md)
* [Development Guide](docs/DEVELOPMENT.md)
* [Director](docs/DIRECTOR.md)
* [HTTP API](docs/HTTP-API.md)
* [Mesos Usage](docs/MESOS-USAGE.md)

[DCOS](http://docs.mesosphere.com/) support is still in development and is not
supported on all platforms.

## Installation

### Setup DCOS CLI

First, the Riak DCOS package needs to be added to sources.

This step will no longer be necessary once the framework passes DCOS certification.

```
dcos config set package.sources '["http://riak-tools.s3.amazonaws.com/riak-mesos/dcos-repo-0.1.0.zip"]'
```

### Create a Configuration File

For the framework to work properly in your environment, a custom config file
may need to be used. Here is a minimal example:

`dcos-riak.json`

```
{
    "riak": {
        "master": "zk://master.mesos:2181/mesos",
        "zk": "master.mesos:2181",
        "user": "root",
        "framework-name": "riak"
    }
}
```

More examples can be found in [dcos-riak-cluster-1.json](dcos-riak-cluster-1.json), [dcos-riak-cluster-2.json](dcos-riak-cluster-2.json), and [dcos-riak-cluster-3.json](dcos-riak-cluster-3.json).

### Install the Riak Service

```
dcos package update
dcos package install riak --options=dcos-riak.json
```

### CLI Usage

```
Command line utility for the Riak Mesos Framework / DCOS Service.
This utility provides tools for modifying and accessing your Riak
on Mesos installation.

Usage: dcos riak <subcommands> [options]

Subcommands:
    cluster list
    cluster create
    node list
    node add [--nodes <number>]
    proxy config [--zk <host:port>]
    proxy install [--zk <host:port>]
    proxy uninstall
    proxy endpoints [--public-dns <host>]

Options (available on most commands):
    --cluster <cluster-name>      Default: riak-cluster
    --framework <framework-name>  Default: riak
    --debug
    --help
    --info
    --version
```

### Add Riak Nodes

Create a 3 node cluster named 'riak-cluster' (this is the default name).

```
dcos riak cluster create
dcos riak node add --nodes 3
```

Create a second 1 node cluster named 'riak-test-cluster'.

```
dcos riak cluster create --cluster riak-test-cluster
dcos riak node add --cluster riak-test-cluster
```

### Accessing Your Riak Nodes

The [Riak Mesos Director](http://github.com/basho-labs/riak-mesos-director) smart proxy can be easily installed on your DCOS cluster with these commands:

```
dcos riak proxy install
```

Once it is up and running, explore your proxy and Riak cluster using the following command:

```
dcos riak proxy endpoints --public-dns <host>
```

The output should look something like this:

```
Load Balanced Riak Cluster (HTTP)
    http://<host>:10002

Load Balanced Riak Cluster (Protobuf)
    http://<host>:10003

Riak Mesos Director API (HTTP)
    http://<host>:10004

Riak Explorer and API (HTTP)
    http://<host>:10005
```

### Uninstalling

To remove the proxy from marathon, run this command:

```
dcos riak proxy uninstall
```

All of the tasks created by the Riak framework can be killed with the following:

```
dcos package uninstall riak
```

***Note:*** Currently, Zookeeper entries are left behind by the framework even after uninstall.
To completely remove these entries, use a Zookeeper client to delete the relevant
nodes.

To remove just one framework instance, delete the `/riak/frameworks/riak` node.

If you have changed the value of `framework-name` in your config, the last
`/riak` will change.

To remove all framework instances, delete the `/riak` node in Zookeeper.

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

## Mesos Users

The Framework can be used on Mesos clusters without DCOS as well. Follow the
instructions in [docs/MESOS_USAGE.md](docs/MESOS_USAGE.md) if you are not a DCOS user.

## Development / Contributing

For initial setup of development environment, please follow the directions in
[docs/DEVELOPMENT-SETUP.md](docs/DEVELOPMENT-SETUP.md). For further build and testing information,
visit [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
