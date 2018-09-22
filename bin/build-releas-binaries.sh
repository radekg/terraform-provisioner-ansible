#!/usr/bin/env sh
set -euo pipefail
base="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. "${base}/env.sh"

version_file="${base}/../.version"

version=`cat ${version_file} | head -n1`

local_output_dir="${base}/.build_output"
docker_output_dir=/output
docker_gopath="/golang"
project=`echo $(dirname "$base") | sed -e 's!'$GOPATH'!!'`

rm -rf "${local_output_dir}"
mkdir -p "${local_output_dir}"

path="/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${docker_gopath}/bin"

docker run \
  -e GOPATH="${docker_gopath}" \
  -e RELEASE_VERSION="v${version}" \
  --rm \
  -v "${GOPATH}${project}":"${docker_gopath}${project}" \
  -v "${local_output_dir}":"${docker_output_dir}" \
  -w "${docker_gopath}${project}" \
  golang:${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR} \
  /bin/bash -c "export PATH=${path} && make build-release && mv ${docker_gopath}/bin/terraform-provisioner-ansible* ${docker_output_dir}/"