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
FRAMEWORK_TARGET   ?= bin
EXECUTOR_TARGET    ?= scheduler/data

.PHONY: all deps build_executor rel dev clean run test vet lint fmt

all: dev

deps:
	godep restore
	cd scheduler/data && $(MAKE)

build_executor:
	go generate ./executor/...
	gox \
		-osarch="linux/amd64" \
		-osarch="darwin/amd64" \
		-output="$(EXECUTOR_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./executor/...

rel: deps vet build_executor
	go generate -tags=rel ./...
	gox \
		-tags=rel \
		-osarch="linux/amd64" \
		-osarch="darwin/amd64" \
		-output="$(FRAMEWORK_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

dev: deps vet build_executor
	go generate -tags=dev ./...
	gox \
		-tags=dev \
		-osarch="linux/amd64" \
		-osarch="darwin/amd64" \
		-output="$(FRAMEWORK_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

clean:
	-rm bin/*_amd64
	-rm scheduler/data/*_amd64
	-rm scheduler/bindata_generated.go
	-rm executor/bindata_generated.go

run:
	cd bin && ./$(SCHEDULER) \
		-master=$(MESOS_MASTER) \
		-zk=$(ZOOKEEPER) \
		-ip=$(FRAMEWORK_IP) \
		-name=$(FRAMEWORK_NAME) \
		-hostname=$(FRAMEWORK_HOSTNAME) \
		-user=$(FRAMEWORK_USER)

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
