# Development Guide

## Environment Setup

### OSX

Install Go

```
brew update && brew upgrade
brew install go
```

Enable cross-compiling with linux/amd64

```
cd /usr/local/Cellar/go/1.4.2/libexec/src && \
    GOOS=linux GOARCH=amd64 ./make.bash --no-clean
```

Create a Go WORKSPACE

```
export GOPATH=~/go/riak-mesos
mkdir -p $GOPATH
export PATH=$PATH:$GOPATH/bin
cd $GOPATH
```

Optionally setup `.bashrc` or `.profile` or `.zprofile` by adding the following

```
export GOPATH=~/go/riak-mesos
export PATH=$PATH:$GOPATH/bin
```

Install Go tools

```
go get github.com/mitchellh/gox
go get github.com/tools/godep
go get github.com/satori/go.uuid
go get -u github.com/golang/protobuf/proto
go get github.com/golang/glog
go get github.com/kr/pretty
go get github.com/kr/text
go get github.com/Sirupsen/logrus
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/campoy/jsonenums
go get golang.org/x/tools/cmd/stringer
go get github.com/gorilla/mux
```

Setup initial directories

```
cd $GOPATH
mkdir -p src/github.com/mesos
mkdir -p src/github.com/basho
```

Download some deps

```
cd $GOPATH/src/github.com/
git clone git@github.com:basho-labs/riak-mesos.git basho-labs/riak-mesos
git clone https://github.com/mesos/mesos-go.git mesos/mesos-go
```

Build Mesos

```
cd $GOPATH/src/github.com/mesos/mesos-go
godep restore
go build ./...
```

## Vagrant Setup

On Mac OS X, configure a static IP for riak-mesos to bind to:

Add the following to your `/etc/hosts` file:

```
127.0.0.1	33.33.33.1
```

Start and connect to the Vagrant VM

```
vagrant up
vagrant ssh
```
