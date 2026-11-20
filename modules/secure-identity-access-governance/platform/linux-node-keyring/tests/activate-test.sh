#!/usr/bin/env bash
# Run inside a disposable Linux container with a private, root-owned /run.
set -euo pipefail
[[ $(id -u) == 0 && $# == 1 && -x $1 ]]
helper=$(realpath -- "$1")
root=$(mktemp -d /run/symphony-activate-XXXXXXXX)
trap 'rm -rf -- "$root"' EXIT
target="$root/venue.binding"
old=$(cat /proc/sys/kernel/random/uuid)
first=$(cat /proc/sys/kernel/random/uuid)
second=$(cat /proc/sys/kernel/random/uuid)
third=$(cat /proc/sys/kernel/random/uuid)
cat > "$root/baseline" <<BINDING
SYMPHONY-NODE-KEYRING-1
uid=1000
bus_socket=/run/user/1000/bus
collection=/org/freedesktop/secrets/collection/login
tops=$(cat /proc/sys/kernel/random/uuid)
reference=$(cat /proc/sys/kernel/random/uuid)
generation=$old
maximum_bytes=4096
BINDING

expect() {
    local expected_rc=$1 expected_line=$2 file=$3 from=$4 to=$5 rc=0
    timeout 5s "$helper" "$file" "$from" "$to" > "$root/out" 2> "$root/err" || rc=$?
    [[ $rc == "$expected_rc" && ! -s "$root/err" ]]
    printf '%s\n' "$expected_line" | cmp - "$root/out"
}
expect 1 REFUSED "$target" "$old" "$first"
install -m 0640 "$root/baseline" "$target"
expect 1 REFUSED "$target" "INVALID" "$first"
expect 3 STALE "$target" "$second" "$first"
ln -s -- "$target" "$root/link"
expect 1 REFUSED "$root/link" "$old" "$first"
ln -- "$target" "$root/hard"
expect 1 REFUSED "$target" "$old" "$first"
unlink "$root/hard"
chmod 0770 "$root"
expect 1 REFUSED "$target" "$old" "$first"
chmod 0700 "$root"
chmod 0666 "$target"
expect 1 REFUSED "$target" "$old" "$first"
chmod 0640 "$target"
sed 's/maximum_bytes=4096/maximum_bytes=4097/' "$root/baseline" > "$target"
expect 1 REFUSED "$target" "$old" "$first"
cat "$root/baseline" > "$target"
if command -v setfattr >/dev/null && setfattr -n user.example -v retain "$target"; then
    expect 1 REFUSED "$target" "$old" "$first"
    [[ $(getfattr --only-values -n user.example "$target") == retain ]]
    setfattr -x user.example "$target"
fi
timeout 5s "$helper" "$target" "$old" "$first" > "$root/first" & a=$!
timeout 5s "$helper" "$target" "$old" "$second" > "$root/second" & b=$!
ra=0; rb=0
wait "$a" || ra=$?
wait "$b" || rb=$?
if [[ $ra == 0 && $rb == 3 ]]; then
    winner=$first; winfile=$root/first; losefile=$root/second
elif [[ $rb == 0 && $ra == 3 ]]; then
    winner=$second; winfile=$root/second; losefile=$root/first
else
    echo "Competing generation changes returned $ra and $rb" >&2
    exit 1
fi
printf 'ACTIVE %s\n' "$winner" | cmp - "$winfile"
printf 'STALE\n' | cmp - "$losefile"
[[ $(stat -c %a "$target") == 640 ]]
sed "s/generation=$old/generation=$winner/" "$root/baseline" | cmp - "$target"
expect 0 "ALREADY_ACTIVE $winner" "$target" "$old" "$winner"
expect 3 STALE "$target" "$old" "$third"
expect 0 "ACTIVE $third" "$target" "$winner" "$third"
[[ -z $(find "$root" -maxdepth 1 -name '*.tmp' -print -quit) ]]
echo "PASS owner-only generation activation, concurrent CAS and retry"
