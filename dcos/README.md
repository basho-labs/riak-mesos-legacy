# Riak Mesos Framework on DCOS

[DCOS](http://docs.mesosphere.com/) support is still in development and is not
supported on all platforms.

## Compatibility

CoreOS based DCOS clusters have known compatibility issues with the
Riak Mesos Framework.

Clusters running RHEL and Debian variants should work

## Installation

### Setup DCOS CLI

First, the Riak DCOS package needs to be added to sources.

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
        "framework-name": "riak-cluster-2"
    }
}
```

More examples can be found in [dcos-riak-cluster-1.json](dcos-riak-cluster-1.json), [dcos-riak-cluster-2.json](dcos-riak-cluster-2.json), and [dcos-riak-cluster-3.json](dcos-riak-cluster-3.json).

### Install the Riak service

```
dcos package update
dcos package install riak --options=dcos-riak.json
```

### Accessing Your Riak Nodes

The [Riak Mesos Director](http://github.com/basho-labs/riak-mesos-director) application can be easily installed on your DCOS cluster
with these commands:

```
dcos riak --generate-director-config mycluster master.mesos:2181 \
    > director.marathon.json
dcos marathon app add director.marathon.json
```

Once it is up and running, explore your Riak cluster using the following command:

```
dcos riak --get-director-urls <public-node-dns>
```

The output should look something like this:

```
Load Balanced Riak Cluster (HTTP)
    http://<public-node-dns>:10002

Load Balanced Riak Cluster (Protobuf)
    http://<public-node-dns>:10003

Riak Mesos Director API (HTTP)
    http://<public-node-dns>:10004

Riak Explorer and API (HTTP)
    http://<public-node-dns>:10005
```

### Uninstalling the framework

All of the tasks created by Riak and the Director applications can be killed
with the following:

```
dcos marathon app remove riak-director
dcos package uninstall riak
```

Currently, Zookeeper entries are left behind by the framework even after uninstall.
To completely remove these entries, use a Zookeeper client to delete the relevant
nodes.

To remove just one framework instance, delete the `/riak/frameworks/riak` node.

If you have changed the value of `framework-name` in your config, the last
`/riak` will change.

To remove all framework instances, delete the `/riak` node in Zookeeper.
