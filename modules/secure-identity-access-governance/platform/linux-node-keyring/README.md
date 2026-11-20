# Linux node keyring

Deploy this package on a trading node to stage credentials over OpenSSH and authenticate a native module from the node's local Secret Service. Each service account has a dedicated trust domain. On session startup and reconnect, its C++ module borrows one bounded binary value for the authentication handshake and keeps the resulting venue session. The order and market-data loops perform no Keychain, D-Bus or SSH calls. Requests such as exchange-mandated per-order signatures require a separate measured in-process signer or hardware signing profile.

## Installation and node identity

Build on the Ubuntu release and architecture you intend to deploy. From this directory run `bash packaging/build-deb.sh /absolute/output`. Install the resulting Debian package through your deployment system on the selected node. It contains the Linux provider SDK, node SDK, receiver, operator tools, and inactive service profiles. The install prefix is `/opt/symphony`; executables are in `libexec/symphony/ssiag-node-keyring/0.1.0-dev` under that prefix. Select a clean or dpkg-managed destination for these versioned files. CMake consumers link `Symphony::NodeKeyring` from `find_package(SymphonyNodeKeyring 0.1.0 EXACT CONFIG REQUIRED)`. Development builds require the libsecret headers. Verify the package through your own signed software distribution channel.

Create a non-root account per trading-service trust domain. Preselect its local bus, collection, TOPS UUID, reference UUID, and private receipt directory. Use a separate opaque reference for each field, or version a binary pair together when its key and password must rotate atomically. Keep names, API keys and account information out of UUIDs, labels and filenames. The node refuses missing or locked stores.

For an unattended node, use the [headless systemd profile](headless/README.md). Enrollment creates a local encrypted unlock credential. At boot the service manager unseals it locally and starts a private Unix bus and keyring for the service account. Select TPM2 sealing for a hardware-backed unattended restart, or explicitly select host-key sealing on a software-only node. A machine unable to unseal its selected credential stays unavailable until recovery. A [FIDO2 volume profile](hardware/README.md) is available for sites requiring an attended PIN and touch at cold boot.

## Stage a version over SSH

Register the node's SSH public host key with the controller through trusted inventory. Use an authorized provisioning key per destination slot. Put its fixed import command in a root-owned authorized-keys file **outside the service user's writable home**. The account's sshd policy must allow only public-key authentication on this ingress, disable forwarding, TTY, user RC and user environment files, and accept no client-controlled environment that could affect the shell, loader or command. Keep server command paths and all configuration ancestors protected. The SSH account uses the keyring service UID. Grant only this restricted key access to this account.

```text
restrict,command="/opt/symphony/libexec/symphony/ssiag-node-keyring/0.1.0-dev/symphony-node-keyring-import COLLECTION TOPS_UUID REFERENCE_UUID RECEIPT_DIR BUS_SOCKET" ssh-ed25519 PROVISIONER_PUBLIC_KEY
```

Replace the placeholders with literal administrator-selected values. `RECEIPT_DIR` is an existing 0700 directory owned by the service UID. Keep it durable across reboot and place the actual persistent vault and its backups under equivalent protection. The headless service uses `/run/symphony-node-keyring-UID/bus`; a desktop user manager normally uses `/run/user/UID/bus`. Use the same configured path in the forced command and consumer binding. Both the service provider and the client connection to the selected bus must belong to the expected UID.

On the controller, feed one opaque value from a trusted credential provider to the supplied Python 3 sender on an anonymous pipe. Give a fresh request UUID and the exact expected length in bytes. The entire secret source must succeed before transfer. The verified OpenSSH client reads the restricted identity and pinned host-key file from protected local paths. The source value never appears in the command line. Use bounded upstream access, disable session/stream recording for this command, and run both ends with protected process memory and swap appropriate to the host.

