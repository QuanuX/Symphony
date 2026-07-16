# STAV Append Authority Intent

## Purpose

Provide the independently installable Go implementation boundary for the one serialized, tamper-evident STAV ledger assigned to each immutable TOPS identity.

## Authority Boundary

`knowledge/stav/` is the sole source of protocol and schema truth. This module implements those contracts. The authority alone assigns event UUID, nanosecond timestamp, sequence, predecessor digest, and configured producer identity. qxctl is a read-only administrative client; SSIAG is an authenticated producer; neither is a writer.

## Operational Intent

- one process, socket, locked ledger, sequence, and digest chain per TOPS;
- explicit authority, producer, and reader UID/GID identities;
- exact producer `(event_class, operation_id)` permissions;
- classification-scoped queries;
- fsync before a committed receipt;
- restart reconstruction and identical-request replay;
- evidence-preserving recovery of only an incomplete final frame;
- fail-closed startup for complete corruption;
- no retention deletion or automatic rotation in v1.

Enrollment grants no caller authority. Agents cannot append, edit, repair, truncate, export, or bypass qxctl's read-only boundary.
