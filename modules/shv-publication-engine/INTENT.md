# INTENT

Publish caller-selected catalogue heads through qxctl with original evidence replay and a distinct permission.

See PUBLICATION.md for the publication contract and limits.

## Explicit composition release 0.4.0-dev

Publication 0.4.0-dev explicitly embeds partition 0.4, admits selected partition 0.2/0.3/0.4, source 0.1/0.2, kernel 0.1/0.2/0.3/0.4 and storage writers 0.1/0.2/0.3/0.4. A partition selected as 0.2 or 0.3 cannot carry source 0.2 or kernel 0.4 dependencies. Historical publishers retain their prior admissions. Retained state transitions replay against each original publisher; aggregate history validation admits the supported union without changing those owners.

Use a fresh install prefix and explicit qxctl version selection. Version 0.3 declarations remain checked against their original compiled digest. No command or default changes; metadata is not semantic authorization and future versions are not automatically admitted.
