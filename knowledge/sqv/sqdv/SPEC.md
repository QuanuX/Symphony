# Symphony Quantitative Delivery Vector Specification

## View and Recipient Contract

An admitted delivery identifies the exact dataset and revision, selected view and transformation release where applicable, representation, coverage and known gaps, recipient and access scope, delivery mode, finite resource allowance, and receiving interface version. Snapshot, subscription, selected projection, retained catch-up, and external handoff have distinct contracts. SQDV may request an SQTV conversion, an SQPV retained read, and an SQFV transfer; it does not silently duplicate those owners' work.

The view states whether its evidence is preview, retained under a named guarantee, or committed at a particular destination boundary. A preview may run ahead of retention only if its contract declares that possibility. Local queue admission, transport receipt, consumer processing, destination acceptance, destination commit, queryability, and physical buffer release remain distinct events where supported. A missing receipt or ambiguous retry is reported as unresolved, not as successful complete delivery.

## Continuity and Acknowledgement

Consumer resume is bound to the view and dataset revision, partition, producer generation or continuity evidence, transformation configuration, and exact position. SQDV states its retry, duplicate, gap, expiry, and backfill behavior under the selected receiving contract. Switching from live memory to retained data must preserve the declared sequence and make duplicates, omissions, corrections, and unavailable ranges visible. A timestamp or offset alone is not universally sufficient.

SQDV interprets acknowledgements only to the level proven by the receiving interface. An SBV input still follows SBV admission and result authority. SIV and web recipients receive bounded selected data or references under their actual contracts. A future external integration owns its destination account, authentication, ingestion call, remote receipt, and commit semantics; SQDV prepares the selected export identity and view for that owner without claiming the remote outcome.

## Rights and Failure

An SQMV rights classification describes source evidence; it is not a delivery grant. Before a recipient receives data, the selected access owner and source terms must permit that scope and purpose. Revocation stops new delivery under its admitted rules while prior bytes and live borrow obligations remain accounted for. A slow or disconnected recipient reaches a finite allowance and follows its declared stop, suspend, catch-up, or explicit-loss policy without silently consuming another recipient's budget.

## First Native Delivery Contract

`modules/sqdv-delivery-cpp/SPEC.md` owns the first `0.1.0-dev` C++26 full-batch interface for trusted local callers. Disposable live and retained-before-delivery profiles select exact immutable metadata and one recipient/interface/view/partition/generation. Separate dispatch and processing positions, opaque session-specific acknowledgement tickets, finite processing windows and SQFV lease credits preserve distinct obligations. Retained catch-up selects exactly the next sequence and uses a matching live retained batch when available. Its SQPV wrapper derives opaque retention evidence only from actual owned-store operations.

The caller persists and selects its processing checkpoint. The checkpoint binds the exact view and selected local store identity; it grants no access and proves no external processing or remote commit. The store guarantee remains the exact SQPV profile. Asynchronous retention workers, projections, transformations, recipient adapters, authorization and remote delivery remain later contracts. This library allocates no `sqdv:` identity family and sends no routine payload through qxctl or SKV.

## Non-Authorization Statement

SQDV does not authorize a recipient, provider licence, external export, strategy result, remote commit, universal exactly-once guarantee, or disclosure of private data.
