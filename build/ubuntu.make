BASE_DIR = $(shell pwd)
BUILD_DIR ?= $(BASE_DIR)/_build
DEPLOY_BASE ?= riak-tools/riak-mesos
DEPLOY_OS ?= ubuntu

.PHONY: package-riak package-director package-explorer sync-riak sync-director sync-explorer sync deps dirs-exist

package-riak: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_linux_amd64.tar.gz
package-director: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64.tar.gz
package-explorer: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_explorer_linux_amd64.tar.gz

sync-riak:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/
sync-director:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_mesos_director_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/
sync-explorer:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_explorer_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/

sync: sync-riak sync-director sync-explorer

deps: $(BUILD_DIR)/riak-2.1.1.tar.gz
deps: $(BUILD_DIR)/otp_src_R16B02-basho8.tar.gz
deps: $(BUILD_DIR)/riak_explorer/README.md
deps: $(BUILD_DIR)/riak-mesos-director/README.md

$(BUILD_DIR)/riak-2.1.1.tar.gz:
	cd $(BUILD_DIR) && curl -O http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/riak-2.1.1.tar.gz
	cd $(BUILD_DIR) && tar zxvf riak-2.1.1.tar.gz
	cd $(BUILD_DIR) && mv riak-2.1.1 riak
	-cd $(BUILD_DIR) && rm -rf riak/deps/node_package
	cd $(BUILD_DIR) && git clone git@github.com:basho/node_package.git --branch no-epmd
	cd $(BUILD_DIR) && mv node_package riak/deps/node_package
$(BUILD_DIR)/otp_src_R16B02-basho8.tar.gz:
	cd $(BUILD_DIR) && curl -O http://s3.amazonaws.com/downloads.basho.com/erlang/otp_src_R16B02-basho8.tar.gz
	cd $(BUILD_DIR) && tar zxvf otp_src_R16B02-basho8.tar.gz
$(BUILD_DIR)/riak_explorer/README.md:
	cd $(BUILD_DIR) && git clone git@github.com:basho-labs/riak_explorer.git
$(BUILD_DIR)/riak-mesos-director/README.md:
	cd $(BUILD_DIR) && git clone git@github.com:basho-labs/riak-mesos-director.git

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_linux_amd64.tar.gz: dirs-exist
	cd $(BUILD_DIR)/riak/rel && tar -zcvf riak_linux_amd64.tar.gz riak
	mv $(BUILD_DIR)/riak/rel/riak_linux_amd64.tar.gz $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/
$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64.tar.gz: dirs-exist
	cd $(BUILD_DIR)/riak-mesos-director/rel && tar -zcvf riak_mesos_director_linux_amd64.tar.gz riak_mesos_director
	mv $(BUILD_DIR)/riak-mesos-director/rel/riak_mesos_director_linux_amd64.tar.gz $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/
$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_explorer_linux_amd64.tar.gz: dirs-exist
	cd $(BUILD_DIR)/riak_explorer/rel && tar -zcvf riak_explorer_linux_amd64.tar.gz riak_explorer
	mv $(BUILD_DIR)/riak_explorer/rel/riak_explorer_linux_amd64.tar.gz $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/

dirs-exist:
	mkdir -p $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/
