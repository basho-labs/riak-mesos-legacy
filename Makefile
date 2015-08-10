BASE_DIR         = $(shell pwd)
BUILD_DIR       ?= _build
PACKAGE_VERSION ?= 0.1.0

### Framework / Executor Architecture
FARC  ?= darwin_amd64
EARC  ?= linux_amd64
FGARC ?= "darwin/amd64"
EGARC ?= "linux/amd64"

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

.PHONY: all deps clean_deps build_executor rel dev clean run install-dcos-cli test vet lint fmt

all: dev

deps:
	godep restore
	cd $(BASE_DIR)/scheduler/data && $(MAKE)
	cd $(BASE_DIR)/riak_explorer/data && $(MAKE)
	cd $(BASE_DIR)/erlang_dist && $(MAKE)

clean_deps:
	-rm $(BASE_DIR)/scheduler/data/*.tar.gz
	-rm $(BASE_DIR)/riak_explorer/data/*.tar.gz
	cd $(BASE_DIR)/erlang_dist/ && $(MAKE) clean

build_cepmd_dev:
	go generate ./cepmd/...

build_cepmd_rel:
	go generate -tags=rel ./cepmd/...

build_executor:
	go generate ./executor/...
	gox \
		-osarch=$(EGARC) \
		-output="$(ETAR)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./executor/

rel: clean deps vet build_cepmd_rel build_executor
	go generate -tags=rel ./...
	gox \
		-tags=rel \
		-osarch=$(FGARC) \
		-output="$(FTAR)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

rel-tools: vet
	gox \
		-tags=rel \
		-osarch=$(FGARC) \
		-output="$(FTAR)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./tools/...

dev: clean deps vet build_cepmd_dev build_executor
	go generate -tags=dev ./...
	gox \
		-tags=dev \
		-osarch=$(FGARC) \
		-output="$(FTAR)/{{.Dir}}_{{.OS}}_{{.Arch}}" \
		-rebuild \
		./framework/... ./tools/...

clean:
	-rm $(BASE_DIR)/bin/*_amd64
	-rm $(BASE_DIR)/scheduler/data/*_amd64
	-rm $(BASE_DIR)/scheduler/bindata_generated.go
	-rm $(BASE_DIR)/executor/bindata_generated.go
	-rm $(BASE_DIR)/riak_explorer/bindata_generated.go

run:
	cd $(BASE_DIR)/bin && ./framework_$(FARC) \
		-master=$(MAST) \
		-zk=$(ZOOK) \
		-ip=$(FIP) \
		-name=$(FNAM) \
		-hostname=$(FHST) \
		-user=$(FUSR)

marathon-run:
	curl -XPOST -v -H 'Content-Type: application/json' -d @marathon.json 'http://33.33.33.2:8080/v2/apps'
marathon-run-director:
	curl -XPOST -v -H 'Content-Type: application/json' -d @director.marathon.json 'http://33.33.33.2:8080/v2/apps'

mesos-kill:
	curl -XPOST -v 'http://33.33.33.2:5050/master/shutdown' --data "frameworkId=riak-mesos-go24"

marathon-kill:
	curl -XDELETE -v 'http://33.33.33.2:8080/v2/apps/riak'

install-dcos-cli:
	mkdir -p $(BASE_DIR)/bin/dcos
	cd $(BASE_DIR)/bin/dcos && \
		sudo pip install virtualenv && \
		curl -O https://downloads.mesosphere.io/dcos-cli/install.sh && \
		sudo /bin/bash install.sh . http://33.33.33.2
	echo "\n\nPlease run the following command to finish installation:\n\nsource $(BASE_DIR)/bin/dcos/bin/env-setup\n\nsudo pip install --upgrade cli/\n"

vagrant-mesos:
	cd vagrant/ubuntu/trusty64/riak-mesos && vagrant up

package-deps:
	cd vagrant/ubuntu/trusty64/dependencies && make
deploy-deps:
	cd vagrant/ubuntu/trusty64/dependencies && make deploy

package-rel:
	-rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)/riak_mesos_framework
	cp bin/framework_linux_amd64 $(BUILD_DIR)/riak_mesos_framework/
	cp bin/tools_linux_amd64 $(BUILD_DIR)/riak_mesos_framework/
	cp packages/0/*.json $(BUILD_DIR)/riak_mesos_framework/
	echo "Thank you for downloading Riak Mesos Framework. Please visit https://github.com/basho-labs/riak-mesos for usage information." > $(BUILD_DIR)/riak_mesos_framework/INSTALL.txt
	cd $(BUILD_DIR) && tar -zcvf riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz riak_mesos_framework

deploy:
	cd $(BUILD_DIR) && s3cmd put --acl-public riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz s3://riak-tools/riak-mesos/

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
