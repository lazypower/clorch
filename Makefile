VERSION := $(shell cat VERSION)

.PHONY: build install clean test

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/clorch ./cmd/clorch

install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" ./cmd/clorch

test:
	go test ./...

clean:
	rm -rf bin/
