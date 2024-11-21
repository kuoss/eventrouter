# Copyright 2017 Heptio Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TARGET = eventrouter
GOTARGET = github.com/kuoss/$(TARGET)
BUILDMNT = /src/
REGISTRY ?= ghcr.io/kuoss
VERSION ?= v0.3.1
IMAGE = $(REGISTRY)/$(BIN)
BUILD_IMAGE ?= golang:1.19
DOCKER ?= docker
DIR := ${CURDIR}

ifneq ($(VERBOSE),)
VERBOSE_FLAG = -v
endif
TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/sinks/...
TEST = go test $(TEST_PKGS) $(TESTARGS)
VET_PKGS ?= $(GOTARGET)/...
VET = go vet $(VET_PKGS)

DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

all: container

container:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 go build'
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):latest -t $(REGISTRY)/$(TARGET):$(VERSION) .

push:
	$(DOCKER) push $(REGISTRY)/$(TARGET):latest
	if git describe --tags --exact-match >/dev/null 2>&1; \
	then \
		$(DOCKER) push $(REGISTRY)/$(TARGET):$(VERSION); \
	fi

_test:
	$(DOCKER_BUILD) '$(TEST)'

vet:
	$(DOCKER_BUILD) '$(VET)'

.PHONY: all local container push

clean:
	rm -f $(TARGET)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):latest
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):$(VERSION)

checks: test build lint

test:
	go test -v ./...

build:
	CGO_ENABLED=0 go build -ldflags=-w -o bin/eventrouter

lint:
	which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --timeout 5m

vulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
