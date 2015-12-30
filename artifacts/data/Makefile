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
all: riak_mesos_executor.tar.gz
riak: riak-bin.tar.gz
riak-s2: riak-s2-bin.tar.gz stanchion-bin.tar.gz riak-s2-init-bin.tar.gz

### Executor begin
riak-mesos-executor:
	git clone https://github.com/basho-labs/riak-mesos-executor.git
	cd riak-mesos-executor && git checkout mtc-dev && git pull
executor-bin: riak-mesos-executor
	cd riak-mesos-executor && $(MAKE) deps
	-rm -rf riak-mesos-executor/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd riak-mesos-executor/deps/node_package
	cd riak-mesos-executor && $(MAKE) rel
	touch executor-bin
riak_mesos_executor.tar.gz: executor-bin
	-tar -C riak-mesos-executor/rel/ -czf riak_mesos_executor.tar.gz riak_mesos_executor || rm -rf riak_mesos_executor.tar.gz
clean: clean_executor
clean_executor:
	-rm -rf riak-mesos-executor
	-rm -rf riak_mesos_executor.tar.gz executor-bin
### Executor end

### Riak Explorer begin
riak_explorer:
	git clone https://github.com/basho-labs/riak_explorer.git
	cd riak_explorer && git checkout riak-addon-master && git pull
riak_explorer-bin: riak_explorer
	cd riak_explorer && $(MAKE) rel
	cd riak_explorer && $(MAKE) riak-addon
	touch riak_explorer-bin
clean: clean_riak_explorer
clean_riak_explorer:
	-rm -rf riak_explorer riak_explorer-bin
### Riak Explorer end

### Riak begin
$(RIAK_SOURCE_PKG):
	curl -C - -O -L $(RIAK_SOURCE_DOWNLOAD)
riak-bin: riak_explorer-bin $(RIAK_SOURCE_PKG)
	tar -xvf $(RIAK_SOURCE_PKG)
	export PATH
	rm -rf $(RIAK_SOURCE_DIR)/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd $(RIAK_SOURCE_DIR)/deps/node_package
	cd $(RIAK_SOURCE_DIR) && $(MAKE) rel
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

### Riak S2 begin
riak-s2-bin-native: riak-s2-bin.tar.gz riak-bin.tar.gz
$(RIAK_S2_SOURCE_PKG):
	curl -C - -O -L $(RIAK_S2_SOURCE_DOWNLOAD)
riak-s2-bin: $(RIAK_S2_SOURCE_PKG) riak-bin
	tar -xvf $(RIAK_S2_SOURCE_PKG)
	cd $(RIAK_S2_SOURCE_DIR) && $(MAKE) rel
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
stanchion-bin: $(STANCHION_SOURCE_PKG)
	tar -xvf $(STANCHION_SOURCE_PKG)
	cd $(STANCHION_SOURCE_DIR) && $(MAKE) rel
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
