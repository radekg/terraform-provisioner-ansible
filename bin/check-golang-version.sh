#!/usr/bin/env sh
REQUIRED_GO_MAJOR=1
REQUIRED_GO_MINOR=10
version=`go version | awk '{ print $3 }' | sed -e s/go//`
major=`echo ${version} | cut -d. -f1`
minor=`echo ${version} | cut -d. -f2`
if [ "$REQUIRED_GO_MAJOR" -gt "$major" ] || [ "$REQUIRED_GO_MINOR" -gt "$minor" ]; then
  echo "ERROR: Required golang version ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR} but found ${major}.${minor}." >&2
  exit 1
fi