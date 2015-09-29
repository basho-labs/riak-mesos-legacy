PACKAGE_VERSION ?= 0.1.1
PROJECT_BASE    ?= riak-mesos
DEPLOY_OS       ?= coreos

.PHONY: all
all: riak_explorer-bin.tar.gz riak-2.1.1-bin.tar.gz trusty.tar.gz riak_mesos_director-bin.tar.gz
ubuntu: riak_explorer-bin.tar.gz riak-2.1.1-bin.tar.gz riak_mesos_director-bin.tar.gz

riak_explorer-bin.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/riak_explorer-bin.tar.gz

riak-2.1.1-bin.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/riak-2.1.1-bin.tar.gz

riak_mesos_director-bin.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/riak_mesos_director-bin.tar.gz

trusty.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/trusty.tar.gz
