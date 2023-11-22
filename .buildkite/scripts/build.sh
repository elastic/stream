#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/common.sh

add_bin_path
with_go "${GOLANG_VERSION}"

echo ":: Starting build ::"
go build
echo "Build done!"
