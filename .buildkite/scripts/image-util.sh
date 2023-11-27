#!/usr/bin/env bash

set -euo pipefail

get_base_commit() {
  local base_commit

  if [[ ${BUILDKITE_PULL_REQUEST} != "false" ]]; then
    base_commit=$(git rev-parse "origin/pr/${BUILDKITE_PULL_REQUEST}" | sed 's/ *$//g')
  fi

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
  local branch=$(echo "$2" | awk -F':' '{print $2}')

  echo "${image}:${branch}"
}

build_image() {
  local commit_tag=$1
  local base_commit=$2

  docker build \
    -t "${commit_tag}" \
    --label BRANCH_NAME="${BUILDKITE_BRANCH}" \
    --label GIT_SHA="${base_commit}" \
    --label GO_VERSION="${GOLANG_VERSION}" \
    --label TIMESTAMP="$(date +%Y-%m-%d_%H:%M)" \
    .
}
