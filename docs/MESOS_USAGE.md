# Riak Framework on Open Source Mesos

## Package Downloads

### CoreOS / Default DCOS Packages / Multi-platform

These packages include an Ubuntu image and they utilize chroot to run Erlang applications for mutiple platform support.

* Riak Mesos Framework [riak_mesos_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/coreos/riak_mesos_linux_amd64_0.1.1.tar.gz)
* Riak Mesos Director [riak_mesos_director_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/coreos/riak_mesos_director_linux_amd64_0.1.1.tar.gz)

### Ubuntu (14.04) Packages

These packages included Ubuntu Trusty flavors of the embedded Erlang applications

* Riak Mesos Framework [riak_mesos_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/ubuntu/riak_mesos_linux_amd64_0.1.1.tar.gz)
* Riak Mesos Director [riak_mesos_director_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/ubuntu/riak_mesos_director_linux_amd64_0.1.1.tar.gz)

### CentOS (7.0) Packages

These packages included CentOS flavors of the embedded Erlang applications

* Riak Mesos Framework [riak_mesos_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/centos/riak_mesos_linux_amd64_0.1.1.tar.gz)
* Riak Mesos Director [riak_mesos_director_linux_amd64_0.1.1.tar.gz](http://riak-tools.s3.amazonaws.com/riak-mesos/centos/riak_mesos_director_linux_amd64_0.1.1.tar.gz)

## Installation

The Riak Mesos Framework can be configured in a few different ways depending on the restraints of the Mesos cluster.

### Marathon Usage

Sample Riak Mesos Framework `marathon.json`: [../mararthon.json](../marathon.json).

**Note**: You may need to replace the `"uris": [".../coreos/..."],` portion with your platform (centos|ubuntu)

Sample Riak Mesos Director `marathon.json`: [../director.mararthon.json](../director.marathon.json).

**Note**: You may need to replace the `"uris": [".../coreos/..."],` portion with your platform (centos|ubuntu)

After modifying `marathon.json` and/or `director.marathon.json` appropriately for your environment, run the following to add them as Marathon apps:

```
curl -v -XPOST http://master.mesos:8080/v2/apps -d @marathon.json
curl -v -XPOST http://master.mesos:8080/v2/apps -d @director.marathon.json
```

Once the framework is up and running in Mesos, create and scale a cluster using `tools_linux_amd64`, the instructions are below in the "Manual Usage, Create a cluster" section.

### Manual Usage

#### Start the framework

Download and extract the Riak Mesos Framework (`riak_mesos_linux_amd64_0.1.1.tar.gz`, links above), and start the framework with an incantation similar to this:

```
./framework_linux_amd64 \
    -master=zk://master.mesos:2181/mesos \
    -zk=master.mesos:2181 \
    -name=riak \
    -user=root \
    -role=*
```

Included with the framework tarball is a CLI tool named `tools_linux_amd64` which can be used to perform a variety of tasks on a running Riak Mesos Framework instance. Following are some usage instructions.

Configure a few environment variables matching your setup for convenience.

#### Create a cluster

Download and extract the Riak Mesos Framework (`riak_mesos_linux_amd64_0.1.1.tar.gz`, links above), and execute the below commands using the included `tools_linux_amd64`:

```
./tools_linux_amd64 \
    -name=riak \
    -zk=master.mesos:2181 \
    -cluster-name=mycluster \
    -command="create-cluster"
```

#### Add Riak nodes

```
./tools_linux_amd64 \
    -name=riak \
    -zk=master.mesos:2181 \
    -cluster-name=mycluster \
    -command="add-nodes" \
    -nodes=5
```

#### Additional HTTP Endpoints

Get the base URL for the Riak Mesos Framework [HTTP API](docs/HTTP-API.md) endpoints for more ways to interact with the framework.

```
./tools_linux_amd64 -name=$NAME -zk=$ZK -command="get-url"
```

#### Start the director

Download and extract the Riak Mesos Director for your platform (`riak_mesos_director_linux_amd64_0.1.1.tar.gz`, links above), and start it with an incantation similar to this:

```
DIRECTOR_CLUSTER=mycluster DIRECTOR_FRAMEWORK=riak DIRECTOR_ZK=master.mesos:2181 ./director_linux_amd64
```

Starting the director should give you access to a number of endpoints:

* Balanced Riak HTTP [http://master.mesos:8098](http://master.mesos:8098)
* Balanced Riak Protobuf [http://master.mesos:8087](http://master.mesos:8087)
* Director HTTP API [http://master.mesos:9000](http://master.mesos:9000)

These ports are the defaults, and will be dynamically assigned when using Marathon or DCOS.

## Uninstalling

To remove the proxy and framework from marathon, run these command:

```
curl -XDELETE http://master.mesos:8080/v2/apps/riak
curl -XDELETE http://master.mesos:8080/v2/apps/riak-director
```

**Note:** Currently, Zookeeper entries are left behind by the framework even after uninstall. To completely remove these entries, use the tools binary included in the framework package download (links for `tools_linux_amd64` above). Execute the following command to remove the framework ZK references:

```
./tools_linux_amd64 -zk=master.mesos:2181 -name=riak -command="delete-framework"
```

Replace "-name=riak" with the framework name if different than "riak".