# Symphony First-Party Libraries

`libraries/` contains shared implementation code for canonical Symphony contracts. A library is imported at build time and is not an independently installed resident runtime module. A native library may provide a versioned development-package install and receipt so independent executable builds can consume it reproducibly; that package still has no process or operational authority.

Libraries have no resident process, command, service identity, socket, runtime state directory, or operational authority. They do not own protocol truth: the corresponding knowledge vector does. Every library must remain independently testable and versionable, and consumers must preserve their own installability.

Current libraries:

- `stav-protocol-go`: pure-Go implementation of the canonical STAV v1 serialization, validation, digest, identifier, and local-frame contracts.
- `knowledge-vector-engine-cpp`: C++26 authority-free bounded process, digest, path, snapshot, receipt, and static-link foundation for independently installed knowledge-vector executables.
