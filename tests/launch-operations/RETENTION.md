# Retained evidence: capacity, export, backup and recovery

## Bounds and capacity refusal

Limits belong to the selected owner/version, not to an aggregate free-disk estimate. Verify the exact descriptor and schema before choosing an input size.

| Store | Implemented bound | Evidence |
| --- | --- | --- |
| SNV private journal | 16 MiB; 512 operations/bundles | `snvstate/store.go`, `TestSNVCapacityPreservesAcknowledgedJournal`, `TestSNVByteCapacityPreservesAcknowledgedHead` |
| SHV source and publication journals | 16 MiB each; 128 retained attempts each; native publication additionally bounds heads to 32 and members to 8 | `shvstate/store.go`, `shvpublicationstate/store.go`, `knowledge/shv/PUBLICATION.md` |
| SCV source journal | 16 MiB; 128 retained operations | `scvstate/store.go`, `TestSourceCapacityPreservesAcknowledgedJournal` |
| SCV corpus | 128 members per job, 1 MiB per artifact, 2 MiB serialized job | `scvcorpus/store.go`; these are not a global store quota |
| Portable SNV bundle | 256 KiB canonical bundle, 64 KiB per owner artifact, 16,384-byte export chunks | `modules/snv-engine/SPEC.md` |

At full capacity, save the refusal and the prior head/journal digest. Do not delete history, enlarge constants, overwrite a same-version receipt, or silently reset the store to make a command succeed. The regression suite verifies refusal preserves acknowledged bytes. Select a separately named fresh store only as an explicit operator decision; it does not inherit the old lineage or become an automatic migration. No automatic pruning or garbage collection is promised.

## Portable export versus backup

A graph export is structural evidence, not a backup of SSIAG/STAV, source captures, private intents or selected heads. SNV's export manifest and all ordered raw-byte chunks form a bundle; an inventory, projection or summary is insufficient. Capture the current head first, retain its digest, and bind every later page/chunk to that same head. Refuse mixed revisions, missing/reordered/duplicate chunks and changed bytes. Export inspection does not select a view or publish a catalogue.

Use exact command discovery, not guessed defaults:

```sh
qxctl snv inspect --help
qxctl snv state status --help
qxctl shv graph store export --help
qxctl shv graph store transfer-plan --help
qxctl shv graph store transfer --help
qxctl scv graph store export --help
```

SNV selected-view export uses `snv inspect --mode export_manifest` followed by `--mode export_chunk --offset <index> --expected-head-digest <captured-head>` with the same explicit installation, TOPS, view and state root. Save every returned JSON response privately. Import requires a complete export-record set through evidence prepare/commit; it grants no selected-view authority. The native owner validates its writer and reducers independently.

For SCV/SHV graph copies, use owner export/transfer with exact source and destination installations, root, namespace, snapshot and operation ID. A transferred graph alone does not restore source authority or catalogue publication. Transfer status and journals are evidence; never relabel a structural transfer as full store recovery.

## Quiescent backup procedure

1. Record exact qxctl/owner/dependency executable and receipt digests, prefixes and versions, TOPS/view/catalogue identities, current heads and pending operation IDs. Inventory caller-owned source documents, decoder identities and graph stores referenced by those intents. Preserve original bytes and licenses; do not assume the package owns them.
2. Stop all cooperating writers and close DuckDB connections using their normal shutdown paths. Do not copy an actively changing database or merely copy a database file while its WAL is live. If writers cannot be quiesced, use a separately qualified owner export or filesystem snapshot procedure; this increment supplies no live-backup tool.
3. Copy the complete selected private state roots and their referenced immutable evidence into a new owner-private backup destination. Include recovery journals and relevant database/WAL state as one quiescent set. Never follow symlinks, merge into an existing backup, change ownership, or include credentials in public CI artifacts. Use the host's established backup tool with metadata preservation and restrictive permissions.
4. Inventory relative paths, regular-file sizes/digests, ownership and permission modes before and after copying. Require identical source inventories across the copy and matching backup bytes. Record successful flush/completion from the backup tool. A hash manifest detects drift; it does not authenticate the publisher or establish power-loss durability.
5. Keep the backup immutable by procedure and restore into a new private root. Retain a second independent copy according to the operator's retention policy. The test-only `retainedtest` helper is not an installed backup program and must not be used against live stores.

## Restore, restart and upgrade

Restore the same owner UID and exact original writer installations first. Do not rewrite receipt prefixes, retained installation bindings, writer IDs, source revision IDs or history digests to fit a new location. If the original required installation cannot be recreated, retain the backup and stop; a newer binary at another prefix is not an automatic substitute.

Before mutation, inspect the restored evidence/head using the original exact owner. Compare every recorded digest and operation disposition. A pending operation may already have taken effect; recover by its original operation ID with present source evidence and fresh authenticated authorization. Never retry an uncertain mutation under a new ID, and never interpret a stored allow decision as current permission. Committed retries must not rewind a newer head.

Keep the source backup and user evidence unchanged. Test recovery in the isolated restored copy, then verify old committed revisions still replay. The new SCV corpus, SHV source/publication and SNV restore regressions cover this boundary with synthetic evidence. They do not transfer authority between accounts or certify a host crash.

Upgrade by installing the candidate beside the original, verifying its exact admission of every retained writer/dependency, and replaying a restored copy before switching any selected binding. SHV publication 0.4 explicitly admits its documented source/kernel/partition/store versions; earlier publishers retain their narrower rules. SNV 0.1 admits its initial writer only. See `knowledge/shv/PUBLICATION.md` and the selected module specification. Migration metadata is not a reader implementation. Preserve old installations until all required retained operations are reconciled. Package removal does not authorize deleting user stores.
