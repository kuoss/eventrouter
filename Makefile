
IMG ?= ghcr.io/kuoss/eventrouter:development

.PHONY: docker
docker:
	docker build -t $(IMG) .

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
