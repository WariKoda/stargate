GO ?= go
BINARY ?= stargate
BUILD_DIR ?= bin
PACKAGE := ./cmd/stargate

.PHONY: build test vet check coverage install clean

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -trimpath -o $(BUILD_DIR)/$(BINARY) $(PACKAGE)

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: test vet

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

install:
	$(GO) install $(PACKAGE)

clean:
	rm -rf $(BUILD_DIR) coverage.out
