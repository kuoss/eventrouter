PROJECT_NAME := eventrouter
CLUSTER_NAME := kind1
VERSION := latest
IMG := ghcr.io/kuoss/eventrouter:$(VERSION)

# checks
.PHONY: checks
checks: test lint docker-build vulncheck

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --timeout 5m

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: vulncheck
vulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...


#### test on kind ####
.PHONY: kind-create
kind-create:
	kind create cluster --name $(CLUSTER_NAME)

.PHONY: kind-deploy
kind-deploy:
	docker pull $(IMG)
	kind load docker-image $(IMG) --name $(CLUSTER_NAME)
	sed 's|latest|$(VERSION)|g' yaml/eventrouter-with-sidecar.yaml | grep image:
	sed 's|latest|$(VERSION)|g' yaml/eventrouter-with-sidecar.yaml | kubectl apply -f -
	kubectl -n kube-system get pod -l app=eventrouter
	kubectl -n kube-system rollout restart deploy -l app=eventrouter
	kubectl -n kube-system logs -l app=eventrouter -f

.PHONY: kind-delete
kind-delete:
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: kind-eventbit
kind-eventbit:
	docker pull fluent/fluent-bit
	kind load docker-image fluent/fluent-bit --name $(CLUSTER_NAME)
	kubectl apply -f yaml/eventbit.yaml
	kubectl -n kube-system get pod -l app=eventbit
	kubectl -n kube-system rollout restart deploy -l app=eventbit
	kubectl -n kube-system logs -l app=eventbit -f
