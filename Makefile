PACKAGE = github.com/xymaxim/ypb

GIT_VERSION = `(git describe 2>/dev/null) | tr -d 'v\n'`
VERSION_LDFLAGS = -X $(PACKAGE)/internal/version.GitVersion=$(GIT_VERSION)

fmt:
	@golangci-lint fmt

lint: fmt
	@golangci-lint run

test:
	go test ./...

run:
	@go run -ldflags "$(VERSION_LDFLAGS)" -buildvcs=true ./cmd/ypb $(ARGS)

build:
	go build -ldflags "$(VERSION_LDFLAGS)" -o ypb ./cmd/ypb
