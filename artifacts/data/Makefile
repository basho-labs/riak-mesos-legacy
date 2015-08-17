.PHONY: all clean
PREFIX := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: riak-2.1.1-bin.tar.gz trusty.tar.gz riak_explorer-bin.tar.gz

trusty.tar.gz:
	mkdir -p trusty_root
	fakechroot fakeroot /usr/sbin/debootstrap trusty trusty_root/
	fakechroot fakeroot chroot trusty_root apt-get install -y busybox-static strace curl wget
	fakechroot fakeroot chroot trusty_root apt-get clean
	rm -rf trusty_root/dev trusty_root/proc trusty_root/sys
	mkdir  trusty_root/dev trusty_root/proc trusty_root/sys trusty_root/parent_root
	ln -sf /parent_root/etc/resolv.conf trusty_root/etc/resolv.conf
	ln -sf /parent_root/etc/hosts trusty_root/etc/hosts
	tar -C trusty_root -czf trusty.tar.gz . || rm -rf trusty.tar.gz
	rm -rf trusty_root

otp_src_R16B02-basho8.tar.gz:
	curl -C - -O http://s3.amazonaws.com/downloads.basho.com/erlang/otp_src_R16B02-basho8.tar.gz || rm otp_src_R16B02-basho8.tar.gz

OTP_R16B02_basho8-bin: otp_src_R16B02-basho8.tar.gz
	tar -xvf otp_src_R16B02-basho8.tar.gz
	cd OTP_R16B02_basho8 && ./otp_build autoconf
	cd OTP_R16B02_basho8 && ./configure --prefix=$(PREFIX)/OTP_R16B02_basho8-bin && $(MAKE) && $(MAKE) install

riak_explorer:
	GIT_SSH=./ssh.sh git clone --depth=1 git@github.com:basho-labs/riak_explorer.git  

riak_explorer/rel/riak_explorer: riak_explorer OTP_R16B02_basho8-bin
	cd riak_explorer && PATH=$(PREFIX)/OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel

riak_explorer-bin.tar.gz: riak_explorer/rel/riak_explorer
	tar -C riak_explorer/rel -czf riak_explorer-bin.tar.gz riak_explorer || rm -rf riak_explorer-bin.tar.gz 

riak-2.1.1.tar.gz:
	curl -C - -O -L https://github.com/basho/riak/archive/riak-2.1.1.tar.gz

riak-riak-2.1.1/rel/riak: riak-2.1.1.tar.gz OTP_R16B02_basho8-bin
	tar -xvf riak-2.1.1.tar.gz
	export PATH
	cd riak-riak-2.1.1 && PATH=$(PREFIX)/OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch riak-riak-2.1.1

riak-2.1.1-bin.tar.gz: riak-riak-2.1.1/rel/riak
	tar -C riak-riak-2.1.1/rel/ -czf riak-2.1.1-bin.tar.gz riak || rm -rf riak-2.1.1-bin.tar.gz


clean:
	rm -rf trusty.tar.gz trusty_root OTP_R16B02_basho8-bin riak_explorer riak-riak-2.1.1 riak-2.1.1-bin.tar.gz riak_explorer-bin.tar.gz

