#!/bin/bash

# Go tools and deps
go get github.com/mitchellh/gox
go get github.com/tools/godep
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/campoy/jsonenums
go get golang.org/x/tools/cmd/stringer
go get -u github.com/golang/lint/golint
go get golang.org/x/tools/cmd/goimports

go get github.com/golang/glog
go get golang.org/x/net/context
go get github.com/gogo/protobuf/proto
go get github.com/stretchr/testify/mock
go get github.com/samuel/go-zookeeper/zk
go get github.com/pborman/uuid
go get github.com/stretchr/testify/assert

# Mesos go
# Using Go1.5 this can be tricky... May need to use go1.4, create protos, then switch to go1.5 and repeat these steps
gvm use go1.4
go get github.com/mesos/mesos-go
go get github.com/gogo/protobuf/protoc-gen-gogo
mkdir -p $GOPATH/src/github.com/mesos
rm -rf $GOPATH/src/github.com/mesos/mesos-go
git clone https://github.com/mesos/mesos-go.git $GOPATH/src/github.com/mesos/mesos-go
cd $GOPATH/src/github.com/mesos/mesos-go/mesosproto && make
gvm use go1.5
go get github.com/gogo/protobuf/protoc-gen-gogo
cd $GOPATH/src/github.com/mesos/mesos-go/mesosproto && make

### Download code and deps
mkdir -p $GOPATH/src/github.com/basho-labs
rm -rf $GOPATH/src/github.com/basho-labs/riak-mesos
# Next step not necessary when cloning riak-mesos code from git
cd $GOPATH/src/github.com/basho-labs/ && ln -fs /vagrant riak-mesos
