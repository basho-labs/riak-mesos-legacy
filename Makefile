BASE_DIR         = $(shell pwd)
PACKAGE_VERSION ?= 0.1.0
### Framework / Executor Binary locations
FTAR  ?= $(BASE_DIR)/bin
ETAR  ?= $(BASE_DIR)/scheduler/data

### Framework Run Arguments
MAST  ?= "zk://33.33.33.2:2181/mesos"
ZOOK  ?= "33.33.33.2:2181"
FIP   ?= "33.33.33.1"
FNAM  ?= "riak-mesos-go3"
FHST  ?= ""
FUSR  ?= ""
# FHST ?= "33.33.33.1"
# FUSR     ?= "vagrant"

TAGS ?= dev
export TAGS

#.PHONY: all deps clean_deps build_executor rel dev clean run install-dcos-cli test vet lint fmt

.PHONY: all clean
all: .godep framework

clean:
	-rm -f bin/*_amd64

## Godeps target:
.godep: Godeps/Godeps.json
	godep restore
	touch .godep


### CEPMd start
.PHONY: cepm clean_cepmd erl_dist
erl_dist:
	cd erlang_dist && $(MAKE)

cepmd/cepm/data/erl_epmd.beam: erl_dist
cepmd/cepm/data/inet_tcp_dist.beam: erl_dist
cepmd/cepm/data/net_kernel.beam: erl_dist

cepmd/cepm/bindata_generated.go: cepmd/cepm/data/erl_epmd.beam cepmd/cepm/data/inet_tcp_dist.beam cepmd/cepm/data/net_kernel.beam
	go generate -tags=$(TAGS) ./cepmd/cepm

cepm: cepmd/cepm/bindata_generated.go

clean: clean_cepmd

clean_cepmd:
	-rm -f cepmd/cepm/bindata_generated.go


### CEPMD end

### Executor start
## Fake Target
.PHONY: executor clean_executor scheduler/data/executor_linux_amd64
clean: clean_executor
executor: scheduler/data/executor_linux_amd64

executor/bindata_generated.go: executor/data/advanced.config executor/data/riak.conf
	go generate -tags=$(TAGS) ./executor/...

scheduler/data/executor_linux_amd64: cepm executor/bindata_generated.go
	go build -o scheduler/data/executor_linux_amd64 -tags=$(TAGS) ./executor/

clean_executor:
	-rm -f executor/bindata_generated.go
	-rm -f scheduler/data/executor_linux_amd64
### Executor end

## Riak Explorer Start
.PHONY: riak_explorer clean_riak_explorer

riak_explorer/bindata_generated.go: riak_explorer/data/advanced.config riak_explorer/data/riak_explorer.conf
	go generate -tags=$(TAGS) ./riak_explorer/...
riak_explorer: artifacts riak_explorer/bindata_generated.go

clean_riak_explorer:
	-rm -f riak_explorer/bindata_generated.go
## Riak Explorer End

### Framework begin
.PHONY: framework clean_framework
clean: clean_framework
# Depends on artifacts, because it depends on scheduler which depends on artifacts
framework: cepm executor riak_explorer artifacts scheduler
	go build -o bin/framework_linux_amd64 -tags=$(TAGS) ./framework/
clean_framework:
	-rm -f bin/framework_linux_amd64
### Framework end


### Scheduler Begin
.PHONY: scheduler clean_scheduler

scheduler/bindata_generated.go: scheduler/data/executor_linux_amd64
	go generate -tags=$(TAGS) ./scheduler

scheduler: scheduler/bindata_generated.go

clean: clean_scheduler
clean_scheduler:
	-rm -rf scheduler/bindata_generated.go

### Scheduler End

## Artifact begin
.PHONY: artifacts clean_artifacts
clean: clean_artifacts
artifacts:
	cd artifacts && $(MAKE)

clean_artifacts:
	cd artifacts && $(MAKE) clean


## Artifact end
## Tools begin
.PHONY: tools clean_tools bin/tools_linux_amd64
bin/tools_linux_amd64:
	go build -o bin/tools_linux_amd64 -tags=$(TAGS) ./tools/

tools: bin/tools_linux_amd64

all: tools
clean_tools:
	-rm -rf bin/tools_linux_amd64
## Tools end

####
	-rm $(BASE_DIR)/bin/*_amd64
	-rm $(BASE_DIR)/scheduler/data/*_amd64
	-rm $(BASE_DIR)/scheduler/bindata_generated.go
	-rm $(BASE_DIR)/executor/bindata_generated.go
	-rm $(BASE_DIR)/riak_explorer/bindata_generated.go

run:
	cd $(BASE_DIR)/bin && ./framework_linux_amd64 \
		-master=$(MAST) \
		-zk=$(ZOOK) \
		-ip=$(FIP) \
		-name=$(FNAM) \
		-hostname=$(FHST) \
		-user=$(FUSR)

install-dcos-cli:
	mkdir -p $(BASE_DIR)/bin/dcos
	cd $(BASE_DIR)/bin/dcos && \
		sudo pip install virtualenv && \
		curl -O https://downloads.mesosphere.io/dcos-cli/install.sh && \
		sudo /bin/bash install.sh . http://33.33.33.2
	echo "\n\nPlease run the following command to finish installation:\n\nsource $(BASE_DIR)/bin/dcos/bin/env-setup\n\nsudo pip install --upgrade cli/\n"

package-rel: package-framework package-director package-dcos package-repo
package-framework:
	cd $(BASE_DIR)/build && make -f coreos.make package-framework
package-director:
	cd $(BASE_DIR)/build && make -f coreos.make package-director
package-dcos:
	cd $(BASE_DIR)/build && make -f coreos.make package-dcos
package-repo:
	cd $(BASE_DIR)/build && make -f coreos.make package-repo
sync-framework:
	cd $(BASE_DIR)/build && make -f coreos.make sync-framework
sync-director:
	cd $(BASE_DIR)/build && make -f coreos.make sync-director
sync-dcos:
	cd $(BASE_DIR)/build && make -f coreos.make sync-dcos
sync-repo:
	cd $(BASE_DIR)/build && make -f coreos.make sync-repo

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
