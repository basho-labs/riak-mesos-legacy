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

Install Go CLI tools

```
go get github.com/mitchellh/gox
go get github.com/tools/godep
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/campoy/jsonenums
go get golang.org/x/tools/cmd/stringer
go get -u github.com/golang/lint/golint
go get golang.org/x/tools/cmd/goimports
```

Setup riak-mesos and mesos-go

```
### Create src directories
cd $GOPATH
mkdir -p src/github.com/mesos
mkdir -p src/github.com/basho
### Download
cd $GOPATH/src/github.com/
git clone git@github.com:basho-labs/riak-mesos.git basho-labs/riak-mesos
git clone https://github.com/mesos/mesos-go.git mesos/mesos-go
### Build Mesos Go
cd $GOPATH/src/github.com/mesos/mesos-go
godep restore
go build ./...
### Build Riak Mesos Framework
cd $GOPATH/src/github.com/basho-labs/riak-mesos
godep restore
cd bin
go generate ../... && gox -osarch="linux/amd64" -osarch=darwin/amd64 ../...
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
