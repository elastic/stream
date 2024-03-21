#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/common.sh
source .buildkite/scripts/image-util.sh

docker_img=$1
base_commit=$(get_base_commit)

commit_tag=$(docker_commit_tag "${docker_img}" "${base_commit}")
branch_tag=$(docker_branch_tag "${docker_img}" "${BUILDKITE_BRANCH}")

echo ":: Building and pushing image - ${commit_tag} ::"
build_and_push_image "${base_commit}" -t "${commit_tag}" -t "${branch_tag}"
