BASE_DIR         = $(shell pwd)
PACKAGE_VERSION ?= 0.1.0
export TAGS     ?= rel

.PHONY: all clean
all: framework

## Godeps begin
.godep: Godeps/Godeps.json
	godep restore
	touch .godep
## Godeps end

### Framework begin
.PHONY: framework clean_framework
# Depends on artifacts, because it depends on scheduler which depends on artifacts
framework: .godep schroot cepm artifacts executor riak_explorer scheduler
	go build -o bin/framework_linux_amd64 -tags=$(TAGS) ./framework/
clean: clean_framework
clean_framework:
	-rm -f bin/framework_linux_amd64
### Framework end

### Scheduler begin
.PHONY: scheduler clean_scheduler
scheduler/bindata_generated.go: scheduler/data/executor_linux_amd64 process_manager/bindata_generated.go
	go generate -tags=$(TAGS) ./scheduler
scheduler: scheduler/bindata_generated.go
clean: clean_scheduler
clean_scheduler:
	-rm -rf scheduler/bindata_generated.go
### Scheduler end

### Executor begin
.PHONY: executor clean_executor scheduler/data/executor_linux_amd64
clean: clean_executor
executor: scheduler/data/executor_linux_amd64
executor/bindata_generated.go: executor/data/advanced.config executor/data/riak.conf
	go generate -tags=$(TAGS) ./executor/...
scheduler/data/executor_linux_amd64: cepm executor/bindata_generated.go process_manager/bindata_generated.go
	go build -o scheduler/data/executor_linux_amd64 -tags=$(TAGS) ./executor/
clean_executor:
	-rm -f executor/bindata_generated.go
	-rm -f scheduler/data/executor_linux_amd64
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
.PHONY: tools clean_tools bin/tools_linux_amd64
bin/tools_linux_amd64:
	go build -o bin/tools_linux_amd64 -tags=$(TAGS) ./tools/
tools: bin/tools_linux_amd64
all: tools
clean_tools:
	-rm -rf bin/tools_linux_amd64
### Tools end

### Schroot begin
.PHONY: schroot clean_schroot
schroot:
	cd process_manager/schroot/data && $(MAKE)
clean_schroot:
	cd process_manager/schroot/data && $(MAKE) clean
### Schroot end

### Process Manager begin
.PHONY: process_manager/bindata_generated.go
process_manager/bindata_generated.go:
	go generate -tags=$(TAGS) ./process_manager/...
clean_process_manager:
	rm -rf process_manager/bindata_generated.go
clean: clean_process_manager
### Process Manager end

### CEPMd begin
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
### CEPMd end

### Riak Explorer begin
.PHONY: riak_explorer clean_riak_explorer
riak_explorer/bindata_generated.go: riak_explorer/data/advanced.config riak_explorer/data/riak_explorer.conf
	go generate -tags=$(TAGS) ./riak_explorer/...
riak_explorer: artifacts riak_explorer/bindata_generated.go
clean_riak_explorer:
	-rm -f riak_explorer/bindata_generated.go
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
