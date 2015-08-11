# Riak Mesos Framework Director

Due to the nature of Apache Mesos and the potential for Riak nodes to come and
go on a regular basis, client applications using a Mesos based cluster must
be kept up to date on the cluster's current state. Instead of requiring this
intelligence to be built into Riak client libraries, a smart proxy application named
`Director` has been created which can run alongside client applications.

The Director communicates with Zookeeper to keep up to date with Riak cluster changes.
The Director in turn will update it's list of balanced Riak connections.

![Director](RiakMesosControlFrame.png)

## Marathon Setup

Example `marathon.json`

```
{
    "id": "/riak/director",
    "cmd": "bin/director console -noinput",
    "cpus": 0.1,
    "mem": 100.0,
    "ports": [8098, 8087, 0],
    "instances": 1,
    "env": {
        "DIRECTOR_ZK": "master.mesos:2181"
        "DIRECTOR_FRAMEWORK": "riak-mesos-go"
        "DIRECTOR_CLUSTER": "mycluster"
    },
    "uris": [
        "http://riak-tools.s3.amazonaws.com/riak_mesos_director_linux_amd64.tar.gz"
    ],
    "healthChecks": [
        {
            "protocol": "HTTP",
            "path": "/health",
            "gracePeriodSeconds": 3,
            "intervalSeconds": 10,
            "portIndex": 2,
            "timeoutSeconds": 10,
            "maxConsecutiveFailures": 3
        }
    ]
}
```

## Manual Setup

### Download

Download and Extract [riak_mesos_director_linux_amd64.tar.gz](http://riak-tools.s3.amazonaws.com/riak_mesos_director_linux_amd64.tar.gz) or [riak_mesos_director_darwin_amd64.tar.gz](http://riak-tools.s3.amazonaws.com/riak_mesos_director_darwin_amd64.tar.gz).

### Configure

Change `etc/director.conf` to match your environment

```
listener.web = on
listener.web.http = 0.0.0.0:9000
listener.proxy.http = 0.0.0.0:8098
listener.proxy.protobuf = 0.0.0.0:8087
zookeeper.address = 33.33.33.2:2181
framework.name = riak-mesos-go
framework.cluster = mycluster
```

### Running the Director

Start

```
./bin/director start
```

Stop

```
./bin/director start
```

Logs are located at `log/console.log`.

## CLI

Running `./bin/director-admin` will list available usage:

```
Usage: Usage: director-admin <sub-command>

  Sub-commands:
    status                            Display current information about the director
    configure -f framework -c cluster Update and resynchronize proxy using the specified framework and cluster
    list-frameworks                   List of running Riak Mesos Framework instance names
    list-clusters                     List of running Riak clusters in the configured framework
    list-nodes                        List of running Riak nodes in the configured cluster
```

## HTTP API

Functionality available in the `director-admin` tool is also available via an HTTP API.

Name | Method | Path | Description
--- | --- | --- | ---
status | **GET** | `/status` | Display current information about the director
configure | **PUT** | `/frameworks/{framework}/clusters/{cluster}` | Changes the framework and cluster names
listFrameworks | **GET** | `/frameworks` | Lists the currently known Riak Mesos Framework instances
listClusters | **GET** | `/clusters` | Lists the clusters for the configured framework
listNodes | **GET** | `/nodes` | Lists the nodes for the configured cluster
healthCheck | **GET** | `/health` | Simple status check for other services like Marathon
