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

The Riak Mesos Director application can be easily installed on your DCOS cluster
with these commands:

```
dcos riak --generate-director-config mycluster master.mesos:2181 \
    > director.marathon.json
dcos marathon app add director.marathon.json
```

After the application boots up, Riak can be accessed using the first port given
by Marathon:
(TODO: add command to show director endpoints using marathon API)
