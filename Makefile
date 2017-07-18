GRAFITI_REL_PATH ?= github.com/coreos/grafiti
INSTALL_PATH = $(GRAFITI_REL_PATH)/cmd/grafiti

all: build test

build:
	go build -a -o grafiti $(INSTALL_PATH)

install:
	go install $(INSTALL_PATH)

test:
	go test -v $(shell glide novendor)

lint:
	go fmt $(shell glide novendor)

clean:
	go clean

.PHONY: all install build test lint clean
