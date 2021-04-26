#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat << EOF
Usage: $(basename "${0}") <function|version>

Extracts version and function information from the current git tag.

Expects a tag of the format 'v1.2.3/function-name'

EOF
  exit 1
}

(($# != 1)) && usage

extract_type="${1}"

function_string="function"
version_string="version"

# If GIT_TAG is not set, just skip processing.
if [[ -z "${GIT_TAG+x}" ]]; then
  echo 2>&1 "Error: GIT_TAG unset"
  exit 1
fi

tag_version=$(cut -d'/' -f1 <<< "${GIT_TAG}")
tag_function=$(cut -d'/' -f2 <<< "${GIT_TAG}")

if [[ "${extract_type}" == "${version_string}" ]]; then
  echo "${tag_version}"
  exit 0
fi

if [[ "${extract_type}" == "${function_string}" ]]; then
  echo "${tag_function}"
  exit 0
fi

echo 2>&1 "Error: Invalid extraction type, got ${extract_type}, expected function or version"
exit 1
