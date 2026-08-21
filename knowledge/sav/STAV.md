# SAV STAV Integration

## Registered Named Version Lifecycle

The canonical machine-readable vocabulary is `knowledge/stav/registries/v1/accordare.json`. SAV owns the meanings below; STAV owns their envelope and serialized record.

| Lifecycle action | Event class | Operation and intent | Target kind |
|---|---|---|---|
| prepare an immutable proposal | `symphony.sav.named-version.lifecycle` | `symphony.sav.named-version.prepare` | `symphony.sav.named-version-proposal` |
| seal one exact prepared object | `symphony.sav.named-version.lifecycle` | `symphony.sav.named-version.seal` | `symphony.sav.named-version` |
| change one registry alias | `symphony.sav.named-version.lifecycle` | `symphony.sav.named-version.alias` | `symphony.sav.named-version-registry` |
| recover one unique registry chain | `symphony.sav.named-version.lifecycle` | `symphony.sav.named-version.recover` | `symphony.sav.named-version-registry` |

Each tuple admits only `failed`, `succeeded`, and `unavailable`, mapped to the same operation prefix plus the outcome. `denied` remains an SSIAG policy-decision event and is not duplicated as a SAV lifecycle result. `allowed` is not a persistence outcome.

`prepare` records immutable proposal creation, not composition acceptance. `seal` records persistence of an already validated object, not SODV publication. `alias` changes selection metadata without changing object identity. `recover` records a forward repair selected from one unique digest-linked chain; it never rewrites history.

The vocabulary is contract-only in this slice. The SAV engine, coordinator, and qxctl must continue reporting `stav_append_enabled: false` until a separately authenticated producer, exact grant, receipt validation, recovery behavior, and negative tests are ratified and implemented.
