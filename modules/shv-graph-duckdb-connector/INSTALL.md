# Independent installation

Build this module with CMake 3.25+, C++26, explicit DUCKDB_ROOT and DUCKDB_LIBRARY for the hash-pinned 1.5.5 macOS x86_64 dependency. No network acquisition occurs. Use a new CMAKE_INSTALL_PREFIX; install produces a v2 receipt owning the executable, adjacent DuckDB library, schemas, documentation and licenses. The uninstall-shv-graph-duckdb-connector target validates receipt ownership and retains unrelated caller files. The source build reuses the exact generic adapter 0.1 structural contract. Existing SCV/SHV installations are not changed.

## Inventory release 0.2.0-dev (SHV-19)

`qxctl shv graph store inventory` is a logical read operation under the existing
storage owner, using explicit connector prefix/version, store root and JSON input.
It takes tops_id, namespace, expected_revision (null or exact manifest digest),
cursor (null or revision/after_operation_id), and limit 1..16. The complete scoped
manifest contains sorted operation summaries and snapshot reference counts;
records contains the selected page of full verified status objects. Prepared
intents remain distinct from committed snapshots. Shared snapshots retain every
operation reference. Empty scope means no retained operations, not no hardware.

The owner verifies every retained intent, committed snapshot and projection,
including other scopes, and rejects orphan snapshots and extra projection rows.
Global counts and an opaque global_revision expose capacity changes without
returning other scopes' records. A change anywhere in the store invalidates the
manifest revision and its cursors. No implicit rebase or newest-row selection.
Physical database/WAL sizes are observations, not content identity or usable-space
promises. Existing 128-intent/128-snapshot bounds are unchanged.

The independent Go verifier checks exact input, seals, ordering, reference
accounting, bounds, selected records and continuation. Global database completeness
and off-page evidence are checked by the native owner, not inferred from a digest
alone by qxctl. Unknown graph properties and retired fields remain intact.

Inventory 0.2 can inspect retained 0.1 and 0.2 writer records in the unchanged v1
database schema. Historical receipt identities are preserved; a reader does not
replace a writer. Other data commands still require the exact selected writer;
0.2 refuses committing a 0.1 intent before changing state. Keep the original 0.1
installation for its writes, exports and SHV-18 publication bindings. Publication
engine 0.1 does not silently admit a 0.2 writer. Existing installed packages remain
unchanged. This release adds no transfer execution, deletion, pruning, retention
policy, catalogue selection or database migration. These require later contracts.

## Transfer release 0.3.0-dev (SHV-20)

The current release adds exact-revision transfer planning and qxctl execution/recovery.
It admits historical 0.1/0.2/0.3 writers for inventory while preserving original writer
identity. See SPEC.md for transfer-plan, transfer and transfer-status, private target
requirements and explicit publication under publication engine 0.2.0-dev.

## Mechanical interface release 0.4.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.

Store 0.4 can inventory retained writers 0.1/0.2/0.3/0.4; other data operations remain bound to the exact writer. Transfer planning binds its selected 0.3 or 0.4 reader. Old readers do not admit new 0.4 writers, and old publication admission remains unchanged. Embedded structural adapter stays exactly 0.1. No database migration or graph database default is introduced.
