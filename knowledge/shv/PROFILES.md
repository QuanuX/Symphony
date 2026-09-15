# SHV caller profiles and portable universes — v1

The independently installable C++ `shv-profile-engine` 0.1.0-dev owns caller class
profiles, mapping declaration diagnostics and portable hardware-universe recipes.
It uses the compiled exact SHV kernel 0.3.0-dev source reader; qxctl independently
validates every result and replays original source bytes for a binding. This is not
an invocation of a separately installed kernel. Upgrading that reader requires an
explicit reviewed engine revision. Existing kernel, coverage, source activation,
publication and graph adapter contracts retain their authority.

## Caller-defined comparison vocabulary

`profile_compile` seals a definition with `id`, `revision`, `hardware_class`,
`metrics` and an open `extensions` object. Each metric declares a unique
`predicate`, exact `value_type`, exact `qualifier`, caller-chosen `required` boolean,
a description and open extensions. Types are the kernel's `string`, `integer`,
`date`, `tokens`, `quarter_20yy` and `table_rows`. Qualifiers encode the caller's
unit, namespace or condition. Descriptions never establish facts or trigger unit
conversion. There is no platform-required class census or predicate list.

Example fixture profiles cover GPU capacity, memory capacity, storage capacity and
networking-card link rate. Those examples are synthetic conformance cases, not
vendor specifications or recommended physical interchangeability tests. Custom
classes and predicates use the same contract. A caller can add fields, change
requiredness, retain a retired field as optional, or omit it from a new revision.
Earlier sealed profiles and mappings remain replayable; this engine has no mutable
profile registry and never deletes data. A revised definition changes the digest;
revision labels are caller metadata and are not globally unique version authority.

## Mapping diagnosis

`mapping_diagnose` takes a sealed profile and full existing kernel subject mapping
rows. Matching classes yield one finding per selected metric: `matched`,
`mismatch`, `unmapped_required` or `unmapped_optional`. Mismatch differences name
`value_type` and/or `qualifier`; no coercion is inferred. Required missing fields
or any mapped mismatch yield `incomplete`. Optional missing fields remain
conformant. Other classes yield `not_applicable`; additional predicates are listed
in sorted `extension_predicates`, without becoming errors. Subject order and
profile metric order are preserved. Counts cover the returned subjects exactly.

The explicit evidence scope is `mapping_declarations_only`. This operation checks
mapping shape and declared comparison vocabulary. It does not parse source HTML,
validate selector applicability, validate a PDF extraction object, establish
source authenticity, or report per-field extraction failures. A conformant mapping
can still fail source replay. Full parser diagnostics remain separate work.

## Portable recipes and local binding

`universe_build` seals `id`, `revision`, exact `kernel_version`, an existing sealed
coverage profile, selected class profiles, source manifests, subject mappings,
locator references and extensions. Manifests preserve source ID, safe relative
path, byte count, exact digest and format. PDF mappings replace `decoder_root` with
a caller-selected `decoder_binding`; their original extraction artifact remains
embedded and is validated by the exact reader on binding. Recipes omit control
fields for local source and decoder roots. Open extensions are caller metadata;
callers must remove sensitive information before choosing to share them.

Each optional locator has `source_id`, opaque `uri` and nullable
`upstream_revision`. Local/custom schemes are allowed. A caller may create a new
recipe revision pointing to a moved source while retaining its byte identity, or
explicitly select new bytes and mappings. Locator changes affect recipe identity.
They do not fetch content, authenticate a publisher, grant an agent permission or
activate a source. Actual active-source changes remain under qxctl source lifecycle
and SSIAG authority. A recipe can be entirely offline, with no locators.

`universe_bind` takes the sealed recipe plus a clean absolute `source_root` and an
exact map of decoder binding IDs to clean absolute roots. Missing or unused decoder
bindings are errors. The compiled kernel replays the selected local source bytes
and exact PDF extraction where applicable, creates the existing catalogue, derives
coverage from observed introduction intervals and reports declaration conformance
for every selected profile. Classes without a selected profile are listed without
rejection. The result exposes `catalogue_input`, `catalogue`, `coverage`,
`conformance`, `unprofiled_classes`, the compiled reader identity and
`canonical_apply_enabled:false`.

The coverage result explains selection; it does not silently trim the catalogue.
The existing 2018-onward coverage default is retained and entirely replaceable by
caller filters, dates or `all`. No historical backfill is needed to exercise this
workflow. Source-root relocation with identical bytes preserves recipe identity;
local binding identity changes. Exact retained PDF extraction may additionally bind
a decoder version or receipt; substituting a decoder is not a portable-path edit.

A bound catalogue is compatible with exact kernel catalogue replay, evaluation and
graph projection. Generic graph adapters retain structural exchange ownership;
DuckDB remains the SQL default and no graph database default is selected. A bind
is not catalogue publication, durable storage, source activation, an SKV knowledge
promotion, a benchmark or a claim about whole-system compatibility.

## qxctl ownership

All commands require the profile engine's explicit `--prefix` and `--version`.
Data operations require `--input`; all support structured `--json` output.

| Surface | Owner operation |
| --- | --- |
| `qxctl shv profile inspect` | `inspect` |
| `qxctl shv profile compile` | `profile_compile` |
| `qxctl shv mapping diagnose` | `mapping_diagnose` |
| `qxctl shv universe build` | `universe_build` |
| `qxctl shv universe bind` | `universe_bind` |
| `qxctl shv profile schema` | exact receipt-owned schema discovery |
| `qxctl shv profile template --operation …` | exact receipt-owned unanswered template |

No competing aliases are introduced for existing catalogue or coverage surfaces.
Every native operation is read-only, idempotent and has an administration descriptor.
Schema checks describe shape; the C++ owner and independent qxctl correspondence
checks enforce seals, identities, semantics and evidence. Receipt validation before
and after invocation prevents installation substitution within that boundary.

## Bounds and failures

At most 16 metrics per profile, 16 selected profiles, 32 mapped subjects, 16 fields
per subject, 256 total mapped fields, 8 sources and 8 locators. Sources are at most
1 MiB each and 4 MiB in aggregate. IDs use `[A-Za-z0-9._-]{1,128}`; paths are at most
4096 bytes, qualifiers 256 and descriptions 4096. Existing process request, JSON,
response and deadline bounds still apply, so not every combination at its individual
maximum fits one request. Unknown control fields, duplicate identities, unsafe
paths, stale seals, missing selected evidence and altered bytes fail closed.
Profile text fields exclude ASCII control characters. Extension objects remain open
within the bounded JSON envelope; floating-point numbers are prohibited by the
existing interoperable process protocol.
