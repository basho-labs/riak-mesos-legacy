PACKAGE_VERSION ?= 0.1.1
PROJECT_BASE    ?= riak-mesos
DEPLOY_BASE     ?= riak-tools/$PROJECT_BASE
DEPLOY_OS       ?= coreos

.PHONY: all
all: plain_chroot super_chroot

plain_chroot:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/plain_chroot
	chmod 755 plain_chroot
super_chroot:
	curl -C - -O -L http://riak-tools.s3.amazonaws.com/$(PROJECT_BASE)/$(DEPLOY_OS)/artifacts/$(PACKAGE_VERSION)/super_chroot
	chmod 755 super_chroot
