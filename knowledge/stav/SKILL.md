# Symphony TOPS Audit Vector Skill

## Safe Agent Use

Agents may read canonical STAV contracts, query safe ledger projections through qxctl, correlate events by allowlisted identifiers, and propose administrative actions through qxctl.

Agents must never:

- edit, repair, truncate, reorder, or append operational ledger files;
- write runtime events under `knowledge/stav/` or another repository path;
- submit arbitrary events outside an authenticated administrative protocol;
- request excluded security material;
- treat a derived projection as canonical;
- claim non-repudiation from a v1 digest chain.

## Implementation Review Procedure

1. Verify the ten envelope groups and every presence rule.
2. Verify the TOPS identity matches the isolated ledger root.
3. Verify one append authority assigns the sequence and preceding digest.
4. Verify all producer fields use safe allowlists.
5. Test crash recovery, partial writes, concurrent submission, digest verification, redaction, and projection rebuilding.
6. Verify producer connections use kernel-attested local identity mapped to an authorized producer subject.
7. Query only through qxctl or another explicitly ratified read interface.
8. Run the protocol-kernel fixture and digest corpus after any schema, serialization, identifier, or framing change.

## Stop Conditions

The dedicated Go process architecture, append-authority namespace, canonical candidate/event/receipt/query/query-page/verification schemas, strict JCS profile, SHA-256 domains, local frame mechanics, `libraries/stav-protocol-go`, and bounded read-only `qxctl stav query` grammar are owner-ratified. Safe-integer sequences are limited to `2^53-1`. The protocol kernel has no runtime authority.

Stop and obtain owner approval before defining configuration, status, or local request/response schema content; enabling a writer or socket listener; authenticating or authorizing a producer/reader; adding a producer class; choosing storage framing, durability, acknowledgement, recovery, idempotency retention, retention, rotation, or repair behavior; changing the envelope or canonical bytes; adding remote export or Merkle/checkpoint authority; or allowing an agent mutation path.

Go 1.27 work may be experimental before release but must not change the production Go 1.26.5 pin. After general availability, follow the differential migration gate in `libraries/stav-protocol-go/GO_1_27_MIGRATION.md`; a new standard-library API is never permission to change the protocol.
