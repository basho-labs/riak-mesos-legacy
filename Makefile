BASE_DIR            = $(shell pwd)
ARCH               ?= darwin_amd64
SCHEDULER          ?= framework_${ARCH}
FRAMEWORK_NAME     ?= "riak-mesos-go3"
# FRAMEWORK_USER     ?= "vagrant"
# FRAMEWORK_HOSTNAME ?= "33.33.33.1"
FRAMEWORK_HOSTNAME ?= ""
FRAMEWORK_IP       ?= "33.33.33.1"
MESOS_MASTER       ?= "zk://33.33.33.2:2181/mesos"
ZOOKEEPER          ?= "33.33.33.2:2181"

.PHONY: all deps build rebuild doc fmt lint run test vet

all: build

deps:
	cd scheduler/data && $(MAKE)
	godep restore

generate:
	go generate ./...

rel: deps vet generate
	cd bin && gox -tags=rel -osarch="linux/amd64" -osarch=darwin/amd64 ../scheduler ../tools

dev: deps vet generate
	# cd bin && gox -tags=dev -osarch="linux/amd64" -osarch=darwin/amd64 ../executor/...
	cd bin && gox -tags=dev -osarch="linux/amd64" -osarch=darwin/amd64 ../...

run:
	cd bin && ./$(SCHEDULER) -master=$(MESOS_MASTER) -zk=$(ZOOKEEPER) -ip=$(FRAMEWORK_IP) -name=$(FRAMEWORK_NAME) -hostname=$(FRAMEWORK_HOSTNAME) -user=$(FRAMEWORK_USER)

test:
	go test ./...

# http://godoc.org/code.google.com/p/go.tools/cmd/vet
# go get code.google.com/p/go.tools/cmd/vet
vet:
	-go vet ./...

# https://github.com/golang/lint
# go get github.com/golang/lint/golint
lint:
	golint ./...

# http://golang.org/cmd/go/#hdr-Run_gofmt_on_package_sources
fmt:
	go fmt ./...
