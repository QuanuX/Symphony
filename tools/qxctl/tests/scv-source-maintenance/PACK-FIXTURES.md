# Portable package conformance specimen

`verify_pack.cpp` exercises the already implemented `provider_pack_prepare` and `provider_pack_evaluate` operations against exact installed `schv-do` owner `0.10.0-dev`. It makes five focused qxctl calls and performs no network acquisition. It changes no installation, selected source head, grant, runtime implementation or provider API version.

## Inputs and provenance

All paths are explicit; the runner has no workspace, installed-prefix, user-home or evidence-directory default. `--qxctl` selects the CLI executable, `--prefix` selects the exact receipt-owned installation, `--old-package` selects a directory containing the two original package files, `--fresh-capture` selects the retained candidate capture, and `--out` must name a new directory. Domain/version are deliberately fixed to this concrete acceptance specimen, not inferred from a latest alias.

The copied offline assets have source records and file hashes in `fixtures/PROVENANCE.json`:

- `fixtures/old-package/pack.json`: historical sealed `symphony-retained-do-knowledge`, authored version `2026-09-11.1`.
- `fixtures/old-package/prepare-input.json`: its exact draft and detached fixtures. The runner reads the original limits/Labs captures, policy, profile objects and old product-only fixture from these retained values. Duplicate historical references must agree exactly.
- `fixtures/fresh-capture.json`: the one real complete generation-2 candidate capture obtained through qxctl on September 13. Its direct canonical Markdown URI differs from the old configured locator; its 16,774-byte body is identical to the old body. It is candidate evidence, not proof of protected adoption.

The source identity is `do-app-platform-limits`, provider `do`, family `schv`, locator `docs`. The old source already recorded the real DigitalOcean `index.md` → `index.html.md` redirect. This fixture does not claim a newly discovered vendor migration, a pre-move body or a known migration date.

## Authored expectations

The runner independently declares the three exact typed limits facts after source reading: App Platform Linux container image architecture `amd64`; host-local filesystem persistence `false`; development-Valkey availability `false`. Managed Valkey and other persistent storage offerings remain separately scoped alternatives. It also retains the opposing historical Labs development-Valkey statement `true`. Expected values are not read from an extractor profile, prior expected-claims table or engine output.

The new package keeps its stable ID and selects authored version `2026-09-13.1`. It changes only the desired limits source declaration and related fixture inputs/provenance. Both sealed interpretation profiles remain exactly version 1. Each new limits fixture uses the retained generation-2 capture; the native owner computes its fresh fixture input digest. Original source-generation-1 inputs are not relabeled.

The three positive detached cases select all evidence, product limits only, and the historical Labs guide only. Expected outcome: three passed, zero failed, zero not run, with four matched production extractions. This preserves the contradictory source statements rather than choosing a provider truth on the caller's behalf.

The original policy admits documented facts, requirements and recommendations; excludes partial captures; and retains its 86,400-second maximum age. Conformance compares authored mappings, not graph eligibility at a current query time. The historical Labs observation is older than that window at the new acquisition time. Passing its fixture does not refresh the observation, establish present vendor wording, resolve the contradiction or prove deployed behavior.

## Two negative checks

First, substitute the exact old product-only fixture under the new sealed fixture manifest. The CLI must reject it using the stable published `engine_rejected` / `pack.invalid` error envelope. The runner does not depend on private native diagnostic prose.

Second, prepare a separately labeled diagnostic pack (`2026-09-13.1-negative-old-source`) that honestly binds the old generation-1 fixture while retaining the new desired source declaration and unchanged positive expectations. Its fixture must fail with no actual claims and three unresolved `source_declaration_differs` extractions. This tests the declaration mismatch independently of the first fixture-identity rejection. The failed fixture remains evidence; preparation does not silently replace the author's expectations.

## Running the focused check

Build the native runner from the repository root, then provide the chosen executable and installation:

```sh
cmake -S tools/qxctl/tests/scv-source-maintenance -B /absolute/maintenance-build -DCMAKE_BUILD_TYPE=Release
cmake --build /absolute/maintenance-build --target scv-verify-pack
/absolute/maintenance-build/scv-verify-pack \
  --qxctl /absolute/path/to/qxctl \
  --prefix /absolute/path/to/exact-installed-prefix \
  --old-package tools/qxctl/tests/scv-source-maintenance/fixtures/old-package \
  --fresh-capture tools/qxctl/tests/scv-source-maintenance/fixtures/fresh-capture.json \
  --out /absolute/path/to/new-pack-evidence
```

The output preserves `COMMANDS.json`, `SUMMARY.json`, independent `SEMANTIC_EXPECTATIONS.json`, full command inputs/outputs/error envelopes, and before/after hashes of the three input files. `package/` contains reusable prepare input, sealed pack, evaluation selection, direct evaluation input/result, profiles, captures and detached fixture inputs. The runner checks every command input against the existing 1 MiB input bound and preserves native deadline behavior. It never trims evidence to fit.

The expected sealed pack digest for these exact copied inputs is `sha256:34b7d1fc739f78418d7b8ab0bc465275788222d3dd60ffe91aa7db3c6077153b`; the expected evaluation digest is `sha256:609e215ba117a3e45921b72b6b7018429f43718ed33bc0b8c131d94b16af6995`. These documented identities are reproduction oracles, not hardcoded substitutes for actual native invocation and conformance assertions.
