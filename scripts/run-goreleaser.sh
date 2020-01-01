#!/bin/sh
set -e

if [ -f "/usr/local/bin/goreleaser" ]; then
    "/usr/local/bin/goreleaser" "$@"
	exit $?
fi

VERSION=v0.123.3
CHECKSUM=cad997014e5c6a462488757087db4145c2ae7d7d73a29cb62bbfd41f18ccea30
TAR_FILE="/tmp/goreleaser_$(uname -s)_$(uname -m).tar.gz"
RELEASES_URL="https://github.com/goreleaser/goreleaser/releases"

curl -o "$TAR_FILE" -fSL "https://github.com/goreleaser/goreleaser/releases/download/${VERSION}/goreleaser_$(uname -s)_$(uname -m).tar.gz"; \
	echo "${CHECKSUM}  ${TAR_FILE}" | shasum -c -;

tar -xf "$TAR_FILE" -C /usr/local/bin/ goreleaser && rm -f $TAR_FILE
"/usr/local/bin/goreleaser" "$@"
