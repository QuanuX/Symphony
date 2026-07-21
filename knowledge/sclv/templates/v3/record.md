- record_id: `SCLV-CHG-<stable-id>`
- record_version: `3`
- title: `<short completed-change title>`
- status: `canonical`
- date: `YYYY-MM-DD`
- change_started_at: `YYYY-MM-DDTHH:MM:SSZ`
- change_completed_at: `YYYY-MM-DDTHH:MM:SSZ`
- recorded_at: `YYYY-MM-DDTHH:MM:SSZ`
- recording_disposition: `post_merge|late_recovery`
- recovery_reason: `not_applicable|factual late-recovery explanation`
- change_type: `<registered change type>`
- change_request_state: `present|not_applicable`
- change_request_provider: `<provider namespace|not_applicable>`
- change_request_id: `<opaque provider id|not_applicable>`
- change_request_reference: `<safe reference|not_applicable>`
- change_request_absence_reason: `<not_applicable|factual reason no change request exists>`
- revision_scheme: `<git-sha1|git-sha256|registered scheme>`
- revision_value: `<opaque exact revision>`
- tree_digest: `sha256:<64 lowercase hexadecimal characters>`
- ratification_subject: `<accountable subject>`
- ratification_permission: `<effective permission>`
- ratification_method: `<ratification method>`
- ratification_evidence_reference: `<safe bounded reference>`
- ratification_evidence_digest: `sha256:<64 lowercase hexadecimal characters>`
- affected_surfaces:
  - `<repository-relative path>`
- skvi_references:
  - `<SKVI-indexed repository-relative path>`
- change_summary: |
    <completed-change summary>
- relationship_changes: |
    <relationship consequences or explicit none>
- doctrine_changes: |
    <doctrine consequences or explicit none>
- compatibility_consequences: |
    <compatibility consequences or explicit none>
- publication_consequences: |
    <publication consequences or explicit none>
- projection_consequences: |
    <projection consequences or explicit none>
- evidence:
  - `<safe bounded evidence reference>`
- non_authorizations:
  - `<explicit non-authorization>`
- notes: |
    <bounded notes or explicit none>
