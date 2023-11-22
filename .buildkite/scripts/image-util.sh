#!/usr/bin/env bash

set -euo pipefail

get_base_commit() {
  local base_commit
  base_commit=$(git rev-parse "origin/pr/${BUILDKITE_PULL_REQUEST}" | sed 's/ *$//g')

  if [[ -z $base_commit ]]; then
    git rev-parse HEAD
  else
    echo "${base_commit}"
  fi
}

docker_commit_tag() {
  local image=$1
  local commit=$2

  echo "${image}:${commit}"
}

docker_branch_tag() {
  local image=$1
  local branch=$2

  echo "${image}:${branch}"
}

build_image() {
  docker build \
    -t ${dockerImageGitCommitTag} \
    --label BRANCH_NAME=${BUILDKITE_BRANCH} \
    --label GIT_SHA=${env.GIT_BASE_COMMIT} \
    --label GO_VERSION=${env.GO_VERSION} \
    --label TIMESTAMP=$(date +%Y-%m-%d_%H:%M) \
    .
}
