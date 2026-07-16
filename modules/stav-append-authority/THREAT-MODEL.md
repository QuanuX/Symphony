# STAV Append Authority Threat Model

## Protected Assets

- integrity and per-TOPS isolation of the future canonical audit ledger;
- authority to assign event identity, order, timestamp, and preceding digest;
- producer and query authorization;
- executable lifecycle integrity;
- separation between canonical protocol truth and implementation.

## Current Attack Surface

The scaffold exposes only local executable install/uninstall and pure path calculation. It imports the authority-free protocol kernel for TOPS-ID validation but opens no socket and parses no STAV message.

## Current Controls

- only regular source and target executables are accepted;
- install uses a same-directory temporary file, file sync, atomic rename, and directory sync;
- identical install and absent uninstall are idempotent;
- differing replacement/removal requires explicit `--force`;
- uninstall never removes directories or state;
- canonical lowercase TOPS UUID validation excludes display-name path confusion;
- canonical schemas and an authority-free codec exist, but no listener, event/receipt emission, ledger, repair, or producer surface exists.

## Future Threats Requiring Closed Gates

- unauthorized or ambiguous local producers;
- candidate spoofing, scope confusion, and field smuggling;
- concurrent ordering races and multiple writers;
- valid-looking partial writes and rollback after acknowledgement;
- digest-chain ambiguity, unsafe genesis, or non-canonical serialization;
- secret-bearing metadata or error leakage;
- recovery, rotation, retention, or repair that destroys evidence;
- qxctl or an agent gaining arbitrary append authority;
- confusing tamper evidence with non-repudiation.

None of these operational threats may be addressed by implementation invention. Their contracts must first be ratified under `knowledge/stav/`.
