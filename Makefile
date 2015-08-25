BASE_DIR         = $(shell pwd)
PACKAGE_VERSION ?= 0.1.0
export TAGS     ?= rel

.PHONY: all clean clean-bin
all: clean-bin framework director
clean: clean-bin

## Godeps begin
.godep: Godeps/Godeps.json
	godep restore
	touch .godep
## Godeps end

### Framework begin
.PHONY: framework clean_framework
# Depends on artifacts, because it depends on scheduler which depends on artifacts
.bin.framework_linux_amd64:
	go build -o bin/framework_linux_amd64 -tags=$(TAGS) ./framework/
	$(shell touch .bin.framework_linux_amd64)
framework: .godep schroot cepm artifacts executor riak_explorer scheduler .bin.framework_linux_amd64
clean-bin: clean_framework
clean_framework:
	-rm -f .bin.framework_linux_amd64 bin/framework_linux_amd64
### Framework end

### Scheduler begin
.PHONY: scheduler clean_scheduler
.scheduler.bindata_generated: .scheduler.data.executor_linux_amd64 .process_manager.bindata_generated
	go generate -tags=$(TAGS) ./scheduler
	$(shell touch .scheduler.bindata_generated)
scheduler: .scheduler.bindata_generated
clean-bin: clean_scheduler
clean_scheduler:
	-rm -rf .scheduler.bindata_generated scheduler/bindata_generated.go
### Scheduler end

### Executor begin
.PHONY: executor clean_executor .scheduler.data.executor_linux_amd64
executor: .scheduler.data.executor_linux_amd64
.executor.bindata_generated: executor/data/advanced.config executor/data/riak.conf
	go generate -tags=$(TAGS) ./executor/...
	$(shell touch .executor.bindata_generated)
.scheduler.data.executor_linux_amd64: cepm .executor.bindata_generated .process_manager.bindata_generated
	go build -o scheduler/data/executor_linux_amd64 -tags=$(TAGS) ./executor/
	$(shell touch .scheduler.data.executor_linux_amd64)
clean-bin: clean_executor
clean_executor:
	-rm -f .executor.bindata_generated executor/bindata_generated.go
	-rm -f .scheduler.data.executor_linux_amd64 scheduler/data/executor_linux_amd64
### Executor end

### Artifact begin
.PHONY: artifacts clean_artifacts
artifacts:
	cd artifacts/data && $(MAKE)
	go generate -tags=$(TAGS) ./artifacts
clean: clean_artifacts
clean_artifacts:
	cd artifacts/data && $(MAKE) clean
### Artifact end

### Tools begin
.PHONY: tools clean_tools .bin.tools_linux_amd64
.bin.tools_linux_amd64:
	go build -o bin/tools_linux_amd64 -tags=$(TAGS) ./tools/
	$(shell touch .bin.tools_linux_amd64)
tools: .bin.tools_linux_amd64
all: tools
clean-bin: clean_tools
clean_tools:
	-rm -rf .bin.tools_linux_amd64 bin/tools_linux_amd64
### Tools end

### Director begin
.PHONY: director clean_director
.director.bindata_generated: .process_manager.bindata_generated
	go generate -tags=$(TAGS) ./director
	$(shell touch .director.bindata_generated)
director: .director.bindata_generated
	go build -o bin/director_linux_amd64 -tags=$(TAGS) ./director/
clean-bin: clean_director
clean_director:
	-rm -rf .director.bindata_generated director/bindata_generated.go
### Scheduler end

### Schroot begin
.PHONY: schroot clean_schroot
schroot:
	cd process_manager/schroot/data && $(MAKE)
clean-bin: clean_schroot
clean_schroot:
	cd process_manager/schroot/data && $(MAKE) clean
### Schroot end

### Process Manager begin
.PHONY: .process_manager.bindata_generated
.process_manager.bindata_generated:
	go generate -tags=$(TAGS) ./process_manager/...
	$(shell touch .process_manager.bindata_generated)
clean-bin: clean_process_manager
clean_process_manager:
	rm -rf .process_manager.bindata_generated process_manager/bindata_generated.go
### Process Manager end

### CEPMd begin
.PHONY: cepm clean_cepmd erl_dist
erl_dist:
	cd erlang_dist && $(MAKE)
.cepmd.cepm.bindata_generated: erl_dist
	go generate -tags=$(TAGS) ./cepmd/cepm
	$(shell touch .cepmd.cepm.bindata_generated)
cepm: .cepmd.cepm.bindata_generated
clean-bin: clean_cepmd
clean_cepmd:
	-rm -f .cepmd.cepm.bindata_generated cepmd/cepm/bindata_generated.go
### CEPMd end

### Riak Explorer begin
.PHONY: riak_explorer clean_riak_explorer
.riak_explorer.bindata_generated: riak_explorer/data/advanced.config riak_explorer/data/riak_explorer.conf
	go generate -tags=$(TAGS) ./riak_explorer/...
	$(shell touch .riak_explorer.bindata_generated)
riak_explorer: artifacts .riak_explorer.bindata_generated
clean-bin: clean_riak_explorer
clean_riak_explorer:
	-rm -f .riak_explorer.bindata_generated riak_explorer/bindata_generated.go
### Riak Explorer end

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
