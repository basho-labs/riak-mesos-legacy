.PHONY: all

all: riak_linux_amd64.tar.gz riak_darwin_amd64.tar.gz riak_explorer_darwin_amd64.tar.gz riak_explorer_linux_amd64.tar.gz executor_darwin_amd64 executor_linux_amd64

build_executor:
	gox -osarch="linux/amd64" -osarch=darwin/amd64 ../../executor/...

executor_darwin_amd64: build_executor
executor_linux_amd64: build_executor

riak_explorer_darwin_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_darwin_amd64.tar.gz

riak_explorer_linux_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_linux_amd64.tar.gz

riak_darwin_amd64.tar.gz:
	curl -O http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/osx/10.8/riak-2.1.1-OSX-x86_64.tar.gz -o riak_darwin_amd64.tar.gz

riak_linux_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_linux_amd64.tar.gz
