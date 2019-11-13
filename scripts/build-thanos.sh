#!/usr/bin/env bash

set -e
set -x
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail

SHA=$1
LOCAL_REPO=$2
if [ ${LOCAL_REPO} != "" ]; then
  echo "local building from ${LOCAL_REPO}"
  cd ${LOCAL_REPO}
  # Repo has to be clean.
  git checkout "${SHA}"
else
  TMP_REPO=/tmp/thanos
  echo ">> fetching thanos@${SHA} revision/version"
  if [ ! -d ${TMP_REPO} ]; then
    git clone git@github.com:thanos-io/thanos.git ${TMP_REPO}
  fi

  cd ${TMP_REPO} && git fetch origin && git reset --hard "${SHA}"
fi

if ! docker images | grep "${SHA}"; then
  TMP_GOPATH=/tmp/gopath
  mkdir -p ${TMP_GOPATH}

  echo ">> building docker"
  GOPATH=${TMP_GOPATH} make docker
  docker tag "thanos" "thanos-local:${SHA}"
fi

echo ">> loading thanos-local:${SHA} image to kind"
kind load docker-image "thanos-local:${SHA}"

