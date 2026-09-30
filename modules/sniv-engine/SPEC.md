# SNIV engine v1

The exact `identity_validate` operation admits `symphony.sniv.identity-validate-input.v1` and returns `symphony.sniv.identity-validate.v1`. Its strict schema and template are installed owner resources. Domain meaning remains with `knowledge/snv/sniv/SPEC.md`.

## Record and provenance contract

All declared input fields are required, nullable where the schema explicitly permits unknown values. Unknown fields or versions reject. Identifiers are supplied opaque UTF-8 text; byte spelling is preserved without case folding or Unicode normalization. Embedded ASCII controls and invalid UTF-8 reject. No namespace family or qualified shorthand is inferred.

`record_limit` selects 512, 1024 or 2048 total identity and participation records, including a proposed record. Each record carries a positive interoperable integer generation, nullable causal predecessor, attributed sources and exact supplied subject. Source IDs are local to their record; source references cannot target missing sources. A source contains its kind, source reference/revision, nullable whole-second UTC observation time, tagged SHA-256 artifact digest and method identity/version. Supplied digest assertions remain assertions. The result `source_digest` is SHA-256 over `Json::dump()` of the entire owner input: lexicographically sorted object keys, input array order, exact supplied UTF-8 and interoperable integer representation, no insignificant whitespace. It never identifies the physical Node.

Physical records distinguish physical Node, provider-resource tuple, predecessor Node and identity assertions. Provider tuples preserve provider, opaque source-defined scope and native identifier; their establishment is explicit evidence. Labels, serials, firmware UUIDs, infrastructure references, user identity assertions and Habitat references retain their sources. Contradictory active leaf assertions and active provider association collisions produce `conflicting`. A null physical identity or missing causal predecessor produces `insufficient`; no subject is generated. Explicit predecessor/generation supersession establishes history, without time-based last-write-wins precedence. Unsupported source meaning remains outside this v1 record profile.

## Physical transitions

`mode=transition` carries either `kind=physical_change` or `kind=participation`. The caller supplies the candidate record, its new record ID and exact causal predecessor/generation. Reduction performs no persistence. Invalid structure rejects; contradictory or incomplete semantic evidence returns findings and no proposal. Source records and caller intent remain bound in the input digest.

Physical change kinds are unchanged, software-only, remote-only, replacement, separately established provider resource, hardware change, correction, retirement and restoration. Replacement and a different explicitly established provider tuple require a separately supplied successor physical ID and predecessor Node reference, even when configuration is identical. Software, restart and remote attachment alone preserve physical identity. Missing successor is insufficient; reusing the predecessor for a required replacement is conflicting.

Hardware classification uses the caller-selected `profile_id` and exact `version`, with unique component-kind/change rules selecting material or non-material. Unlisted local changes are unresolved. Remote and software changes never trigger local materiality. Any classified material local change requires the supplied new identity. The result preserves the full profile and classification; the profile is no automatic hardware authority. A change reference names supplied evidence; composition checks its correspondence in a retained bundle.

Retirement and restoration are explicit active-to-retired and retired-to-active records. A correction retains physical continuity unless separate replacement/provider/material evidence establishes a new identity. A proposed record cannot reuse an already supplied record ID. Prior evidence stays in the supplied history and is never overwritten by this pure operation.

## Participation lifecycle

A participation record separately identifies physical Node, incarnation, user-selected system scope, bus associations, optional boot identity, lifecycle state, event and effective time. A single physical Node may participate in separate systems concurrently. Competing active incarnations in one physical-Node/system scope are explicit conflict. More than one bus may support one episode; cluster identity is optional and owned by SCIV.

Establishment requires a supplied incarnation and one or more bus references. Temporary disconnect/reconnect and boot change preserve incarnation. End is explicit. Rejoin requires an ended predecessor and a different supplied incarnation. A connection event changes recorded event evidence, without changing bus association or boot identity; observed connectivity remains SCIV's domain. Correction preserves lifecycle state. No bus client, link timeout, synthetic incarnation, provider permission or deployment authority follows.

## Process and library contract

Native reducers use C++26 with extensions off. A deadline is checked before and during bounded work. Foundation framing validates duplicate JSON keys and administrative limits; the library validates owner field types and limits. One bounded process request produces one owner result or a structured error through the shared process envelope. `invalid_input` is safe input rejection; `deadline_exceeded` denotes expired work. The SDK has no network/filesystem side effects. The parent embeds this exact semantic release; a separately installed newer child does not change replay admission.
