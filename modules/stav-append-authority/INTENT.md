# STAV Append Authority Intent

## Purpose

Provide the independently installable Go implementation boundary for the one serialized STAV append authority assigned to each immutable TOPS identity.

## Authority Boundary

`knowledge/stav/` is the sole source of STAV protocol and schema truth. This module implements ratified contracts and never owns the event model, producer policy, query policy, or schema definitions. qxctl remains the administrative and query interface; it is not a ledger writer.

## Current Increment

The owner has ratified the module, executable, socket path, canonical semantic/read schemas, protocol-kernel namespace, and bounded read-only qxctl grammar. This increment therefore supplies:

- the `symphony-stav-append-authority` executable namespace;
- reversible user/system binary installation;
- pure path resolution for the ratified per-TOPS namespace;
- shared protocol-kernel TOPS identity validation;
- no configuration, state, runtime directory, socket, listener, event, receipt, or ledger creation.

## Safety Intent

Until the remaining STAV contracts are ratified, every operational command fails closed. Agents cannot use this executable to append, edit, repair, truncate, export, or otherwise mutate a ledger.
