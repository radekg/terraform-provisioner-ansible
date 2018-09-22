#!/usr/bin/env sh
base="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
set -euo pipefail

version_file="${base}/../.version"

version=`cat ${version_file} | head -n1`

local_output_dir="${base}/.build_output"
docker_output_dir=/output
docker_gopath="/golang"
project=`echo $(dirname "$base") | sed -e 's!'$GOPATH'!!'`

mkdir -p "${local_output_dir}"
rm -rf "${local_output_dir}/*"

path="/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${docker_gopath}/bin"

docker run \
  -e GOPATH="${docker_gopath}" \
  -e RELEASE_VERSION="v${version}" \
  --rm \
  -v "${GOPATH}${project}":"${docker_gopath}${project}" \
  -v "${local_output_dir}":"${docker_output_dir}" \
  -w "${docker_gopath}${project}" \
  golang:1.11 \
  /bin/bash -c "export PATH=${path} && make build-release && mv ${docker_gopath}/bin/terraform-provisioner-ansible* ${docker_output_dir}/"