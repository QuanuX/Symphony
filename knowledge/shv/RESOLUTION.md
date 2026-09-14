# SHV exact partition dependency resolution

qxctl `shv partition resolution run|schema|template` routes caller-provided immutable partitions into an existing manifest. The unchanged C++ partition engine owns partition validity, manifest semantics and required-reference status. The Go client independently rederives native correspondence.

Run requires explicit `--prefix`, exact `--version` and `--input`. Input exactly `{manifest,candidates}`: a complete native manifest and an ordered array of 0–64 full native partitions with unique digests. Strict one-MiB JSON admission applies before native invocation. All candidates are validated, including unused ones. A duplicate or malformed candidate rejects the whole request; bounds never silently truncate.

Only declared entries whose partition is null can be filled, and only by exact digest equality. Existing loaded partitions, manifest entry order and required references remain unchanged. Unlisted required references remain unlisted; providing a candidate does not expand inventory. The caller may explicitly construct a broader manifest with the existing native manifest command. Candidate ordering does not change result entry order.

The resolver validates the base manifest, complete candidate inventory and resulting manifest through three pure native manifest_build calls, each independently checked. It pins the partition installation across the operation. No source files, endpoint discovery, implicit provider choice, artifact mutation or catalogue publication occurs.

Output `symphony.qxctl.shv-resolution.v1` contains the complete request digest (canonical input), base_manifest_digest, native manifest, resolved_partition_digests in manifest order, unused_candidate_digests in candidate order, exact installation, dependency_validation=`reference_declarations_only`, catalogue_published=false and self-seal. Already-loaded matches count as unused. Unresolved partitions and required-reference statuses stay visible in the native result. Repeating a resolution preserves the resulting manifest and reports no newly resolved entries.

This verifies reference declarations, not underlying source bytes, hardware identity, cross-partition equivalence or publisher authentication. For evidence replay, callers can use `shv partition from-refresh` or materialization export before explicitly supplying their partitions here. Resolution never promotes a retained proof into fresh verification. A complete declared inventory can still contain missing-subject or unlisted-reference findings; it is not a complete hardware corpus.

Discovery schemas and unanswered templates belong to qxctl and are available without installation flags. Existing receipt-owned partition schemas, native version0.1.0-dev and earlier CLI protocols remain unchanged. Future acquisition tasks with unknown final digests remain separate from missing known immutable partitions.
