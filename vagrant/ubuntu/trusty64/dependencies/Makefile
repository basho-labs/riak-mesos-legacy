BASE_DIR         = $(shell pwd)
BUILD_DIR       ?= _build

.PHONY: all clean clone package deploy deploy-riak deploy-explorer deploy-director

all: otp_src_R16B02-basho8.tar.gz riak-2.1.1.tar.gz clean clone package

clean:
	# -vagrant destroy
	-rm -rf riak_explorer
	-rm -rf riak
	-rm -rf riak-mesos-director
	-rm -rf node_package
	-rm -rf OTP_R16B02-basho8
	-rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

riak-2.1.1.tar.gz:
	curl -O http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/riak-2.1.1.tar.gz

otp_src_R16B02-basho8.tar.gz:
	curl -O http://s3.amazonaws.com/downloads.basho.com/erlang/otp_src_R16B02-basho8.tar.gz

clone:
	git clone git@github.com:basho-labs/riak_explorer.git
	git clone git@github.com:basho-labs/riak-mesos-director.git
	tar zxvf riak-2.1.1.tar.gz
	tar zxvf otp_src_R16B02-basho8.tar.gz
	mv riak-2.1.1 riak
	-rm -rf riak/deps/node_package
	git clone git@github.com:basho/node_package.git --branch no-epmd
	cp -R node_package riak/deps/node_package

package:
	#vagrant up
	cd riak/rel && tar -zcvf riak_linux_amd64.tar.gz riak
	mv riak/rel/riak_linux_amd64.tar.gz $(BUILD_DIR)/
	cd riak_explorer/rel && tar -zcvf riak_explorer_linux_amd64.tar.gz riak_explorer
	mv riak_explorer/rel/riak_explorer_linux_amd64.tar.gz $(BUILD_DIR)/
	cd riak-mesos-director/rel && tar -zcvf riak_mesos_director_linux_amd64.tar.gz riak_mesos_director
	mv riak-mesos-director/rel/riak_mesos_director_linux_amd64.tar.gz $(BUILD_DIR)/

# Deployment
deploy-riak:
	cd $(BUILD_DIR) && s3cmd put --acl-public riak_linux_amd64.tar.gz s3://riak-tools/
deploy-explorer:
	cd $(BUILD_DIR) && s3cmd put --acl-public riak_explorer_linux_amd64.tar.gz s3://riak-tools/
deploy-director:
	cd $(BUILD_DIR) && s3cmd put --acl-public riak_mesos_director_linux_amd64.tar.gz s3://riak-tools/
deploy: deploy-riak deploy-explorer deploy-director
