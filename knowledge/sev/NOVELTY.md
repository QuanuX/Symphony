# SEV Novelty Bundles

## Purpose

A Novelty Bundle is an optional, inspectable, offline-capable export projection for one exact evolution case. It binds the source CURRENT, case, findings, dispositions, schemas, and a complete redaction manifest. It is never uploaded by the SEV engine and never becomes canonical merely because it was exported.

## Privacy and Approval

Export is disabled unless explicitly requested. Every included item carries its digest and disclosure class. Redactions are schema-aware and recorded as paths plus reasons; raw credentials, tokens, proofs, provider payloads, local secret material, and excluded STAV fields are prohibited. A fresh caller-neutral SSIAG decision binds the exact post-redaction bundle digest. STAV receives only safe outcome metadata and the digest, never the payload.

Offline export writes to a caller-selected local destination through the durable external-action circuit. Network transfer, publication, and model ingestion are separate operations outside SEV v1.
