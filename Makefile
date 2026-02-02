fmt:
	golangci-lint fmt

lint:
	golangci-lint run

test:
	go test ./...
