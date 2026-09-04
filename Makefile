# Recipes fail on the first error, on unset variables and on broken pipes.
# Without pipefail, `some-check | tail` reports the exit status of tail.
SHELL       := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

PROJECT_NAME := eventrouter
CLUSTER_NAME := kind1
# The release version, and what a local image gets tagged with. Changing
# VERSION and merging it is what cuts a release - see CONTRIBUTING.md.
VERSION      := $(shell cat VERSION)
IMAGE_TAG    ?= latest
IMG          := ghcr.io/kuoss/$(PROJECT_NAME):$(IMAGE_TAG)
MANIFEST     := tests/eventrouter/eventrouter-with-sidecar.yaml

# Tool versions - keep GOLANGCI_LINT_VERSION in sync with
# .github/workflows/pull-request.yml.
GOLANGCI_LINT_VERSION := v2.13.2
GOVULNCHECK_VERSION   := v1.7.0
ACTIONLINT_VERSION    := v1.7.12

# Tools are pinned and installed under ./bin, so a check never depends on
# whatever happens to be in $$PATH. The version is part of the file name, so
# bumping a version above reinstalls the tool.
LOCALBIN      := $(CURDIR)/bin
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GOVULNCHECK   := $(LOCALBIN)/govulncheck-$(GOVULNCHECK_VERSION)
ACTIONLINT    := $(LOCALBIN)/actionlint-$(ACTIONLINT_VERSION)

GO_TEST_FLAGS := -shuffle=on -covermode=atomic -coverprofile=coverage.out

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Checks

.PHONY: checks
checks: lint lint-workflows test docker-build vulncheck ## Run every check (lint, test, image build, vulnerabilities)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint, including gofmt/goimports formatting
	$(GOLANGCI_LINT) run

.PHONY: lint-workflows
lint-workflows: $(ACTIONLINT) ## Lint the GitHub Actions workflows
	$(ACTIONLINT) .github/workflows/*.yml

.PHONY: fmt
fmt: $(GOLANGCI_LINT) ## Format the code in place
	$(GOLANGCI_LINT) fmt

.PHONY: lint-fix
lint-fix: fmt $(GOLANGCI_LINT) ## Format and apply the fixes linters can make
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: ## Run the tests with coverage
	go test $(GO_TEST_FLAGS) ./...

.PHONY: test-race
test-race: ## Run the tests under the race detector (needs a C toolchain)
	CGO_ENABLED=1 go test -race $(GO_TEST_FLAGS) ./...

.PHONY: cover
cover: test ## Report coverage per function
	go tool cover -func=coverage.out

.PHONY: vulncheck
vulncheck: $(GOVULNCHECK) ## Report known vulnerabilities reachable from this code
	$(GOVULNCHECK) ./...

.PHONY: docker-build
docker-build: ## Build the container image
	docker build --build-arg VERSION=$(VERSION) -t $(IMG) .

.PHONY: sample
sample: ## Regenerate tests/sample/pod-log.ndjson
	go run ./tests/sample/gen > tests/sample/pod-log.ndjson

.PHONY: clean
clean: ## Remove build artifacts and downloaded tools
	rm -rf $(LOCALBIN) coverage.out

##@ Tools

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(GOLANGCI_LINT): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(LOCALBIN)/golangci-lint $@

$(GOVULNCHECK): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	mv $(LOCALBIN)/govulncheck $@

$(ACTIONLINT): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)
	mv $(LOCALBIN)/actionlint $@

##@ Test on kind

.PHONY: kind-create
kind-create: ## Create the kind cluster
	kind create cluster --name $(CLUSTER_NAME)

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: kind-deploy
kind-deploy: ## Load the image into kind and (re)deploy eventrouter
	docker pull $(IMG)
	kind load docker-image $(IMG) --name $(CLUSTER_NAME)
	sed 's|:latest|:$(IMAGE_TAG)|g' $(MANIFEST) | grep image:
	sed 's|:latest|:$(IMAGE_TAG)|g' $(MANIFEST) | kubectl apply -f -
	kubectl -n kube-system get pod -l app=eventrouter
	kubectl -n kube-system rollout restart deploy -l app=eventrouter
	kubectl -n kube-system logs -l app=eventrouter -f
