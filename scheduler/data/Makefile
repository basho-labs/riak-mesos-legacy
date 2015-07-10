BASE_DIR            = $(shell pwd)

### System Architecture
FRAMEWORK_ARCH     ?= darwin_amd64
EXECUTOR_ARCH      ?= linux_amd64

.PHONY: all testing

all: riak_$(EXECUTOR_ARCH).tar.gz riak_explorer_$(FRAMEWORK_ARCH).tar.gz

riak_explorer_$(FRAMEWORK_ARCH).tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_$(FRAMEWORK_ARCH).tar.gz

riak_darwin_amd64.tar.gz:
	curl -O http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/osx/10.8/riak-2.1.1-OSX-x86_64.tar.gz -o $(BASE_DIR)/riak_darwin_amd64.tar.gz

riak_linux_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_linux_amd64.tar.gz
