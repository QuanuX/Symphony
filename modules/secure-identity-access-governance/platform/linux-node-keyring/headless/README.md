# Unattended local Linux keyring

A dedicated system service starts a private Unix session bus and GNOME Keyring under the trading workload's UID. The node unlocks its own encrypted login collection on restart using a systemd encrypted credential. Fresh consumer boots and reconnects can use the active credential generation while the controller is offline. The exchange connection itself still needs its normal network path.

The bootstrap credential unlocks the vault. Trading credentials remain items in the vault. The selected service UID, every process trusted to run as that UID, and the host administrator form the trust boundary. Use a dedicated account per independent application or trust domain. Restrict debugger access, disable core dumps for consumers and use encrypted swap or disable swap. An administrator or a compromised running workload can access plaintext in memory.

## Select the cold-start policy

| Mode | What unlocks the vault after reboot |
| --- | --- |
| `host` | The root-owned local systemd host key opens the sealed bootstrap credential. A copy of the disk including that host key can be sufficient to unlock it. Protect the underlying disk accordingly. |
| `tpm2` | The local TPM2 and the recorded PCR 7 value open the bootstrap credential. Boot or Secure Boot policy changes can make it unavailable. Select a node-owner recovery/reprovisioning procedure before relying on this mode. |
| Attended FIDO2 | A separate deployment profile can require PIN and physical touch to open the keyring volume. A restart then needs the token and operator. See the [hardware profile](../hardware/README.md). |

The TPM mode uses explicit `systemd-creds encrypt --with-key=tpm2 --tpm2-device=auto --tpm2-pcrs=7`. Both supported unattended modes are available through the system service manager. Review the node's TPM model, ownership, measured boot and recovery policy. [systemd-creds](https://www.freedesktop.org/software/systemd/man/259/systemd-creds.html) · [systemd service credentials](https://www.freedesktop.org/software/systemd/man/259/systemd.exec.html#Credentials)

## Enroll an empty node vault

1. Install the verified node package under `/opt/symphony` and the operating system packages for systemd credentials, Python 3, D-Bus, gdbus and GNOME Keyring. Use a dedicated non-root numeric UID with its own primary GID, no supplementary groups and a private 0700 real home directory. The UID's passwd entry identifies the home. Keep other desktop sessions, keyring daemons, PAM keyring unlocks and unrestricted logins out of that account.
2. Install the two templates in this directory as root-owned system units in `/etc/systemd/system/` and run `systemctl daemon-reload`. Set `uid` below to the selected numeric service UID. Only the node administrator performs enrollment.
3. Generate the local vault password and seal it using one explicitly selected mode. This example selects a TPM. To select the host-key mode, use `host` as the last argument.

   ```sh
   /opt/symphony/libexec/symphony/ssiag-node-keyring/0.1.0-dev/seal_unlock.py "$uid" tpm2
   systemctl start "symphony-node-keyring-enroll@$uid.service"
   systemctl enable --now "symphony-node-keyring@$uid.service"
   ```

   Sealing atomically creates a new encrypted file under `/etc/credstore.encrypted/`. An existing file or populated GNOME vault prevents enrollment. If sealing succeeds and initial enrollment is interrupted, inspect that node’s existing ciphertext and vault status before retrying the enroll unit. A missing seal on an existing vault requires recovery with the corresponding password or intentional reprovisioning.
4. Set the import policy and the protected consumer binding to the exact socket `/run/symphony-node-keyring-<uid>/bus` and collection `/org/freedesktop/secrets/collection/login`. The node administrator selects the active generation after staging each secret. Workload units can require the exact instance with `Requires=` and `After=`. The daemon reports systemd readiness only after confirming that the encrypted login collection is unlocked.
5. Import generated test data, authenticate once from the installed consumer, stop and start the keyring service, disconnect the provisioning channel, and authenticate again using the same selected generation. Follow the trading connector's own reconnect and session-expiry rules. Verify the real service’s startup and reboot behavior on the target node before activation.

The service manager decrypts `vault-unlock` into its protected credentials directory. The launcher gives that open file to the keyring daemon on standard input. Runtime files and the private session bus live under `/run`. Only encrypted credential material belongs on persistent storage. Normal startup requires the existing encrypted login keyring under the service account's `.local/share/keyrings`. Missing or incorrect unlock material, a missing vault or an unavailable TPM leaves the service unavailable. Restarting the keyring does not revoke an established exchange session.

## Qualification

`test_restart.py` runs in a disposable Ubuntu container with networking disabled. It starts the actual D-Bus and GNOME daemons under a test UID, provisions a binary fixture through the installed importer, stops the daemons, checks missing and incorrect unlock refusals, then starts fresh daemons and exercises the installed C++ consumer on the original generation. It simulates the systemd credentials directory using private tmpfs. No physical TPM, production venue or actual machine reboot is part of that test.
