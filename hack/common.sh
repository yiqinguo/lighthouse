#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

PACKAGE_PREFIX="github.com/mYmNeo"
PACKAGE_NAME="lighthouse"
PACKAGE="${PACKAGE_PREFIX}/${PACKAGE_NAME}"
OUTPUT_PATH="${BASE_DIR}/_output"
mkdir -p ${OUTPUT_PATH}
USER_ID=$(id -u)
GROUP_ID=$(id -g)

GIT_COMMIT=$(git rev-parse "HEAD^{commit}")
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

GIT_VERSION=$(cat ${BASE_DIR}/VERSION)
if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
  TREE_STATE="clean"
else
  TREE_STATE="-dirty"
  GIT_VERSION+=${TREE_STATE}
fi

if [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?([-].*)?([+].*)?$ ]]; then
  GIT_MAJOR=${BASH_REMATCH[1]}
  GIT_MINOR=${BASH_REMATCH[2]}
  if [[ -n "${BASH_REMATCH[4]}" ]]; then
    GIT_MINOR+="+"
  fi
fi
