#pragma once

#include "symphony/knowledge/engine/protocol.hpp"

#include <string>

namespace symphony::knowledge::scv {

// All operations are pure, bounded projections; none selects a durable graph head,
// executes source text, verifies a live provider, or changes a user's choices.
//
// knowledge_interpret: {captures, claims, interpreter_version, selection_policy}
// Policy: {policy_id, max_age_seconds: integer|null, allowed_statement_kinds,
//          partial_capture: "exclude"|"include_qualified"}.
// Claim: {claim_id, subject, predicate, value:{type,value,unit:null|string},
//   scope:{string:string}, statement_kind, evidence:[{capture_digest,quote}],
//   dependencies:[{claim_id,role:"support"|"requires"|"scope"}],
//   valid_from:null|UTC-seconds, valid_until:null|UTC-seconds,
//   alternative_supports?:[{support_id,evidence,dependencies}],
//   scope_dependencies?:[{source_id,capture_digests:[tagged-SHA256]}]}.
// Types: string, reference, boolean, integer, decimal (canonical exact string).
// Kinds: documented_fact, requirement, recommendation, observation, user_assertion,
//        inference, hypothesis. Each evidence quote must occur in its capture.
// This establishes attribution only; semantic interpretations remain proposed.
// The primary support set is conjunctive; alternative sets are independently
// sufficient. Scope dependencies bind the exact searched capture selection.
//
// graph_build: {knowledge:[Knowledge]}; policies must agree. Capture/source
// validators enforce the invoked domain, permitting SCV/family composition.
// graph_query / graph_evaluate: {graph,query_time,claim_ids?,subject?,predicate?}.
// graph_explain: {graph,query_time,claim_id}; includes transitive dependencies.
// graph_diff: {before,after,query_time}; reports changed/affected claims and both
// evaluations at the same explicit time. All UTC strings use STSC whole seconds.
//
// Knowledge and Graph carry protocol, domain, policy, exact retained captures,
// normalized claims, native_nodes/native_edges, limitations and sealed digest.
// Graph additionally carries knowledge_digests and interpretations containing
// {knowledge_digest,domain,interpreter_version}. These are content references,
// not publisher signatures or confirmation of semantic acceptance.
// Query results carry graph_digest, query_time, policy, findings (claim, status,
// reasons, support_sets, evidence_roots, dependent_claim_ids, validity boundary),
// typed conflicts and coverage. "supported" means supported under this bounded
// evidence policy, never empirical truth. Assumptions remain "conditional".
// Inference claims are always conditional: this profile does not execute or
// validate arbitrary inference rules. Evidence origins retain source identities
// while identical body bytes share one root; independence remains unasserted.
// All result protocols use symphony.scv.*.v1: knowledge, graph, query-result,
// evaluate-result, explain-result and diff-result. Findings also expose
// semantic_validation and support_cycle_present. Query adds query_selection and
// optional bounded absence; explain adds claim_id and dependency_closure.
// No confidence score or independent-corroboration count is manufactured.
// Bounds: 16 captures, 128 claims, 8 support sets/claim, 16 premises/set,
// 512 native nodes and 1024 native edges. Overflow fails or explicitly records
// partial native extraction; no provider-wide completeness claim is made.
[[nodiscard]] engine::Json handle_knowledge(
    const engine::Request& request, const std::string& domain);

} // namespace symphony::knowledge::scv
