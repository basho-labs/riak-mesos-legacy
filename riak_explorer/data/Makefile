BASE_DIR   = $(shell pwd)

### System Architecture
FARC      ?= darwin_amd64

.PHONY: all

all: riak_explorer_$(FARC).tar.gz

riak_explorer_$(FARC).tar.gz:
	curl -O http://riak-tools.s3.amazonaws.com/riak_explorer_$(FARC).tar.gz
