#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/common.sh
source .buildkite/scripts/image-util.sh

docker_img=$1
base_commit=$(get_base_commit)

old_tag=$(docker_commit_tag "${docker_img}" "${base_commit}")
new_tag="${BUILDKITE_TAG}"

echo ":: Re-tagging image from ${old_tag} to ${new_tag} ::"
retry 3 docker pull "${old_tag}"
retry 3 docker tag  "${old_tag}" "${new_tag}"
retry 3 docker push "${new_tag}"
