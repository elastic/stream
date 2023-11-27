#!/usr/bin/env bash

set -euo pipefail

get_base_commit() {
  if [[ ${BUILDKITE_PULL_REQUEST} != "false" ]]; then
    git rev-parse "origin/pr/${BUILDKITE_PULL_REQUEST}" | sed 's/ *$//g'
  else
    git rev-parse HEAD
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
  local branch_tag

  if [[ "${branch}" == *:* ]]; then
    branch_tag="$(echo "${branch}" | awk -F':' '{print $2}')"
  else
    branch_tag="${branch}"
  fi

  echo "${image}:${branch_tag}"
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
