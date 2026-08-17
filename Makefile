PACKAGE = github.com/xymaxim/ypb

GIT_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | tr -d 'v\n')

ifdef GITHUB_REF_NAME
GIT_VERSION := $(GITHUB_REF_NAME)
endif

VERSION_LDFLAGS = -X $(PACKAGE)/internal/version.GitVersion=$(GIT_VERSION)

fmt:
	@golangci-lint fmt

lint: fmt
	@golangci-lint run

test:
	go test ./...

run:
	@go run -ldflags "$(VERSION_LDFLAGS)" -buildvcs=true ./cmd/ypb $(ARGS)

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "$(VERSION_LDFLAGS)" -o build/ypb ./cmd/ypb

release:
	goreleaser release --clean

snapshot:
	goreleaser release --clean --snapshot --skip=publish

.PHONY: docs
docs:
	uv run zensical build
