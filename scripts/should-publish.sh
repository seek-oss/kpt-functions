#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat << EOF
Usage: $(basename "${0}") <master|tag> <git-ref>

Decides whether we should publish, based on the desired publish target and the
supplied git ref

EOF
  exit 1
}

(($# != 2)) && usage

publish_type="${1}"
git_ref="${2}"

MASTER_PUBLISH_TYPE="master"
TAG_PUBLISH_TYPE="tag"

functions=$(ls cmd)

if [[ "${publish_type}" == "${MASTER_PUBLISH_TYPE}" ]]; then
  if [[ "${git_ref}" == "refs/heads/master" ]]; then
    exit 0
  else
    exit 1
  fi
elif [[ "${publish_type}" == "${TAG_PUBLISH_TYPE}" ]]; then
  for func in ${functions}; do
    if [[ "${git_ref}" == "refs/tags/${func}/v"* ]]; then
      exit 0
    fi
  done
  exit 1
else
  echo 2>&1 "Error: Expected publish type of tag or master, got ${publish_type}"
  exit 1
fi
