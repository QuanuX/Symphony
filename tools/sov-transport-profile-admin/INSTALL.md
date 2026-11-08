# Explicit protected host installation

This is an administrator-reviewed prerequisite, not an automatic qxctl action. Build the exact reviewed source with Go 1.26.5 and `GOWORK=off`; retain the source identity, `go.sum`, binary hash and focused-test evidence. No helper or package is installed by profile apply.

The versioned executable destination is:

| Platform | Administrator executable | Profile store | Package store |
| --- | --- | --- | --- |
| Linux | `/opt/symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin` | `/var/lib/symphony/sov/transport-profiles` | `/opt/symphony/transport-packages` |
| macOS | `/Library/Application Support/Symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin` | `/Library/Application Support/Symphony/sov/transport-profiles` | `/Library/Application Support/Symphony/transport-packages` |

Create all destination ancestors as root-owned directories without group/other write permission. Install a reviewed regular single-link binary, mode 0755, and verify its bytes against the selected SHA256. Do not run an installer from a user-writable tree as root or preserve unreviewed ownership/modes. The process verifies all ancestors from `/`; symlinked convenience paths are not supported. A normal unprivileged prefix installation is insufficient.

Provision one store at `<profile-store>/<host-UUID>/<TOPS-UUID>` with root-owned 0755 `profiles`, `history`, and `attempts` directories, a stable root-owned 0644 empty `LOCK`, and a root-owned 0644 `scope.json` containing exactly:

```json
{"protocol":"symphony.sov.transport-profile-scope.v1","scope":{"host_id":"11111111-1111-4111-8111-111111111111","tops_id":"22222222-2222-4222-8222-222222222222"}}
```

Replace those example IDs with the explicitly selected local host and TOPS. Never recreate or truncate an existing lock/store during an upgrade. The helper does not bootstrap arbitrary scopes. Existing system SSIAG must listen at `/run/symphony/<TOPS>/ssiag/ssiag.sock` on Linux or physical `/private/var/run/symphony/<TOPS>/ssiag/ssiag.sock` on macOS, with protected root ancestry, root socket owner and actual root server peer. Its authenticated STAV producer grant must already admit the two event/operation pairs in SPEC.md. Without committed audit, apply retains pending evidence and refuses publication.

Install each independently reviewed carrier/SSH/proxy closure at `<package-store>/<package-UUID>/<wrapper-version>`. Preserve its exact knowledge install-receipt.v2 relative file paths. Add a qualification evidence file and a `manifest.json` matching the generated PackageManifest schema. The manifest lists every file except itself, including receipt and qualification evidence. Hash complete file bytes; receipt_digest here means the SHA256 of the receipt file bytes, while that receipt separately contains its existing intrinsic receipt_digest. Bind the exact upstream receipt component/version/entry point, platform, role and executable. Seal the manifest using the domain-separated PackageProtocol algorithm in SPEC.md. Review source provenance, licenses, dynamic dependency closure and actual execution-profile qualification before admitting it. Neither arbitrary JSON labeled as a receipt nor a convenient user cloudflared installation qualifies. I2 ships no qualified I3 carrier package.

For a narrowly scoped sudo policy, replace `PROFILE_ADMIN` with the selected existing OS username and validate the entry using the host's sudoers editor. Example Linux command specification:

```sudoers
PROFILE_ADMIN ALL=(root) /opt/symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin administer
```

macOS requires escaped spaces in the path:

```sudoers
PROFILE_ADMIN ALL=(root) /Library/Application\ Support/Symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin administer
```

No wildcard, arbitrary argv, shell, SETENV, package installer or query elevation is granted. These examples retain the host's normal authentication policy. qxctl invokes sudo with `-n`; establish authorization separately if required (for example, `sudo -v`) and rerun only after inspecting any prior uncertain attempt. Do not add NOPASSWD merely to make a test pass. Existing broader host policy is outside this helper's control.

As the ordinary user, run `qxctl sov transport profile template --json`, replace synthetic identities and submit it with `profile plan --input request.json --json`. Copy the exact returned `plan` into an apply request with the same scope, a fresh request UUID and all other optional fields explicitly null. Inspect/status use the same Request shape, with only profile_id or operation_id respectively populated. List has all optional fields null. Recover populates only recovery with the operation ID, plan digest, current pending digest and `complete` or `abort`. All commands accept complete JSON or text output.

Remove authorization through the host's reviewed sudo policy and remove the exact executable to uninstall the helper. Retain profile history/audit until an explicit retention decision; removing a binary or package does not settle pending evidence or revoke an established connection. Settle pending operations before replacing the administrator binary. This installation guide is not proof of deployed host qualification.
