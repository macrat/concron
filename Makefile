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

.PHONY: fulltest
fulltest:
	@echo WARNING: fulltest started. it takes very long time.
	@echo
	go test -v -cover -race -tags=fulltest ./...

.PHONY: fulltest-only
fulltest-only:
	@echo WARNING: fulltest started. it takes very long time.
	@echo
	go test -v -cover -race -tags=fulltest -run=fulltest
