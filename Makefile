# Common flags passed into Go's linker.
GOTEST := go test --race -v

GOOS ?= linux
GOARCH ?= amd64

test:
	${GOTEST} ./pkg/...
