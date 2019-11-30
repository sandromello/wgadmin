SHORT_NAME ?= wireguard

GIT_TAG ?= $(or ${TRAVIS_TAG},${TRAVIS_TAG},latest)
VERSION ?= ${GIT_TAG}
GITCOMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DOCKER_REGISTRY ?= quay.io/
IMAGE_PREFIX ?= sandromello
IMAGE := ${DOCKER_REGISTRY}${IMAGE_PREFIX}/${SHORT_NAME}:${VERSION}

BINARY_DEST_DIR := rootfs/usr/local/bin

GOOS ?= linux
GOARCH ?= amd64
# https://launchpad.net/ubuntu/+source/linux-gcp
# get name from command `uname -r`
KERNEL_RELEASE ?= 4.15.0-1048-gcp

LDFLAGS := "-s -w \
-X github.com/sandromello/wgadmin/pkg/version.version=${VERSION} \
-X github.com/sandromello/wgadmin/pkg/version.gitCommit=${GITCOMMIT} \
-X github.com/sandromello/wgadmin/pkg/version.buildDate=${DATE}"

GOTEST := go test --race -v

test:
	${GOTEST} ./pkg/...

build:
	mkdir -p ${BINARY_DEST_DIR}
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} -o rootfs/usr/local/bin/wgadmin cmd/main.go

docker-build:
	docker build --build-arg=KERNEL_RELEASE=${KERNEL_RELEASE} -f rootfs/Dockerfile --rm -t ${IMAGE} rootfs/

docker-login:
	docker login quay.io -u="${DOCKER_USERNAME}" -p="${DOCKER_PASSWORD}"

docker-push: docker-login
	docker push ${IMAGE}

publish: build docker-build docker-push
