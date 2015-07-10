BASE_DIR            = $(shell pwd)

### System Architecture
FRAMEWORK_ARCH     ?= darwin_amd64
EXECUTOR_ARCH      ?= linux_amd64
FRAMEWORK_GOX_ARCH ?= "darwin/amd64"
EXECUTOR_GOX_ARCH ?= "linux/amd64"

### Binary locations
FRAMEWORK_TARGET   ?= $(BASE_DIR)/bin
EXECUTOR_TARGET    ?= $(BASE_DIR)/scheduler/data

### Framework Run Arguments
MESOS_MASTER       ?= "zk://33.33.33.2:2181/mesos"
ZOOKEEPER          ?= "33.33.33.2:2181"
FRAMEWORK_IP       ?= "33.33.33.1"
FRAMEWORK_NAME     ?= "riak-mesos-go3"
FRAMEWORK_HOSTNAME ?= ""
FRAMEWORK_USER     ?= ""
# FRAMEWORK_HOSTNAME ?= "33.33.33.1"
# FRAMEWORK_USER     ?= "vagrant"

.PHONY: all deps clean_deps build_executor rel dev clean run test vet lint fmt

all: dev

deps:
	godep restore
	cd $(BASE_DIR)/scheduler/data && $(MAKE)

clean_deps:
	rm $(BASE_DIR)/scheduler/data/*.tar.gz

build_executor:
	go generate ./executor/...
	gox \
		-osarch=$(EXECUTOR_GOX_ARCH) \
		-output="$(EXECUTOR_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./executor/...

rel: clean deps vet build_executor
	go generate -tags=rel ./...
	gox \
		-tags=rel \
		-osarch=$(FRAMEWORK_GOX_ARCH) \
		-output="$(FRAMEWORK_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

dev: clean deps vet build_executor
	go generate -tags=dev ./...
	gox \
		-tags=dev \
		-osarch=$(FRAMEWORK_GOX_ARCH) \
		-output="$(FRAMEWORK_TARGET)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

clean:
	-rm $(BASE_DIR)/bin/*_amd64
	-rm $(BASE_DIR)/scheduler/data/*_amd64
	-rm $(BASE_DIR)/scheduler/bindata_generated.go
	-rm $(BASE_DIR)/executor/bindata_generated.go

run:
	cd $(BASE_DIR)/bin && ./framework_$(FRAMEWORK_ARCH) \
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
