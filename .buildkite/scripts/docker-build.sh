#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/common.sh
source .buildkite/scripts/image-util.sh

docker_img=$1
base_commit=$(get_base_commit)

commit_tag=$(docker_commit_tag "${docker_img}" "${base_commit}")
branch_tag=$(docker_branch_tag "${docker_img}" "${BUILDKITE_BRANCH}")

echo ":: Building image - ${commit_tag} ::"
build_image "${commit_tag}" "${base_commit}"

echo ":: Pushing image - ${commit_tag} ::"
retry 3 docker push "${commit_tag}"

echo ":: Re-tagging image from ${commit_tag} to ${branch_tag} ::"
retry 3 docker tag  "${commit_tag}" "${branch_tag}"
retry 3 docker push "${branch_tag}"
