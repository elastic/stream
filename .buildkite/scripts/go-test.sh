#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/common.sh

add_bin_path
with_go "${GOLANG_VERSION}"
with_mage

echo ":: Starting tests ::"
gotestsum --format testname --junitfile junit-report.xml -- -v ./...
echo ":: Tests done! ::"
