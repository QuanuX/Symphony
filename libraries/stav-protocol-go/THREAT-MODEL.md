# STAV Protocol Kernel Threat Model

## Defends Against

- parser differentials from duplicate or case-folded keys;
- invalid UTF-8 replacement and unpaired surrogate normalization;
- cross-language integer precision loss;
- non-canonical bytes producing inconsistent digests;
- unknown-field and registry-value smuggling;
- oversized local frames before allocation;
- digest cross-protocol substitution.

## Does Not Defend Against

- a compromised producer or append authority;
- unauthorized socket access;
- ledger rollback, deletion, persistence failure, or repair error;
- disclosure through an incorrectly designed projection policy;
- non-repudiation or historical writer authentication.

Those controls belong to later authenticated transport, authorization, durability, and checkpoint contracts.
