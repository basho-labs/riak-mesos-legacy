.PHONY: all clean

all: ebin/erl_epmd.beam ebin/inet_tcp_dist.beam ebin/net_kernel.beam


ebin/erl_epmd.beam: src/erl_epmd.erl
	mkdir -p ebin
	erlc -o ebin/ src/erl_epmd.erl
ebin/inet_tcp_dist.beam: src/inet_tcp_dist.erl
	mkdir -p ebin
	erlc -o ebin/ src/inet_tcp_dist.erl
ebin/net_kernel.beam: src/net_kernel.erl
	mkdir -p ebin
	erlc -o ebin/ src/net_kernel.erl


clean:
	rm ebin/*
