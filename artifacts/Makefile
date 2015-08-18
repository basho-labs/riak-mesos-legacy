.PHONY: all clean

TAGS ?=

all:
	cd data/ && $(MAKE) all
	go generate -tags=$(TAGS) ./

clean:
	cd data/ && $(MAKE) clean