```sh
trusted-secret-source | symphony-node-keyring-send --host NODE --port 22 --user SERVICE_USER \
  --known-hosts /etc/symphony/pins/NODE --identity /secure/keys/PROVISIONER \
  --request REQUEST_UUID --tops TOPS_UUID --reference REFERENCE_UUID --length EXPECTED_BYTES
```

`trusted-secret-source` represents the application's existing protected value provider. The SSH client always requests the literal command `symphony-keyring-import-v1`. Inside the encrypted channel the request is one ASCII header, `SNKI1 REQUEST_UUID TOPS_UUID REFERENCE_UUID BYTE_LENGTH\n`, followed by 1–4096 raw bytes and EOF. No network service is added to SSIAG. The existing SOV carrier handles auditable control messages and does not carry this payload.

`STORED request generation` identifies a newly staged generation. Repeating a committed request yields `ALREADY_STORED` with its original generation, even if different bytes are supplied. Never infer equality of secret values from that result. On a missing reply, reconcile the original request with the local deployment authority. A durable incomplete reservation returns `UNCERTAIN` and fences further writes under that ID. Inspect and reconcile the selected slot before issuing a replacement request. Pending or orphaned generations may need explicit retention/retirement by the owner. Retain request receipts while their IDs may be replayed. Restore receipts and the vault consistently after a disaster.

## Select the generation locally

The node's deployment authority installs a metadata-only binding under a root-owned, non-writable directory such as `/etc/symphony/node-keyring`. Use a regular file readable by the service UID, with a terminal newline and the exact field order below. The value of `generation` is the UUID from a confirmed import.

```text
SYMPHONY-NODE-KEYRING-1
uid=1001
bus_socket=/run/user/1001/bus
collection=/org/freedesktop/secrets/collection/login
tops=TOPS_UUID
reference=REFERENCE_UUID
generation=GENERATION_UUID
maximum_bytes=4096
```

Once enrolled, a local rollout service can rotate the binding by calling `symphony-node-keyring-activate BINDING EXPECTED_GENERATION NEW_GENERATION`. It replaces the exact generation atomically and refuses stale updates. No interactive operator is required for this operation. Use the same conditional update to roll back to a retained version. A switch affects new handshakes. Drain old sessions and revoke or rotate the old venue-side credential before retiring it using the Secret Service SDK. If an active venue session needs immediate termination, the session owner must close it.

## Fetch locally from a C++ module

```cpp
#include <symphony/ssiag/node_keyring_bootstrap.hpp>

namespace node = symphony::ssiag::node_keyring;
node::SessionBootstrap login("/etc/symphony/node-keyring/venue-a");
auto result = login.authenticate(node::Clock::now() + std::chrono::seconds(5), stop,
    [&](std::span<const std::byte> credential, auto deadline, std::stop_token cancel) {
        return venue.authenticate(credential, deadline, cancel);
    });
```

The venue adapter returns `node::Handshake::established`, `refused`, or `indeterminate`. It must honor the deadline and cancellation, avoid logging the value, and release borrowed bytes when the callback returns. A fresh bootstrap object rereads the binding for each reconnect. The local API opens the selected Unix socket, checks the bus peer and Secret Service owner, then addresses the configured collection and exact generation. No controller connection is required for lookup. Keep core dumps disabled and debugger/ptrace access restricted on credential-bearing processes.

For APIs that authenticate every order, initialize the venue SDK's signing context in the bootstrap callback. Sign each request locally according to that venue's protocol and replace the context on rotation. Keep its lifetime bounded by the connector's credential policy. Neither this package nor a controller can revoke bytes an already running process has retained.

This profile trusts all code with access as the dedicated service UID. It provides neither mutual distrust between plugins in that process nor a new SSIAG lease issuer. Deploy untrusted modules under separate accounts. The [shared SSIAG qualification handoff](../../CREDENTIAL-QUALIFICATION.md) describes the additional per-recipient, lease and audit work for the general governed broker.
