VERSION ?= $(shell git describe --tags --dirty | grep -o '[0-9].*')
COMMIT ?= $(shell git rev-parse --short HEAD)
DEFAULT_LISTEN ?= :8000

concron: *.go
	go build -ldflags="-s -w -X main.version=$(or ${VERSION},HEAD) -X main.commit=${COMMIT} -X main.DefaultListen=${DEFAULT_LISTEN}" -trimpath .

.PHONY: fmt
fmt:
	gofmt -s -w *.go

.PHONY: test
test:
	go test -cover -race ./...
