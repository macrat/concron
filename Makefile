VERSION = $(shell git describe --tags --dirty | grep -o '[0-9].*')
COMMIT = $(shell git rev-parse --short $(shell git describe))

concron: *.go
	go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -trimpath .

.PHONY: fmt
fmt:
	gofmt -s -w *.go

.PHONY: test
test:
	go test -cover -race ./...
