# STAV Protocol Kernel for Go

This library is Symphony's first-party, pure-Go implementation of the STAV v1 protocol contracts in `knowledge/stav/`. It centralizes strict JSON/JCS behavior so qxctl, SSIAG, and the operational append authority cannot drift into different parsers or digest rules.

The kernel has no executable install surface and no operational authority. It contains no listener, writer, credential access, policy engine, supervisor, or ledger implementation. The current source release is the independently consumable Go module `github.com/QuanuX/Symphony/libraries/stav-protocol-go@v0.2.0`.

Production remains pinned to Go 1.26.5. `GO_1_27_MIGRATION.md` defines the confirmed-release migration gate for the intended Go 1.27 target.
