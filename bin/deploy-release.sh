#!/usr/bin/env sh
set -euo pipefail

architecture=amd64
version=not-set
ostype=unknown
plugins_dir="$HOME/.terraform.d/plugins"

show_help() {
  cat <<EOF >&2
Download and deploy a version of terraform-provisioner-ansible for the OS this program is executed on.
Usage:

  $(basename $0) [-h]
  $(basename $0) [-v version] [-a architecture]

Arguments:

    -h    show this help
    -a    architecture to look for, default: amd64
    -v    released version; no default
          do not use the 'v' prefix; for release v1.0.0, use 1.0.0

EOF
  exit 1
}

while getopts ":hav:" optname ; do
  case "$optname" in
    h)
      show_help
      ;;
    a)
      architecture="$OPTARG"
      ;;
    v)
      version="$OPTARG"
      ;;
  esac
done

if [ "${version}" == "not-set" ]; then
  echo "Version not given, use -v argument." >&2
  show_help
fi

case "$OSTYPE" in
  # release does not necessarily exist for a given OS
  darwin*)  ostype=darwin ;; 
  linux*)   ostype=linux ;;
  msys*)    ostype=linux ;;
  freebsd*) ostype=freebsd ;;
  *)        echo "unknown: $OSTYPE." >&2; exit 2 ;;
esac

download_url="https://github.com/radekg/terraform-provisioner-ansible/releases/download/v${version}/terraform-provisioner-ansible-${ostype}-${architecture}_v${version}"

# GitHub stores files under a different URL, thus it redirects, we are looking for 302:
echo "Checking existence of ${download_url}..."
status=`curl -sSI HEAD "${download_url}" | grep 'Status:' | awk '{ print $2 }'`

if [ "$status" != "302" ]; then
  echo "Error: no release available for ${ostype}, version ${version}\n\tat ${download_url}" >&2
  exit 3
fi

echo "Fetching terraform-provisioner-ansible ${version} from ${download_url}..."

mkdir -p "${plugins_dir}"
curl -sSL "${download_url}" --output "${plugins_dir}/terraform-provisioner-ansible_v${version}"
chmod +x "${plugins_dir}/terraform-provisioner-ansible_v${version}"

ls -la "${plugins_dir}"/terraform-provisioner-ansible*