#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat << EOF
Usage: $(basename "${0}") <function|version> <git-ref>

Extracts version and function information from the provided git ref.

Expects a ref of the format 'refs/tags/function-name/v1.2.3'

EOF
  exit 1
}

(($# != 2)) && usage

extract_type="${1}"
git_ref="${2}"

if [[ "${git_ref}" != "refs/tags/"* ]]; then
  echo 2>&1 "Error: invalid git ref format. Expected ref that started with refs/tags/, got ${git_ref}"
  exit 1
fi

function_string="function"
version_string="version"

tag_function=$(cut -d'/' -f3 <<< "${git_ref}")
tag_version=$(cut -d'/' -f4 <<< "${git_ref}")

if [[ "${extract_type}" == "${version_string}" ]]; then
  # If the version starts with a v, omit the leading v
  if [[ "${tag_version}" == "v"* ]]; then
    echo "${tag_version:1}"
    exit 0
  else
    echo "${tag_version}"
    exit 0
  fi
fi

if [[ "${extract_type}" == "${function_string}" ]]; then
  echo "${tag_function}"
  exit 0
fi

echo 2>&1 "Error: Invalid extraction type, got ${extract_type}, expected function or version"
exit 1
