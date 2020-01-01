#!/bin/sh
set -e

if [ -f "/usr/local/bin/goreleaser" ]; then
    "/usr/local/bin/goreleaser" "$@"
	exit $?
fi

VERSION=v0.123.3
CHECKSUM=efaafbb3ced464274bc3574edcf31be1d4c69f7a5932fa9b741275add2d35ab8
TAR_FILE="/tmp/goreleaser_$(uname -s)_$(uname -m).tar.gz"
RELEASES_URL="https://github.com/goreleaser/goreleaser/releases"

curl -o "$TAR_FILE" -fSL "https://github.com/goreleaser/goreleaser/releases/download/${VERSION}/goreleaser_$(uname -s)_$(uname -m).tar.gz"; \
	echo "${CHECKSUM}  ${TAR_FILE}" | shasum -c -;

tar -xf "$TAR_FILE" -C /usr/local/bin/ goreleaser && rm -f $TAR_FILE
"/usr/local/bin/goreleaser" "$@"
