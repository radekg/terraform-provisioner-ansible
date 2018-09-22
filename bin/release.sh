#!/usr/bin/env sh
base="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
set -eu

git_release_branch=master
version_file="${base}/../.version"

version=`cat ${version_file} | head -n1`

default="\033[0m"
red="\033[1;31m"
green="\033[1;32m"

log() {
  echo $*
}

git_check_branch() {
  local given_branch="${1}"
  local current_branch=`git branch | grep '*' | awk '{ print $2 }'`
  if [ "${current_branch}" != "${given_branch}" ]; then
    log " - current branch is ${red}${current_branch}${default}"
    log " - releases can be created only from ${green}${given_branch}${default} branch"
    exit 1
  else
    log " - branch is ${green}${given_branch}${default}."
  fi
}

git_check_origin_ahead() {
  local given_branch="${1}"
  local num_commits=`git rev-list --left-right --count "origin/${given_branch}...${given_branch}" | awk '{ print $2 }'`
  if [ "${num_commits}" -gt "0" ]; then
    log " - origin/${given_branch} is ahead of ${given_branch} by ${red}${num_commits}${default} commit(s)"
    exit 2
  else
    log " - ${given_branch} is ${green}up to date${default} with origin/${given_branch}"
  fi
}

next_version() {
  local given_version="${1}"
  local v_next_major=`echo ${given_version} | cut -d. -f1`
  local v_next_minor=`echo ${given_version} | cut -d. -f2`
  local v_next_patch=`echo ${given_version} | cut -d. -f3`
  local patch_increased=$(( $v_next_patch + 1 ))
  echo "${v_next_major}.${v_next_minor}.${patch_increased}-SNAPSHOT"
}

get_release_version() {
  read version_to_release
  if [ -z "${version_to_release}" ]; then
    echo "${current_version}"
  else
    echo "${version_to_release}"
  fi
}

get_next_version() {
  local given_version_after_release="${1}"
  read version_after_release
  if [ -z "${version_after_release}" ]; then
    echo "${given_version_after_release}"
  else
    echo "${version_after_release}"
  fi
}

get_continue_with_release() {
  read confirmation
  if [ -z "${confirmation}" ]; then
    echo "y"
  else
    echo "${confirmation}"
  fi
}

git_check_branch "${git_release_branch}"
git_check_origin_ahead "${git_release_branch}"

current_version=`echo ${version} | sed -e s/-SNAPSHOT//`

log ""
log "Release version [$current_version]: "
version_to_release=`get_release_version`

default_version_next=`next_version ${version_to_release}`

log "Next version [${default_version_next}]: "
version_after_release=`get_next_version ${default_version_next}`

cd "${base}/.."
make lint
make test-verbose

log ""
log "Write changes to remote repository and create the release (y/n)? [y]: "
user_confirmation=`get_continue_with_release`
if [ "${user_confirmation}" == "y" ]; then
  log "creating release ${version_to_release}..."
  git tag --force "v${version_to_release}"
  git push origin "v${version_to_release}"
  echo "${version_after_release}" > "${version_file}"
  git commit -m "Released ${version_to_release}, update version to ${version_after_release}"
  git push origin "${git_release_branch}"
  log "${green}Done \\o/${default}."
else
  log "okay, ${red}release aborted${default}"
fi