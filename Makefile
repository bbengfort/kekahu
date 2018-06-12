# Shell to use with Make
SHELL := /bin/bash

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_PATH=_build/kekahu

# Export targets not associated with files.
.PHONY: build test clean deps protobuf

all: deps test build

build:
	$(GOBUILD) -o $(BINARY_PATH) cmd/kekahu/main.go

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	@rm -rf _build

deps:
	@echo "fetching dependencies with go get"
	@$(GOGET) ./...

# Compile protocol buffers
protobuf:
	@echo "compiling protocol buffers"
	@protoc -I ping/ ping/*.proto --go_out=plugins=grpc:ping/
