# STAV Append Authority Architecture

## Current Component Boundary

```text
knowledge/stav/                 canonical protocol and gate authority
        |
        v
libraries/stav-protocol-go/     authority-free protocol mechanics
        |
        v
stav-append-authority module    namespace, path resolver, binary lifecycle
        |
        +-- no socket
        +-- no active codec or candidate ingestion
        +-- no producer client
        +-- no ledger
```

The current executable is intentionally not a daemon. It cannot create a serialization domain because the content and durability contracts that make such a domain safe remain gated.

## Future Ratified Shape

After later contracts pass, one process will serve one per-TOPS serialization domain over authenticated local IPC. The authority—not producers or qxctl—will validate candidate metadata, assign trusted order and integrity fields, durably append, and return a safe receipt. This statement records the ratified architecture; it does not activate it.

## Source-Truth Direction

Dependencies flow from canonical knowledge through the shared protocol kernel to runtime implementations. No implementation struct, annotation, or kernel behavior may become the source of protocol truth.

## Isolation

Every future configuration, runtime, socket, state root, sequence, and digest chain is keyed by an immutable canonical lowercase TOPS UUID. Human-readable names never participate in a security path.
