# Riak Mesos Framework on DCOS

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

### Install the Riak service

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

### Uninstalling the director proxy

To remove the proxy from marathon, run this command:

```
dcos riak proxy uninstall
```

### Uninstalling the framework

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
