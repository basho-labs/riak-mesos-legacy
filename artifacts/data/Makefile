PREFIX := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ERLC   := $(shell which erlc)
TMP	   := /tmp

.PHONY: all clean
all: riak_explorer-bin.tar.gz riak-2.1.1-bin.tar.gz trusty.tar.gz riak_mesos_director-bin.tar.gz
ubuntu: riak_explorer-bin.tar.gz riak-2.1.1-bin.tar.gz riak_mesos_director-bin.tar.gz

### Riak Explorer begin
riak_explorer:
	git clone --depth=1 https://github.com/basho-labs/riak_explorer.git
riak_explorer-bin: riak_explorer OTP_R16B02_basho8-bin
	cd riak_explorer && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel-backend
	touch riak_explorer-bin
riak_explorer-bin.tar.gz: riak_explorer-bin
	tar -C riak_explorer/rel -czf riak_explorer-bin.tar.gz riak_explorer || rm -rf riak_explorer-bin.tar.gz
clean: clean_riak_explorer
clean_riak_explorer:
	-rm -rf riak_explorer-bin.tar.gz riak_explorer riak_explorer-bin
### Riak Explorer end

### Riak begin
riak-2.1.1.tar.gz:
	curl -C - -O -L http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/riak-2.1.1.tar.gz
riak-2.1.1-bin: riak-2.1.1.tar.gz OTP_R16B02_basho8-bin
	tar -xvf riak-2.1.1.tar.gz
	export PATH
	rm -rf riak-2.1.1/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd riak-2.1.1/deps/node_package
	cd riak-2.1.1 && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch riak-2.1.1-bin
riak-2.1.1-bin.tar.gz: riak-2.1.1-bin
	tar -C riak-2.1.1/rel/ -czf riak-2.1.1-bin.tar.gz riak || rm -rf riak-2.1.1-bin.tar.gz
clean: clean_riak
clean_riak:
	-rm -rf riak-2.1.1.tar.gz riak-2.1.1 riak-2.1.1-bin.tar.gz riak-2.1.1-bin
### Riak end

### Director begin
riak_mesos_director:
	git clone --depth=1 https://github.com/basho-labs/riak-mesos-director.git riak_mesos_director
riak_mesos_director-bin: riak_mesos_director OTP_R16B02_basho8-bin
	rm -rf riak_mesos_director/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd riak_mesos_director/deps/node_package
	cd riak_mesos_director && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch riak_mesos_director-bin
riak_mesos_director-bin.tar.gz: riak_mesos_director-bin
	mv riak_mesos_director/rel/riak_mesos_director riak_mesos_director/rel/director
	tar -C riak_mesos_director/rel -czf riak_mesos_director-bin.tar.gz director || rm -rf riak_mesos_director-bin.tar.gz
clean: clean_riak_mesos_director
clean_riak_mesos_director:
	-rm -rf riak_mesos_director-bin.tar.gz riak_mesos_director riak_mesos_director-bin
### Director end

### Erlang begin
ifneq ($(ERLC),"")
ifeq ("$(wildcard OTP_R16B02_basho8-bin)","")
$(shell touch OTP_R16B02_basho8-bin)
endif
endif
OTP_R16B02_basho8-bin:
	curl -C - -O http://s3.amazonaws.com/downloads.basho.com/erlang/otp_src_R16B02-basho8.tar.gz || rm otp_src_R16B02-basho8.tar.gz
	tar -xvf otp_src_R16B02-basho8.tar.gz
	cd OTP_R16B02_basho8 && ./otp_build autoconf && ./configure --prefix=$(PREFIX)/OTP_R16B02_basho8-bin && $(MAKE) && $(MAKE) install
clean: clean_erlang
clean_erlang:
	-rm -rf otp_src_R16B02-basho8.tar.gz OTP_R16B02_basho8-bin
### Erlang end

### schroot, debootstrap begin
trusty.tar.gz:
	cd $(TMP) && sudo debootstrap trusty trusty_root/
	cd $(TMP) && sudo chroot trusty_root apt-get install -y busybox-static strace curl wget openssl
	cd $(TMP) && sudo chroot trusty_root apt-get clean
	cd $(TMP) && sudo chown -R $(shell id -un): trusty_root/
	cd $(TMP) && rm -rf trusty_root/dev trusty_root/proc trusty_root/sys
	cd $(TMP) && mkdir -p trusty_root/dev trusty_root/proc trusty_root/sys
	cd $(TMP) && tar -C trusty_root -czf trusty.tar.gz . || rm -rf trusty.tar.gz
	cd $(TMP) && rm -rf trusty_root
	mv $(TMP)/trusty.tar.gz ./
clean: clean_schroot
clean_schroot:
	-rm -rf trusty.tar.gz trusty_root
	-cd $(TMP) && rm -rf trusty.tar.gz trusty_root
### schroot, debootstrap end
