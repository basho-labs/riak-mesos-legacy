.PHONY: all build_executor

all: riak_linux_amd64.tar.gz riak_darwin_amd64.tar.gz riak_explorer_darwin_amd64.tar.gz riak_explorer_linux_amd64.tar.gz

riak_explorer_darwin_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_darwin_amd64.tar.gz

riak_explorer_linux_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_linux_amd64.tar.gz

riak_darwin_amd64.tar.gz:
	curl -O http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/osx/10.8/riak-2.1.1-OSX-x86_64.tar.gz -o riak_darwin_amd64.tar.gz

riak_linux_amd64.tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_linux_amd64.tar.gz
