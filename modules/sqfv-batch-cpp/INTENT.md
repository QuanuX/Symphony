# SQFV Batch C++ Intent

## Role

`sqfv-batch-cpp` is SQFV's first native, trusted same-process C++26 batch library. It is an independently selectable static package for copying a caller's prepared bytes into an immutable batch, offering that batch to bounded consumer ports, and releasing each reader's lifetime obligation explicitly. A versioned binary frame permits bounded local serialization and fixture exchange.

The module carries exact caller-supplied metadata bindings and sequence context. SQMV owns the meaning and grammar of research-data metadata. SQFV does not choose providers, resolve a latest dataset revision, grant access, retain data durably, or attest that a recipient processed a batch. One port's credit cannot silently consume another port's credit.

The package supplies no resident service, process entry point, qxctl SQV command, IPC, network transport, or device lease. Its same-process scope assumes trusted participating code; exact scope matching is an accident guard, not an authorization boundary. These limitations let the first slice establish memory lifetime and flow behavior without assigning unrelated SQV owners' decisions to SQFV.
