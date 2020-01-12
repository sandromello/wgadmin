SHORT_NAME ?= wireguard

GIT_TAG ?= $(or ${TRAVIS_TAG},${TRAVIS_TAG},latest)
VERSION ?= ${GIT_TAG}
GITCOMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DOCKER_REGISTRY ?= quay.io/
IMAGE_PREFIX ?= sandromello
IMAGE := ${DOCKER_REGISTRY}${IMAGE_PREFIX}/${SHORT_NAME}:${VERSION}

BINARY_DEST_DIR := dist

GOOS ?= linux
GOARCH ?= amd64

LDFLAGS := "-s -w \
-X github.com/sandromello/wgadmin/pkg/version.version=${VERSION} \
-X github.com/sandromello/wgadmin/pkg/version.gitCommit=${GITCOMMIT} \
-X github.com/sandromello/wgadmin/pkg/version.buildDate=${DATE}"

GOTEST := go test --race -v

test:
	${GOTEST} ./pkg/...

publish:
	./scripts/run-goreleaser.sh

build:
	mkdir -p ${BINARY_DEST_DIR}
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} -o dist/wgadmin cmd/main.go
