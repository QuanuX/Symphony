#!/bin/sh
set -eu
[ "${SYMPHONY_I3_DISPOSABLE_HOST:-}" = 1 ] || { echo "Disposable Linux fixture opt-in required" >&2; exit 2; }
[ "$(id -u)" = 0 ] || exit 2
case "$1" in /*) fixture_binaries=$1 ;; *) exit 2 ;; esac
apt-get update -qq
apt-get install -y -qq openssh-server netcat-openbsd >/dev/null
useradd -m -u 1001 -s /bin/sh carrier
passwd -d carrier >/dev/null
mkdir -p /fixture /run/sshd
cp "$fixture_binaries/i3-carrier" /fixture/carrier
cp "$fixture_binaries/i3-carrier.test" /fixture/carrier.test
chmod 755 /fixture/*
ssh-keygen -q -t ed25519 -N '' -f /fixture/host
ssh-keygen -q -t ed25519 -N '' -f /home/carrier/key
mkdir -m700 /home/carrier/.ssh
cp /home/carrier/key.pub /home/carrier/.ssh/authorized_keys
chown -R carrier:carrier /home/carrier
chmod 600 /home/carrier/.ssh/authorized_keys
cat > /fixture/sshd_config <<'CONFIG'
Port 22222
ListenAddress 127.0.0.1
HostKey /fixture/host
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
UsePAM no
AllowUsers carrier
PidFile /fixture/sshd.pid
LogLevel ERROR
CONFIG
cat > /fixture/target <<'TARGET'
#!/bin/sh
cat >/dev/null
printf '{"fixture":true}\n'
TARGET
chmod 755 /fixture/target
/usr/sbin/sshd -f /fixture/sshd_config
runuser -u carrier -- ssh-agent -D -a /home/carrier/agent.sock >/fixture/agent.log 2>&1 &
for n in 1 2 3 4 5; do [ ! -S /home/carrier/agent.sock ] || break; sleep 1; done
runuser -u carrier -- env SSH_AUTH_SOCK=/home/carrier/agent.sock ssh-add /home/carrier/key >/dev/null 2>&1
SYMPHONY_I3_ROOT_FIXTURE=1 /fixture/carrier.test -test.v -test.run '^TestInstalledProfileRealOpenSSH$'
