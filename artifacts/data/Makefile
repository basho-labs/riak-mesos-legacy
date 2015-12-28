PREFIX := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ERLC   := $(shell which erlc)
TMP	   := /tmp
RIAK_SOURCE_DIR ?= riak-2.1.1
RIAK_SOURCE_PKG ?= riak-2.1.1.tar.gz
RIAK_SOURCE_DOWNLOAD ?= http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/riak-2.1.1.tar.gz
RIAK_S2_SOURCE_DIR ?= riak-cs-2.1.0
RIAK_S2_SOURCE_PKG ?= riak-cs-2.1.0.tar.gz
RIAK_S2_SOURCE_DOWNLOAD ?= http://s3.amazonaws.com/downloads.basho.com/riak-cs/2.1/2.1.0/riak-cs-2.1.0.tar.gz
STANCHION_SOURCE_DIR ?= stanchion-2.1.0
STANCHION_SOURCE_PKG ?= stanchion-2.1.0.tar.gz
STANCHION_SOURCE_DOWNLOAD ?= http://s3.amazonaws.com/downloads.basho.com/stanchion/2.1/2.1.0/stanchion-2.1.0.tar.gz

.PHONY: all clean
all: riak-bin.tar.gz trusty.tar.gz riak_mesos_director-bin.tar.gz riak-s2-bin.tar.gz stanchion-bin.tar.gz riak-s2-init-bin.tar.gz
native: riak-bin.tar.gz riak_mesos_director-bin.tar.gz riak-s2-bin-native stanchion-bin.tar.gz riak-s2-init-bin.tar.gz

### Riak Explorer begin
riak_explorer:
	git clone https://github.com/basho-labs/riak_explorer.git
	cd riak_explorer && git checkout riak-addon-master && git pull
riak_explorer-bin: riak_explorer OTP_R16B02_basho8-bin
	cd riak_explorer && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	cd riak_explorer && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) riak-addon
	touch riak_explorer-bin
clean: clean_riak_explorer
clean_riak_explorer:
	-rm -rf riak_explorer riak_explorer-bin
### Riak Explorer end

### Riak S2 begin
riak-s2-bin-native: riak-s2-bin.tar.gz riak-bin.tar.gz
$(RIAK_S2_SOURCE_PKG):
	curl -C - -O -L $(RIAK_S2_SOURCE_DOWNLOAD)
