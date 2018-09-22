#!/usr/bin/env sh
set -euo pipefail
base="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. "${base}/env.sh"
version=`go version | awk '{ print $3 }' | sed -e s/go//`
major=`echo ${version} | cut -d. -f1`
minor=`echo ${version} | cut -d. -f2`
if [ "${REQUIRED_GO_MAJOR}" -gt "$major" ] || [ "${REQUIRED_GO_MINOR}" -gt "$minor" ]; then
  echo "ERROR: Required golang version ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR} but found ${major}.${minor}." >&2
  exit 1
fi