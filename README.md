# Riak Mesos Framework

## Development

For initial setup of development environment, please follow the directions in
[DEVELOPMENT.md](https://github.com/basho-labs/riak-mesos/tree/master/docs/DEVELOPMENT.md).

Build Bletchley

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos/bin
go generate ../... && gox -osarch="linux/amd64" -osarch=darwin/amd64 ../...
```

## Running the Framework

Mac OS X

```
./scheduler_darwin_amd64 -master=zk://33.33.33.2:2181/mesos -zk=33.33.33.2:2181 -hostname=33.33.33.1 -ip=33.33.33.1
```

Vagrant / Linux

```
cd /riak-mesos/src/github.com/basho-labs/riak-mesos/bin
./scheduler_linux_amd64 -master=zk://33.33.33.2:2181/mesos -zk=33.33.33.2:2181 -hostname=33.33.33.2 -ip=localhost
```
