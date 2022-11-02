VERSION ?= $(shell git describe --tags 2> /dev/null || echo v0)
BUILD := $(shell git rev-parse --short HEAD)

# Go related variables.
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin

# Use linker flags to provide version/build settings
LDFLAGS=-ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD)"

# Redirect error output to a file, so we can show it in development mode.
STDERR := /tmp/.exp-stderr.txt
CHANGELOG_TMP := /tmp/.exp-changelog.txt

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

## install: Install missing dependencies. Runs `go get` internally. e.g; make install get=github.com/foo/bar
install: go-get

## run: Run server.
run: go-get go-run

build: go-get go-clean go-build


## compile: Compile the binary.
compile:
	@-touch $(STDERR)
	@-rm $(STDERR) $(GOBIN)/*
	@-$(MAKE) -s go-compile 2> $(STDERR)
	@cat $(STDERR) | sed -e '1s/.*/\nError:\n/'  | sed 's/make\[.*/ /' | sed "/^/s/^/     /" 1>&2

## clean: Clean build files. Runs `go clean` internally.
clean:
	@-$(MAKE) go-clean

## check: Format and lint
check: go-fmt go-lint

go-compile: go-get go-build

go-build:
	@echo "  >  Building service binary..."
	go build $(LDFLAGS) -o rocket-storagemanager main.go

go-run:
	@echo "  >  Running server..."
	@GOBIN=$(GOBIN) GO111MODULE=off go get github.com/cespare/reflex
	@$(GOBIN)/reflex -r '\.go' -s -- /bin/sh -c 'go run $(GOBASE)/main.go'

go-get:
	@echo "  >  Checking if there is any missing dependencies..."
	go get ./...

go-fmt:
	@echo "  >  Running go formatter..."
	gofmt -w -s ./ 1>&2

go-lint:
	@echo "  >  Running go linter..."
	golint ./...

go-install:
	@GOBIN=$(GOBIN) go install $(GOFILES)

go-clean:
	@echo "  >  Cleaning build cache"
	@GOBIN=$(GOBIN) go clean

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run:"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
