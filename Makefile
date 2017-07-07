GRAFITI_REL_PATH ?= github.com/coreos/grafiti
INSTALL_PATH = $(GRAFITI_REL_PATH)/cmd/grafiti

all: install test

install:
	go install -v $(INSTALL_PATH)

test:
	go test -v $(shell glide novendor)

lint:
	go fmt $(shell glide novendor)

.PHONY: all install test lint
