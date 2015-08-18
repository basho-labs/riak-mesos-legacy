.PHONY: all clean

all: ebin/erl_epmd.beam ebin/inet_tcp_dist.beam ebin/net_kernel.beam

ebin:
	mkdir -p ebin
ebin/erl_epmd.beam: ebin src/erl_epmd.erl
	erlc -o ebin/ src/erl_epmd.erl
ebin/inet_tcp_dist.beam: ebin src/inet_tcp_dist.erl
	erlc -o ebin/ src/inet_tcp_dist.erl
ebin/net_kernel.beam: ebin src/net_kernel.erl
	erlc -o ebin/ src/net_kernel.erl


clean:
	rm ebin/*
