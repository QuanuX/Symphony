# Symphony First-Party Libraries

`libraries/` contains shared implementation code for canonical Symphony contracts. A library is imported at build time and is not an independently installed runtime module.

Libraries have no resident process, command, service identity, socket, state directory, installer, uninstaller, or operational authority. They do not own protocol truth: the corresponding knowledge vector does. Every library must remain independently testable and versionable, and consumers must preserve their own installability.

Current library:

- `stav-protocol-go`: pure-Go implementation of the canonical STAV v1 serialization, validation, digest, identifier, and local-frame contracts.
