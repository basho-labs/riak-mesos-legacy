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

### Download code and deps
mkdir -p $GOPATH/src/github.com/basho-labs
rm -rf $GOPATH/src/github.com/basho-labs/riak-mesos
git clone https://github.com/basho-labs/riak-mesos.git $GOPATH/src/github.com/basho-labs/riak-mesos
cd $GOPATH/src/github.com/basho-labs/riak-mesos && godep restore

### Mesos Go
rm -rf $GOPATH/src/github.com/mesos/mesos-go
git clone https://github.com/mesos/mesos-go.git $GOPATH/src/github.com/mesos/mesos-go
go get github.com/gogo/protobuf/protoc-gen-gogo
go get github.com/gogo/protobuf/protoc-gen-gogofast
cd $GOPATH/src/github.com/mesos/mesos-go/mesosproto && \
  protoc --proto_path=${GOPATH}/src:${GOPATH}/src/github.com/gogo/protobuf/protobuf:. --gogofast_out=. *.proto
