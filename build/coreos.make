BASE_DIR = $(shell pwd)
SCHEDULER_DIR = $(BASE_DIR)/../scheduler/data
EXPLORER_DIR = $(BASE_DIR)/../riak_explorer/data
DIRECTOR_DIR = $(BASE_DIR)/../director/data
BUILD_DIR ?= $(BASE_DIR)/_build
BIN_DIR ?= $(BASE_DIR)/../bin
DEPLOY_BASE ?= riak-tools/riak-mesos
DOWNLOAD_BASE ?= riak-mesos
DEPLOY_OS ?= coreos
PACKAGE_VERSION ?= 0.1.0

.PHONY: clean package-framework package-director package-dcos package-repo sync-framework sync-director sync-dcos sync-repo sync-deps sync deps dirs-exist

clean:
	-rm $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz
	-rm $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64_$(PACKAGE_VERSION).tar.gz
	-rm $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-riak-$(PACKAGE_VERSION).tar.gz
	-rm $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-$(PACKAGE_VERSION).zip

package-framework: deps $(BIN_DIR)/framework_linux_amd64
package-framework: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz
package-director: deps $(BIN_DIR)/director_linux_amd64
package-director: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64_$(PACKAGE_VERSION).tar.gz
package-dcos: $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-riak-$(PACKAGE_VERSION).tar.gz
package-repo: $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-$(PACKAGE_VERSION).zip

sync-framework-test:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/test2/
sync-framework:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/
sync-director:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_mesos_director_linux_amd64_$(PACKAGE_VERSION).tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/
sync-dcos:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/ && \
		s3cmd put --acl-public dcos-riak-$(PACKAGE_VERSION).tar.gz s3://$(DEPLOY_BASE)/
sync-repo:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/ && \
		s3cmd put --acl-public dcos-repo-$(PACKAGE_VERSION).zip s3://$(DEPLOY_BASE)/

sync-deps:
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public ubuntu_chroot_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_explorer_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		s3cmd put --acl-public riak_mesos_director_linux_amd64.tar.gz s3://$(DEPLOY_BASE)/$(DEPLOY_OS)/

sync: sync-framework sync-director sync-dcos sync-repo sync-deps

deps: $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_explorer_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_mesos_director_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ubuntu_chroot_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_explorer_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_linux_amd64.tar.gz
deps: $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_explorer_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && rm -rf riak_explorer
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && \
		curl -O http://riak-tools.s3.amazonaws.com/$(DOWNLOAD_BASE)/ubuntu/riak_explorer_linux_amd64.tar.gz && \
		tar -zxvf riak_explorer_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && rm -rf riak
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && \
		curl -O http://riak-tools.s3.amazonaws.com/$(DOWNLOAD_BASE)/ubuntu/riak_linux_amd64.tar.gz && \
		tar -zxvf riak_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_mesos_director_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && rm -rf riak_mesos_director
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/ && \
		curl -O http://riak-tools.s3.amazonaws.com/$(DOWNLOAD_BASE)/ubuntu/riak_mesos_director_linux_amd64.tar.gz && \
		tar -zxvf riak_mesos_director_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ubuntu_chroot_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf ubuntu_chroot
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		curl -O http://riak-tools.s3.amazonaws.com/$(DOWNLOAD_BASE)/$(DEPLOY_OS)/ubuntu_chroot_linux_amd64.tar.gz && \
		tar -zxvf ubuntu_chroot_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_explorer_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf rex_root
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		cp -R ubuntu_chroot rex_root && \
		cp -R $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_explorer rex_root/ && \
		tar -zcvf riak_explorer_linux_amd64.tar.gz rex_root

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf riak_root
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		cp -R ubuntu_chroot riak_root && \
		cp -R $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak riak_root/ && \
		tar -zcvf riak_linux_amd64.tar.gz riak_root

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64.tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf director_root
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		cp -R ubuntu_chroot director_root && \
		cp -R $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/riak_mesos_director director_root/ && \
		tar -zcvf riak_mesos_director_linux_amd64.tar.gz director_root

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz: dirs-exist
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf riak_mesos_framework
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		mkdir riak_mesos_framework && \
		cp $(BIN_DIR)/framework_linux_amd64 riak_mesos_framework/ && \
		echo "Thank you for downloading Riak Mesos Framework. Please visit https://github.com/basho-labs/riak-mesos for usage information." > INSTALL.txt && \
		tar -zcvf riak_mesos_linux_amd64_$(PACKAGE_VERSION).tar.gz riak_mesos_framework

