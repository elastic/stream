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

build_and_push_image() {
  local base_commit=$1
  shift
  local extra_args=("$@")

  docker buildx create --platform linux/amd64,linux/arm64 --use
  docker buildx build \
    --progress=plain \
    --platform linux/amd64,linux/arm64 \
    --push \
    --label BRANCH_NAME="${BUILDKITE_BRANCH}" \
    --label GIT_SHA="${base_commit}" \
    --label GO_VERSION="${GOLANG_VERSION}" \
    --label TIMESTAMP="$(date +%Y-%m-%d_%H:%M)" \
    ${extra_args[@]} \
    .
}
