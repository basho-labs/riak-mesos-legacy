# Riak Mesos Framework

## Development

For initial setup of development environment, please follow the directions in
[DEVELOPMENT.md](https://github.com/basho/bletchley/tree/master/docs/DEVELOPMENT.md).

Build Bletchley

```
cd $GOPATH/src/github.com/basho/bletchley/bin
gox -osarch="linux/amd64" -osarch=darwin/amd64 ../...
```

## Running Bletchley

```
./scheduler_darwin_amd64 -master=33.33.33.2:5050 -zk=33.33.33.2:2181 -host=33.33.33.1 # Mac OS X
#./scheduler_linux_amd64 -master=33.33.33.2:5050 -zk=33.33.33.2:2181 -host=33.33.33.2 # Linux / Vagrant
```
