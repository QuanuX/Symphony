# SHV PDF adapter intent

Turn the selected AMD PDF table into a replayable derived artifact while retaining the original document and its OPN namespace. This SHV consumer owns one table profile; SODV publication and SMCV Markdown conversion retain their existing responsibilities. It does not assign general document-decoder ownership.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.
