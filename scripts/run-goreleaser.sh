#!/bin/sh
set -e

if [ -f "/usr/local/bin/goreleaser" ]; then
    "/usr/local/bin/goreleaser" "$@"
	exit $?
fi

VERSION=v0.123.3
CHECKSUM=d1da899c4ec81bfd07fbc810757c504912cfca92eb87efaa27dacf2561f7e408
TAR_FILE="/tmp/goreleaser_$(uname -s)_$(uname -m).tar.gz"
RELEASES_URL="https://github.com/goreleaser/goreleaser/releases"

curl -o "$TAR_FILE" -fSL "https://github.com/goreleaser/goreleaser/releases/download/${VERSION}/goreleaser_$(uname -s)_$(uname -m).tar.gz"; \
	echo "${CHECKSUM}  ${TAR_FILE}" | shasum -c -;

tar -xf "$TAR_FILE" -C /usr/local/bin/ goreleaser && rm -f $TAR_FILE
"/usr/local/bin/goreleaser" "$@"
