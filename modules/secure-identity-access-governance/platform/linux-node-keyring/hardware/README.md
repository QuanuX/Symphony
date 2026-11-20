# Optional hardware unlock for a Linux node

This deployment profile puts a dedicated GNOME Keyring account's home on a LUKS2 volume. A local FIDO2 authenticator supporting `hmac-secret` (including suitable YubiKeys) supplies the volume unlock. The deployment requires PIN and touch. Once the volume and collection are unlocked, local consumers can establish or reconnect their trading sessions without a control-plane connection. Removing the token leaves an already open volume and existing trading sessions usable. Choose this profile when a human can attend a cold restart.

## Prepare the node

1. Select a dedicated Linux account whose home is `/var/lib/symphony/nodevault`. Its systemd user manager must own the sole trusted session bus for the keyring, normally `/run/user/<uid>/bus`. Use the systemd-managed GNOME daemon. Configure PAM, login and D-Bus activation so a second daemon cannot start outside that manager. Other node services and unrestricted SSH sessions must run under separate accounts. Processes sharing the keyring UID and bus remain in the same trust domain.
2. Have the node administrator provision a dedicated ext4 filesystem directly on the `symphonynodevault` LUKS2 mapping. Mount the filesystem root at the account's exact home. Keep the underlying empty mountpoint root-owned and inaccessible to the account while unmounted. On the mounted volume pre-create `.local/share/keyrings` as the account. Both its home and the keyrings directory must be private (0700). Use the GNOME provider's normal local collection-unlock procedure. Volume unlock and collection unlock are separate.
3. Enroll a FIDO2 token with `systemd-cryptenroll`, requiring `--fido2-with-client-pin=yes` and `--fido2-with-user-presence=yes`. Register the second physical recovery token and prove it opens the disposable test volume before choosing how to retire any bootstrap key. The supplied checker accepts only slots associated one-to-one with PIN-and-touch FIDO2 enrollments. An independent passphrase, recovery-key or TPM slot changes the unlock policy and the checker will refuse it. Keep existing data or keyslots intact until the node owner approves and verifies a recovery procedure. Systemd documents token requirements and supported firmware options. [systemd-cryptenroll](https://www.freedesktop.org/software/systemd/man/255/systemd-cryptenroll.html)
4. Configure the boot mapping and mount for the selected LUKS UUID. A crypttab entry has the form `symphonynodevault UUID=<LUKS-UUID> - fido2-device=auto`. Configure the matching system mount at `/var/lib/symphony/nodevault`. Protect stored volume-key material, header backups, swap and crash dumps under the node owner's recovery policy. The package does not install, format, enroll or mount devices. [crypttab](https://www.freedesktop.org/software/systemd/man/255/crypttab.html)

## Apply the startup gate

Install these assets from the verified package on the selected node. The system unit, checker, selection file and their ancestors must be owned by root and protected against writes by the service account.

| Source | Destination |
| --- | --- |
| Installed native checker | `/opt/symphony/libexec/symphony/ssiag-node-keyring/0.1.1-dev/symphony-node-keyring-fido2` |
| Plain text containing the selected lowercase LUKS UUID | `/etc/symphony/node-keyring/luks-uuid` |
| `symphony-node-keyring-fido2@.service` | `/etc/systemd/system/symphony-node-keyring-fido2@.service` |
| `user-manager.conf` | `/etc/systemd/system/user@<numeric-uid>.service.d/80-symphony-keyring.conf` |
| `keyring-daemon.conf` | `/var/lib/symphony/nodevault/.config/systemd/user/gnome-keyring-daemon.service.d/80-symphony-data-home.conf` |

The checker requires Linux, cryptsetup, the system manager's device namespace, procfs, sysfs and udev's block-device links. It reads mount and enrollment metadata and never opens a keyring item. It checks the actual mounted mapper and its backing device, UUID, filesystem root, keyring directory and active keyslots. The user manager depends on the check and the check depends on the mount. Losing the mount stops the dependent units. The dedicated manager and daemon must start fresh under these settings before enabling import or consumption. A running previously configured daemon or manually started session has not passed this gate. Stopping services can interrupt live trading; perform activation in a node maintenance window. [systemd unit dependencies](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html)

When using the package's headless service with this attended disk-unlock policy, apply the same `user-manager.conf` contents to the selected `symphony-node-keyring@<uid>.service.d/` and `symphony-node-keyring-enroll@<uid>.service.d/` under `/etc/systemd/system`. That profile runs its own daemon and private bus inside the system service. It already selects the data directory beneath this account's home, so the desktop daemon drop-in applies only to the desktop profile. Pre-create the empty private directories before initial enrollment. For unattended cold starts, configure the [headless service](../headless/README.md) with its TPM2 unlock policy.

The checker observes the active mapping and enrollment policy. It cannot attest which method opened a mapping that was already active, prove hardware firmware policy, or prevent a privileged administrator with access to an exported volume key from reopening it. Recheck the configuration whenever enrollment or mount topology changes. A per-operation hardware signature would require venue or protocol support and a separate consumer implementation.

## Qualify a physical node

On a spare node and disposable volume, cold-start with the authenticator absent. Verify the mount and user manager stay inactive, import fails, and a fresh client cannot establish a new venue session. Repeat with an unenrolled token. Then unlock locally with the enrolled token, PIN and touch, start the selected account's normal keyring session and run the package's generated-value import and consumer exercise with upstream credential services disconnected. Check recovery with the second token and cleanup with the dedicated services stopped. Record the results without PINs, token metadata, key material or trading credentials.

Run `ctest --test-dir BUILD_DIR -R node-keyring-hardware-policy --output-on-failure` after a test-enabled node build to exercise the mount and enrollment policy on synthetic input. Physical unlock, account activation and venue connectivity require the selected hardware and node.
