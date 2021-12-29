VERSION := $(shell git describe --tags --dirty | sed -e 's/^v//' -e 's/-g.*$$//')
COMMIT := $(shell git rev-parse --short HEAD)
DEFAULT_LISTEN := :8000

concron: *.go
	go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.DefaultListen=${DEFAULT_LISTEN}" -trimpath .

.PHONY: build
build: concron

.PHONY: fmt
fmt:
	gofmt -s -w *.go

.PHONY: fmttest
fmttest:
	! gofmt -s -d -e *.go | grep .

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
