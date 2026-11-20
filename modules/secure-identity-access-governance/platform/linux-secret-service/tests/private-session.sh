#!/usr/bin/env bash
set -euo pipefail
umask 077
fixture=$(mktemp -d /tmp/symphony-secret-service.XXXXXXXX)
keyring_pid=
bus_pid=
cleanup() {
    if [[ -n "$keyring_pid" ]]; then kill "$keyring_pid" 2>/dev/null || true; wait "$keyring_pid" 2>/dev/null || true; fi
    if [[ -n "$bus_pid" ]]; then kill "$bus_pid" 2>/dev/null || true; wait "$bus_pid" 2>/dev/null || true; fi
    rm -rf -- "$fixture"
}
trap cleanup EXIT
mkdir "$fixture/runtime" "$fixture/data" "$fixture/config" "$fixture/cache" "$fixture/keyring"
export XDG_RUNTIME_DIR="$fixture/runtime" XDG_DATA_HOME="$fixture/data" XDG_CONFIG_HOME="$fixture/config" XDG_CACHE_HOME="$fixture/cache"
export DBUS_SESSION_BUS_ADDRESS="unix:path=$fixture/bus"
unset DISPLAY WAYLAND_DISPLAY GNOME_KEYRING_CONTROL DBUS_STARTER_ADDRESS DBUS_STARTER_BUS_TYPE
# No service activation directories. Only our explicitly launched daemon can own the name.
cat > "$fixture/bus.conf" <<CONF
<busconfig><type>session</type><listen>$DBUS_SESSION_BUS_ADDRESS</listen><auth>EXTERNAL</auth><policy context="default"><allow send_destination="*"/><allow receive_sender="*"/><allow own="*"/></policy></busconfig>
CONF
dbus-daemon --config-file="$fixture/bus.conf" --nofork &
bus_pid=$!
for ((i=0; i<100; i++)); do
    if [[ -S "$fixture/bus" ]]; then break; fi
    sleep .05
done
"$1" absent
printf %s disposable-qualification | env HOME="$fixture" gnome-keyring-daemon --foreground --unlock --components=secrets --control-directory="$fixture/keyring" >"$fixture/daemon.log" 2>&1 &
keyring_pid=$!
gdbus wait --session --timeout 10 org.freedesktop.secrets
# This password and the GNOME login collection exist only in the disposable session.
for ((i=0; i<100; i++)); do
    if gdbus call --session --dest org.freedesktop.secrets --object-path /org/freedesktop/secrets --method org.freedesktop.DBus.Properties.Get org.freedesktop.Secret.Service Collections | grep -q /collection/login; then break; fi
    sleep .05
done
# Confirm libsecret’s optional ambient service-name override cannot redirect use.
SECRET_SERVICE_BUS_NAME=org.quanux.Symphony.UnselectedFixture "$1" lifecycle
"$1" ambiguous
"$1" cached-bus
gdbus call --session --dest org.freedesktop.secrets --object-path /org/freedesktop/secrets --method org.freedesktop.Secret.Service.Lock "['/org/freedesktop/secrets/collection/login']" >/dev/null
"$1" locked