$(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64_$(PACKAGE_VERSION).tar.gz: dirs-exist $(BIN_DIR)/director_linux_amd64
	-cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && rm -rf riak_mesos_director
	cd $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/ && \
		mkdir riak_mesos_director && \
		cp $(BIN_DIR)/director_linux_amd64 riak_mesos_director/ && \
		echo "Thank you for downloading Riak Mesos Framework. Please visit https://github.com/basho-labs/riak-mesos for usage information." > INSTALL.txt && \
		tar -zcvf riak_mesos_director_linux_amd64_$(PACKAGE_VERSION).tar.gz riak_mesos_director

$(SCHEDULER_DIR)/riak_linux_amd64.tar.gz:
	cp $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_linux_amd64.tar.gz $(SCHEDULER_DIR)/riak_linux_amd64.tar.gz

$(EXPLORER_DIR)/riak_explorer_linux_amd64.tar.gz:
	cp $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_explorer_linux_amd64.tar.gz $(EXPLORER_DIR)/riak_explorer_linux_amd64.tar.gz

$(DIRECTOR_DIR)/riak_mesos_director_linux_amd64.tar.gz:
	cp $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/riak_mesos_director_linux_amd64.tar.gz $(DIRECTOR_DIR)/riak_mesos_director_linux_amd64.tar.gz

$(BUILD_DIR)/$(DEPLOY_BASE)/dcos-riak-$(PACKAGE_VERSION).tar.gz: dirs-exist
	-rm -rf $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-riak-*
	cp -R $(BASE_DIR)/../dcos/dcos-riak $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-riak-$(PACKAGE_VERSION)
	cd $(BUILD_DIR)/$(DEPLOY_BASE) && tar -zcvf dcos-riak-$(PACKAGE_VERSION).tar.gz dcos-riak-$(PACKAGE_VERSION)
$(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-$(PACKAGE_VERSION).zip: dirs-exist
	-rm -rf $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-*
	git clone https://github.com/mesosphere/universe.git $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-$(PACKAGE_VERSION)
	cp -R $(BASE_DIR)/../dcos/repo/* $(BUILD_DIR)/$(DEPLOY_BASE)/dcos-repo-$(PACKAGE_VERSION)/repo/
	cd $(BUILD_DIR)/$(DEPLOY_BASE) && zip -r dcos-repo-$(PACKAGE_VERSION).zip dcos-repo-$(PACKAGE_VERSION)

$(BIN_DIR)/framework_linux_amd64: $(SCHEDULER_DIR)/riak_linux_amd64.tar.gz
$(BIN_DIR)/framework_linux_amd64: $(EXPLORER_DIR)/riak_explorer_linux_amd64.tar.gz
$(BIN_DIR)/framework_linux_amd64: export FARC = linux_amd64
$(BIN_DIR)/framework_linux_amd64: export FGARC = linux/amd64
$(BIN_DIR)/framework_linux_amd64:
	cd ../ && make rel

$(BIN_DIR)/director_linux_amd64: $(DIRECTOR_DIR)/riak_mesos_director_linux_amd64.tar.gz
$(BIN_DIR)/director_linux_amd64: export FARC = linux_amd64
$(BIN_DIR)/director_linux_amd64: export FGARC = linux/amd64
$(BIN_DIR)/director_linux_amd64:
	cd ../ && make rel-director

dirs-exist:
	mkdir -p $(BUILD_DIR)/$(DEPLOY_BASE)/ubuntu/
	mkdir -p $(BUILD_DIR)/$(DEPLOY_BASE)/$(DEPLOY_OS)/
