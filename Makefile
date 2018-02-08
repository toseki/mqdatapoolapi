.PHONY: build clean test package serve run-compose-test
PKGS := $(shell go list ./... | grep -v /vendor/)
VERSION := $(shell git describe --always)
#VERSION := $(shell date "+%Y-%m-%d:%H:%M:%S")
GOOS ?= linux
#GOOS ?= darwin
GOARCH ?= amd64

build:
	@echo "Compiling source for $(GOOS) $(GOARCH)"
	@mkdir -p build
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.version=$(VERSION)" -o build/mqdatapoolapi$(BINEXT) mqdatapoolapi.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build

build-osx:
	@mkdir -p build
	@echo "Compiling source for darwin amd64"
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-w -X main.version=$(VERSION)" -o build/mqdatapoolapi_darwin$(BINEXT) mqdatapoolapi.go
