PROJ=grafiti
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)
INSTALL_PATH=$(REPO_PATH)/cmd/grafiti
export PATH := $(PWD)/bin:$(PATH)

VERSION ?= $(shell ./scripts/git-version)

DOCKER_REPO=quay.io/coreos/grafiti
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$(	shell mkdir -p bin	)
$(	shell mkdir -p _output/images	)
$(	shell mkdir -p _output/bin	)

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

build: bin/grafiti

bin/grafiti:
	@go install -v -ldflags $(LD_FLAGS) $(INSTALL_PATH)

install:
	@GOBIN=$(GOPATH)/bin go install -v -ldflags $(LD_FLAGS) $(INSTALL_PATH)

.PHONY: release-binary
release-binary:
	@go build -o /go/bin/grafiti -v -ldflags $(LD_FLAGS) $(INSTALL_PATH)

.PHONY: revendor
revendor:
	@glide up -v
	@glide-vc --use-lock-file --no-tests --only-code

test:
	@go test -v -i $(shell go list ./... | grep -v '/vendor/')
	@go test -v $(shell go list ./... | grep -v '/vendor/')

testrace:
	@go test -v -i --race $(shell go list ./... | grep -v '/vendor/')
	@go test -v --race $(shell go list ./... | grep -v '/vendor/')

vet:
	@go vet $(shell go list ./... | grep -v '/vendor/')

fmt:
	@go fmt $(shell go list ./... | grep -v '/vendor/')

lint:
	@for package in $(shell go list ./... | grep -v '/vendor/' | grep -v '/api' | grep -v '/server/internal'); do \
      golint -set_exit_status $$package $$i || exit 1; \
	done

_output/bin/grafiti:
	@./scripts/docker-build
	@sudo chown $(user):$(group) _output/bin/grafiti

.PHONY: docker-image
docker-image: clean-release _output/bin/grafiti
	@sudo docker build -t $(DOCKER_IMAGE) .

clean: clean-release
	@rm -rf bin/

.PHONY: clean-release
clean-release:
	@rm -rf _output/

testall: testrace vet fmt lint

FORCE:

.PHONY: build install test testrace vet fmt lint testall
