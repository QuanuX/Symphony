# SCIV engine specification — 0.1.0-dev

## Input and scope

The strict `symphony.snv.sciv.evidence.v1` object contains `cluster {cluster_id,system_id}`, `nodes`, `memberships`, `observations` and an explicit `profile`. `at_unix_ms` and `record_limit` are optional. Node references have an explicit system and optionally a distinct supplied incarnation. Every referenced Node must exist in the collection and use the cluster's system. An incarnation cannot refer to two Nodes in one supplied system; a membership's optional incarnation must equal its Node's supplied incarnation.

Identifiers use exact UTF-8 byte comparison, 1–256 bytes, without ASCII controls or DEL. No slash, colon or bus namespace grammar is inferred. Unknown fields and future protocols fail. Times are exact integer Unix milliseconds in `0..9007199254740991`; fractions and values outside that range fail in both SDK and process. Unknown time remains absent. Schemas describe wire types; native validation also checks references, aggregate bounds, collisions and time ordering.

Each relationship record has a distinct `record_id` and `source {source_id,source_revision,method,method_version}`. Optional `source_digest` is a lowercase tagged SHA256 supplied-artifact reference. Source fields are opaque and do not certify observation authenticity. No automatic source precedence hierarchy is imposed.

Capacity is the aggregate Node, membership and observation count: selectable 512, 1024 or 2048, default 2048. Parser bytes/depth/values, process output and deadline bounds are independent mechanics. Derived fabric count cannot exceed observation count. No hidden clock supplies evaluation time.

## Membership and participation

Membership states are `intended`, `joined` and `removed`, with optional `effective_unix_ms`. Within one source/method/version series, the greatest effective time no later than evaluation applies; equal-time contradictions are retained. Different current source states produce `conflicting`, without trusting a newer competing source automatically. Undated evidence is retained but cannot establish an as-of state. Later dated events never erase history.

Only unconflicted, effective `joined` Nodes enter connectivity evaluation. Intent alone proves no participation or bus connection. Explicit removal, no observation, stale observation and observed disconnect remain separate findings. The result is limited to the supplied population, evidence and evaluation time; it does not certify an exhaustive population of every possible member.

Temporary disconnect/reconnect preserves supplied physical and incarnation references. SCIV creates and ends no incarnations. SNIV owns explicit episode transitions. System references distinguish separately scoped participation. One Node may have an incarnation without satisfying the cluster condition.

## Connectivity profile

`profile {direction,transitive,max_age_ms}` selects `undirected|directed`, Boolean transitivity and a caller-selected freshness duration. A connection identifies one bus and two distinct declared Nodes, state `connected|disconnected|unknown`, coverage `complete|partial`, source, optional `observed_unix_ms` and optional `valid_until_unix_ms`.

Within each source/method/version series, the greatest observation time no later than evaluation applies. Equal-time contradictions and differing independent fresh sources remain conflicting. A newer event in one source series supersedes that series' earlier event. A stale latest event cannot fall back to an earlier connected event from that series.

Freshness requires age at most `max_age_ms` and evaluation before an optional validity end. Validity is `[observed_unix_ms,valid_until_unix_ms)`, with end strictly later than observation. Future, undated and stale evidence cannot prove a current link. No universal freshness duration is chosen by SCIV.

Undirected links support both directions. Directed links support only the stated direction. With transitivity enabled, each eligible Node must reach every other through proven positive edges on the same exact bus. Without transitivity, every required direct pair must be positively evidenced. At least two eligible identified Nodes are required. Partial collection coverage can prove an explicitly observed positive edge; it cannot prove missing or negative edges.

Each fabric is evaluated independently. One bus positively connecting every eligible Node proves the supplied cluster condition. Multiple buses can support the same cluster identity and never generate cluster identities automatically. Paths switching buses do not establish a proof without a separately supported bridge contract; bridge records are outside v1. A disconnected alternate bus does not erase another connected fabric.

Negative findings require fresh, complete observations for every required pair on every represented bus. Disjoint proven groups with complete negative coverage produce `partitioned`; isolated Nodes produce `disconnected`; missing required pairs produce `insufficient`. In a non-transitive profile groups are pairwise complete groups, avoiding implied transitive reachability. Findings cover the represented buses and population, never prove absence of every possible additional transport.

## Results and candidate transitions

Results carry `owner:sciv`, `owner_version:0.1.0-dev`, original-input `source_digest`, sorted distinct `subject_ids`, system/cluster references, as-of time, selected profile, membership/fabric findings and original histories. `cluster_state` is `connected`, `partitioned`, `disconnected`, `unobserved`, `insufficient`, `insufficient_membership`, `not_cluster` or `evaluation_time_unknown`. `connected_cluster` is true only for `connected`. Fabric findings carry source record references and distinct unknown/stale/conflicting/disconnected counts. `cross_fabric_bridge_inferred` is false.

`sciv_transition` accepts `{protocol,evidence,expected_source_digest,change:{kind,record}}`, kind `membership|connection`. It replays original evidence, checks the expected digest, validates the typed addition and returns proposed evidence, complete replay and original/candidate digests. Existing record ID collisions fail unless the same typed record is already present; exact retries return `already_present`. Corrections append attributed events and cannot overwrite prior evidence. A candidate changes no selected head and writes no files. SNV owns retention and protected selection.

Shared exact mechanics own process framing, receipt verification, safe errors and process bounds. Every operation is directly callable by the installed finite process. Qxctl discovery and administration bind that contract without moving its semantics into Go.

All numeric times and limits use the exact nonnegative interoperable JSON integer range `0..9007199254740991`. The directly linked SDK enforces the same range as the bounded process and packaged schemas. This range does not require floating-point rounding or a wider implicit time representation.
