BASE_DIR         = $(shell pwd)
export TAGS     ?= rel
PACKAGE_VERSION ?= 0.3.1
BUILD_DIR       ?= $(BASE_DIR)/_build
export PROJECT_BASE    ?= riak-mesos
export DEPLOY_BASE     ?= riak-tools/$(PROJECT_BASE)
export DEPLOY_OS       ?= ubuntu
export OS_ARCH		   ?= linux_amd64
export EXECUTOR_LANG ?= golang

.PHONY: all clean clean_bin package clean_package sync
all: clean_bin framework
clean: clean_package clean_bin
package: clean_package

## Godeps begin
.godep: Godeps/Godeps.json
	godep restore
	touch .godep
## Godeps end

### Framework begin
.PHONY: framework clean_framework
# Depends on artifacts
.bin.framework_$(OS_ARCH):
	go build -o bin/framework_$(OS_ARCH) -tags="$(TAGS) $(EXECUTOR_LANG)" ./framework/
	$(shell touch .bin.framework_$(OS_ARCH))
framework: .godep executor artifacts .bin.framework_$(OS_ARCH)
clean_bin: clean_framework
clean_framework:
	-rm -f .bin.framework_$(OS_ARCH) bin/framework_$(OS_ARCH)
### Framework end

### Executor begin
.PHONY: executor clean_executor .scheduler.data.executor_$(OS_ARCH)
executor: .scheduler.data.executor_$(OS_ARCH)
.scheduler.data.executor_$(OS_ARCH): cepm
	GOOS=linux GOARCH=amd64 go build -o artifacts/data/executor_$(OS_ARCH) -tags="$(TAGS) $(EXECUTOR_LANG)" ./executor/
	$(shell touch .scheduler.data.executor_$(OS_ARCH))
clean_bin: clean_executor
clean_executor:
	-rm -f .scheduler.data.executor_$(OS_ARCH) artifacts/data/executor_$(OS_ARCH)
### Executor end

### CEPMd begin
.PHONY: cepm clean_cepmd erl_dist
erl_dist:
	cd erlang_dist && $(MAKE)
.cepmd.cepm.bindata_generated: erl_dist
	go generate -tags="$(TAGS) $(EXECUTOR_LANG)" ./cepmd/cepm
	go build -o artifacts/data/cepmd_$(OS_ARCH) -tags="$(TAGS) $(EXECUTOR_LANG)" ./cepmd/
	$(shell touch .cepmd.cepm.bindata_generated)
cepm: .cepmd.cepm.bindata_generated
clean_bin: clean_cepmd
clean_cepmd:
	-rm -f .cepmd.cepm.bindata_generated cepmd/cepm/bindata_generated.go artifacts/data/cepmd_$(OS_ARCH)
### CEPMd end

### Artifact begin
.PHONY: artifacts clean_artifacts
artifacts:
	cd artifacts/data && $(MAKE)
	go generate -tags="$(TAGS) $(EXECUTOR_LANG)" ./artifacts
clean_bin: clean_artifacts_bin
clean_artifacts_bin:
	cd artifacts/data && $(MAKE) clean_bin
clean: clean_artifacts
clean_artifacts:
	cd artifacts/data && $(MAKE) clean
	-rm -rf artifacts/bindata_generated.go
### Artifact end

### Go Tools begin
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
### Go Tools end

### Framework Package begin
.PHONY: package_framework sync_framework clean_framework_package
package: package_framework
package_framework: $(BUILD_DIR)/riak_mesos_$(OS_ARCH)_$(PACKAGE_VERSION).tar.gz
$(BUILD_DIR)/riak_mesos_$(OS_ARCH)_$(PACKAGE_VERSION).tar.gz:
	-rm -rf $(BUILD_DIR)/riak_mesos_framework
	mkdir -p $(BUILD_DIR)/riak_mesos_framework
	cp bin/framework_$(OS_ARCH) $(BUILD_DIR)/riak_mesos_framework/
	echo "Thank you for downloading Riak Mesos Framework. Please visit https://github.com/basho-labs/riak-mesos for usage information." > $(BUILD_DIR)/riak_mesos_framework/INSTALL.txt
	cd $(BUILD_DIR) && tar -zcvf riak_mesos_$(OS_ARCH)_$(PACKAGE_VERSION).tar.gz riak_mesos_framework
sync: sync_framework
sync_framework:
	cd $(BUILD_DIR)/ && \
		s3cmd put --acl-public riak_mesos_$(OS_ARCH)_$(PACKAGE_VERSION).tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/
clean_package: clean_framework_package
clean_framework_package:
	-rm $(BUILD_DIR)/riak_mesos_$(OS_ARCH)_$(PACKAGE_VERSION).tar.gz
### Framework Package end
