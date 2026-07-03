NAME      ?= anthropic
NAMESPACE ?= sauterdigital
VERSION   ?= 0.3.0
HOSTNAME  ?= registry.terraform.io
BINARY    := terraform-provider-$(NAME)
OS_ARCH   := $(shell go env GOOS)_$(shell go env GOARCH)

.PHONY: build install test testacc fmt vet tidy docs clean

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	mv $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

test:
	go test ./... -timeout 120s

testacc:
	TF_ACC=1 go test ./internal/provider/... -v -timeout 30m

fmt:
	gofmt -s -w .

vet:
	go vet ./...

tidy:
	go mod tidy

docs:
	tfplugindocs generate

clean:
	rm -f $(BINARY)
