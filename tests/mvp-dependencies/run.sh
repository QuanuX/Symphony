#!/bin/sh
set -eu
umask 077
if [ "$#" -lt 2 ]; then
  echo "usage: $0 GROUP /absolute/fresh-work [CMake configure arguments...]" >&2
  exit 2
fi
group=$1
work=$2
shift 2
case "$group" in bootstrap|knowledge|cloud|quantitative|legacy_duckdb|build_support) ;; *) echo 'unknown qualification group' >&2; exit 2 ;; esac
case "$work" in /*) ;; *) echo 'work directory must be absolute' >&2; exit 2 ;; esac
if [ -e "$work" ] || [ -L "$work" ]; then
  echo 'work directory must not already exist' >&2
  exit 2
fi
harness=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repository=${MVP_REPOSITORY:-$(CDPATH= cd -- "$harness/../.." && pwd -P)}
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/logs" "$work/tmp" "$work/prefix" "$work/metadata"
export TMPDIR="$work/tmp" SYMPHONY_TEST_TMPDIR="$work/tmp"
arguments="$work/configure-arguments.cmake"
printf 'set(MVP_EXTRA_ARGS "")\n' > "$arguments"
for argument do
  case "$argument" in *']==]'*) echo 'unsupported CMake bracket delimiter in argument' >&2; exit 2 ;; esac
  printf 'string(REPLACE ";" "\\\\;" _argument [==[%s]==])\nlist(APPEND MVP_EXTRA_ARGS "${_argument}")\n' "$argument" >> "$arguments"
done
{
  uname -srm
  if [ "$(uname -s)" = Darwin ]; then
    sw_vers
  else
    stat -f -c 'Selected work filesystem: %T' "$work"
    if [ -r /etc/os-release ]; then cat /etc/os-release; fi
  fi
  df -h "$work"
  cmake --version
  "${CXX:-c++}" --version
  git -C "$repository" rev-parse HEAD
  git -C "$repository" status --short
} > "$work/logs/host-info.log"
cat "$work/logs/host-info.log"
cmake -DMVP_REPOSITORY="$repository" -P "$harness/verify.cmake"
cmake -DMVP_REPOSITORY="$repository" -DMVP_WORK="$work" -DMVP_GROUP="$group" \
  -DMVP_ARGUMENTS_FILE="$arguments" -DMVP_JOBS="${MVP_BUILD_JOBS:-2}" \
  -P "$harness/driver.cmake"