riak-s2-bin: $(RIAK_S2_SOURCE_PKG) OTP_R16B02_basho8-bin riak-bin
	tar -xvf $(RIAK_S2_SOURCE_PKG)
	cd $(RIAK_S2_SOURCE_DIR) && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	cp $(RIAK_S2_SOURCE_DIR)/rel/riak-cs/lib/riak_cs-2.1.0/ebin/* $(RIAK_SOURCE_DIR)/rel/riak/lib/basho-patches/
	touch riak-s2-bin
riak-s2-bin.tar.gz: riak-s2-bin
	mkdir -p riak-s2/bin
	ln -snf ../$(RIAK_S2_SOURCE_DIR)/rel/riak-cs riak-s2/riak-s2
	ln -snf ../../../../riak_s2/bin/riak-s2 riak-s2/bin/riak-s2
	tar chzf riak-s2-bin.tar.gz riak-s2 || rm -rf riak-s2-bin.tar.gz
	$(MAKE) clean_riak_tar riak-bin.tar.gz
clean: clean_riak-s2
clean_riak-s2:
	-rm -rf $(RIAK_S2_SOURCE_DIR) $(RIAK_S2_SOURCE_PKG) riak-s2-bin.tar.gz riak-s2-binS riak-s2
### Riak S2 end

### Stanchion begin
$(STANCHION_SOURCE_PKG):
	curl -C - -O -L $(STANCHION_SOURCE_DOWNLOAD)
stanchion-bin: $(STANCHION_SOURCE_PKG) OTP_R16B02_basho8-bin
	tar -xvf $(STANCHION_SOURCE_PKG)
	cd $(STANCHION_SOURCE_DIR) && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch stanchion-bin
stanchion-bin.tar.gz: stanchion-bin
	mkdir -p stanchion/bin
	ln -snf ../$(STANCHION_SOURCE_DIR)/rel/stanchion stanchion/stanchion
	ln -snf ../../../../riak_s2/bin/stanchion stanchion/bin/stanchion
	tar chzf stanchion-bin.tar.gz stanchion || rm -rf stanchion.tar.gz
clean: clean_stanchion
clean_stanchion:
	- rm -rf $(STANCHION_SOURCE_DIR) $(STANCHION_SOURCE_PKG) stanchion.tar.gz stanchion-bin stanchion
### Stanchion end

### Riak S2 Init begin
riak-s2-init-bin.tar.gz: stanchion-bin riak-s2-bin
	mkdir -p riak-s2-init/bin
	ln -snf ../$(STANCHION_SOURCE_DIR)/rel/stanchion riak-s2-init/stanchion
	ln -snf ../$(RIAK_S2_SOURCE_DIR)/rel/riak-cs riak-s2-init/riak-s2
	ln -snf ../../../../riak_s2/bin/riak-s2-init riak-s2-init/bin/riak-s2-init
	touch riak-s2-init-bin
	tar chzf riak-s2-init-bin.tar.gz riak-s2-init || rm -rf riaks2-init-bin.tar.gz
clean: clean_riak-s2-init
clean_riak-s2-init:
	- rm -rf riak-s2-init-bin.tar.gz riak-s2-init-bin riak-s2-init
### Riak S2 Init end

### Riak begin
$(RIAK_SOURCE_PKG):
	curl -C - -O -L $(RIAK_SOURCE_DOWNLOAD)
riak-bin: riak_explorer-bin $(RIAK_SOURCE_PKG) OTP_R16B02_basho8-bin
	tar -xvf $(RIAK_SOURCE_PKG)
	export PATH
	rm -rf $(RIAK_SOURCE_DIR)/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd $(RIAK_SOURCE_DIR)/deps/node_package
	cd $(RIAK_SOURCE_DIR) && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	cp riak_explorer/rel/riak-addon/ebin/* $(RIAK_SOURCE_DIR)/rel/riak/lib/basho-patches/
	-rm -rf $(RIAK_SOURCE_DIR)/rel/riak/priv
	cp -R riak_explorer/rel/riak-addon/priv $(RIAK_SOURCE_DIR)/rel/riak/
	touch riak-bin
riak-bin.tar.gz: riak-bin
	tar -C $(RIAK_SOURCE_DIR)/rel/ -czf riak-bin.tar.gz riak || rm -rf riak-bin.tar.gz
clean: clean_riak clean_riak_tar
clean_riak:
	-rm -rf $(RIAK_SOURCE_PKG) $(RIAK_SOURCE_DIR)
clean_riak_tar:
	-rm -rf riak-bin.tar.gz riak-bin
### Riak end

### Executor begin
$(EXECUTOR_SOURCE_PKG):
	curl -C - -O -L $(EXECUTOR_SOURCE_DOWNLOAD)
executor-bin: riak_explorer-bin $(EXECUTOR_SOURCE_PKG) OTP_R16B02_basho8-bin
	tar -xvf $(EXECUTOR_SOURCE_PKG)
	cd $(EXECUTOR_SOURCE_DIR) && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) deps
	-rm -rf $(EXECUTOR_SOURCE_DIR)/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd $(EXECUTOR_SOURCE_DIR)/deps/node_package
	cd $(EXECUTOR_SOURCE_DIR) && PATH=$(PREFIX)OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch executor-bin
riak_mesos_executor.tar.gz: executor-bin
	-tar -C $(EXECUTOR_SOURCE_DIR)/rel/ -czf riak_mesos_executor.tar.gz riak_mesos_executor || rm -rf riak_mesos_executor.tar.gz
clean: clean_executor clean_executor_tar
clean_executor:
	-rm -rf $(EXECUTOR_SOURCE_PKG) $(EXECUTOR_SOURCE_DIR)
clean_executor_tar:
	-rm -rf riak_mesos_executor.tar.gz executor-bin
### Executor end

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
