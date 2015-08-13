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

For the framework to work properly in your environment, a config file should be
used. Here is a minimal example:

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

A successful run of the install command should look like the following:

```
The Mesos Riak Framework implementation is alpha and there may be bugs, incomplete features, incorrect documentation or other discrepancies.
Continue installing? [yes/no] yes
Installing package [riak] version [0.1.0]
Installing CLI subcommand for package [riak]
New command available: dcos riak
Thank you for installing Riak on Mesos. Visit https://github.com/basho-labs/riak-mesos for usage information.
```

### Building a cluster

```
dcos riak -name=riak -cluster-name=mycluster -command=create-cluster
dcos riak -name=riak -cluster-name=mycluster -command=create-cluster
```
