PACKAGE_VERSION ?= 0.2.0
PROJECT_BASE    ?= riak-mesos
DEPLOY_OS       ?= coreos

.PHONY: all
all: riak-bin.tar.gz trusty.tar.gz riak_mesos_director-bin.tar.gz

riak-bin.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/riak-bin.tar.gz

riak_mesos_director-bin.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/riak_mesos_director-bin.tar.gz

trusty.tar.gz:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/trusty.tar.gz
