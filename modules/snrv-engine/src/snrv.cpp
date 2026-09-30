#include "validation.hpp"
#include <map>
#include <symphony/snv/snrv.hpp>
namespace symphony::snv::snrv {
namespace {
using namespace owner_validation;
std::uint64_t quantity(const Json &q) {
  keys(q, {"value", "unit"});
  auto s = text(q, "value", 20);
  choice(q.at("unit"), {"bytes", "count", "hertz", "bits_per_second"});
  require((s == "0" || s.front() != '0') &&
              std::all_of(s.begin(), s.end(),
                          [](char c) { return c >= '0' && c <= '9'; }),
          "Quantity must be canonical unsigned decimal");
  std::uint64_t value = 0;
  for (char c : s) {
    auto digit = static_cast<std::uint64_t>(c - '0');
    require(value <= (std::numeric_limits<std::uint64_t>::max() - digit) / 10,
            "Quantity exceeds uint64");
    value = value * 10 + digit;
  }
  return value;
}
void nullable_quantity(const Json &q) {
  if (!q.is_null())
    (void)quantity(q);
}
void resource(const Json &r, const std::set<std::string> &refs) {
  keys(r, {"resource_id", "component_kind", "instance_ref", "instance_scope",
           "model_ref", "presence", "capacity_kind", "capacity",
           "available_capacity", "operational_state", "source_ids"});
  (void)text(r, "resource_id");
  (void)text(r, "component_kind");
  nullable_text(r.at("instance_ref"));
  choice(r.at("instance_scope"), {"physical", "exposed", "unknown"});
  nullable_text(r.at("model_ref"));
  choice(r.at("presence"), {"present", "absent", "unknown"});
  choice(r.at("capacity_kind"), {"physical", "exposed", "unknown"});
  choice(r.at("operational_state"), {"available", "unavailable", "unknown"});
  nullable_quantity(r.at("capacity"));
  nullable_quantity(r.at("available_capacity"));
  references(r.at("source_ids"), refs);
  if (r.at("presence") == "absent")
    require(r.at("capacity").is_null() &&
                r.at("available_capacity").is_null() &&
                r.at("operational_state") != "available",
            "Absent resource cannot claim available capacity");
  if (!r.at("capacity").is_null() && !r.at("available_capacity").is_null()) {
    require(r.at("capacity").at("unit") ==
                r.at("available_capacity").at("unit"),
            "Capacity units differ");
    require(quantity(r.at("available_capacity")) <= quantity(r.at("capacity")),
            "Available capacity exceeds exposed/present capacity");
  }
}
void attachment(const Json &r, const std::set<std::string> &refs) {
  keys(r, {"attachment_id", "resource_kind", "allocation_ref",
           "remote_subject_ref", "endpoint_ref", "capacity", "state",
           "source_ids"});
  (void)text(r, "attachment_id");
  (void)text(r, "resource_kind");
  (void)text(r, "allocation_ref", 1024);
  nullable_text(r.at("remote_subject_ref"));
  nullable_text(r.at("endpoint_ref"));
  nullable_quantity(r.at("capacity"));
  choice(r.at("state"), {"attached", "detached", "unknown"});
  references(r.at("source_ids"), refs);
}
void inventory(const Json &j, std::size_t maximum) {
  keys(j, {"record_id", "generation", "predecessor_record_id",
           "physical_node_id", "status", "coverage", "observed_at", "sources",
           "local_resources", "remote_attachments", "offering_refs",
           "model_refs", "identity_rebind"});
  (void)text(j, "record_id");
  require(integer(j.at("generation")) > 0, "Generation must be positive");
  nullable_text(j.at("predecessor_record_id"));
  (void)text(j, "physical_node_id");
  if (!j.at("identity_rebind").is_null()) {
    const auto &binding = j.at("identity_rebind");
    keys(binding,
         {"sniv_record_ref", "predecessor_node_id", "successor_node_id"});
    (void)text(binding, "sniv_record_ref");
    (void)text(binding, "predecessor_node_id");
    (void)text(binding, "successor_node_id");
    require(binding.at("successor_node_id") == j.at("physical_node_id") &&
                binding.at("predecessor_node_id") !=
                    binding.at("successor_node_id"),
            "Inventory identity rebind is inconsistent");
  }
  choice(j.at("status"), {"active", "retired"});
  choice(j.at("coverage"), {"complete", "partial", "unknown"});
  timestamp(j.at("observed_at"));
  auto refs = sources(j.at("sources"));
  require(!refs.empty(), "Inventory requires attributed evidence");
  std::set<std::string> resource_ids, attachment_ids;
  for (const auto &r : array(j.at("local_resources"), maximum)) {
    resource(r, refs);
    require(resource_ids.insert(text(r, "resource_id")).second,
            "Duplicate local resource identifier");
  }
  for (const auto &r : array(j.at("remote_attachments"), maximum)) {
    attachment(r, refs);
    require(attachment_ids.insert(text(r, "attachment_id")).second,
            "Duplicate remote attachment identifier");
  }
  require(j.at("local_resources").size() + j.at("remote_attachments").size() <=
              maximum,
          "Combined inventory capacity exceeded");
  unique_strings(j.at("offering_refs"), 128);
  unique_strings(j.at("model_refs"), 128);
}
Json changes(const Json &old, const Json &next) {
  std::map<std::string, const Json *> before, after;
  for (const auto &r : old.at("local_resources"))
    before.emplace(text(r, "resource_id"), &r);
  for (const auto &r : next.at("local_resources"))
    after.emplace(text(r, "resource_id"), &r);
  std::set<std::string> keys;
  for (const auto &[id, r] : before) {
    (void)r;
    keys.insert(id);
  }
  for (const auto &[id, r] : after) {
    (void)r;
    keys.insert(id);
  }
  Json out = Json::array();
  for (const auto &id : keys) {
    auto b = before.find(id), a = after.find(id);
    std::string kind = "unknown", component;
    bool physical_change = false;
    if (b == before.end()) {
      component = text(*a->second, "component_kind");
      if (old.at("coverage") == "complete" &&
          a->second->at("presence") == "present") {
        kind = "added";
        physical_change = a->second->at("instance_scope") == "physical";
      }
    } else if (a == after.end()) {
      component = text(*b->second, "component_kind");
      if (next.at("coverage") == "complete" &&
          b->second->at("presence") == "present") {
        kind = "removed";
        physical_change = b->second->at("instance_scope") == "physical";
      }
    } else {
      const auto &br = *b->second;
      const auto &ar = *a->second;
      component = text(ar, "component_kind");
      if (ar.at("presence") == "unknown" || br.at("presence") == "unknown")
        kind = "unknown";
      else if (br.at("presence") == "present" &&
               ar.at("presence") == "absent") {
        kind = "removed";
        physical_change = br.at("instance_scope") == "physical";
      } else if (br.at("presence") == "absent" &&
                 ar.at("presence") == "present") {
        kind = "added";
        physical_change = ar.at("instance_scope") == "physical";
      } else if (br.at("component_kind") != ar.at("component_kind") ||
                 (!br.at("instance_ref").is_null() &&
                  !ar.at("instance_ref").is_null() &&
                  br.at("instance_ref") != ar.at("instance_ref"))) {
        kind = "replaced";
        physical_change = ar.at("instance_scope") == "physical" &&
                          br.at("instance_scope") == "physical";
      } else if (!br.at("capacity").is_null() && !ar.at("capacity").is_null() &&
                 br.at("capacity") != ar.at("capacity")) {
        if (br.at("capacity").at("unit") != ar.at("capacity").at("unit") ||
            br.at("capacity_kind") != ar.at("capacity_kind"))
          kind = "unknown";
        else {
          kind = "capacity_changed";
          physical_change = br.at("capacity_kind") == "physical";
        }
      } else if (br.at("available_capacity") != ar.at("available_capacity") ||
                 br.at("operational_state") != ar.at("operational_state")) {
        kind = "availability_changed";
      } else if (br.at("presence") == "absent")
        kind = "unchanged";
      else if (br.at("instance_ref").is_null() ||
               ar.at("instance_ref").is_null() || br.at("capacity").is_null() ||
               ar.at("capacity").is_null())
        kind = "unknown";
      else
        kind = "unchanged";
    }
    out.push_back({{"resource_id", id},
                   {"component_kind", component},
                   {"locality", "local"},
                   {"change", kind},
                   {"physical_change_evidence", physical_change},
                   {"evidence_refs",
                    Json::array({old.at("record_id"), next.at("record_id")})}});
  }
  std::map<std::string, Json> remote_before, remote_after;
  for (const auto &r : old.at("remote_attachments"))
    remote_before.emplace(text(r, "attachment_id"), r);
  for (const auto &r : next.at("remote_attachments"))
    remote_after.emplace(text(r, "attachment_id"), r);
  std::set<std::string> remote_ids;
  for (const auto &[id, r] : remote_before) {
    (void)r;
    remote_ids.insert(id);
  }
  for (const auto &[id, r] : remote_after) {
    (void)r;
    remote_ids.insert(id);
  }
  for (const auto &id : remote_ids) {
    auto b = remote_before.find(id), a = remote_after.find(id);
    if (b != remote_before.end() && a != remote_after.end() &&
        b->second == a->second)
      continue;
    out.push_back({{"resource_id", id},
                   {"component_kind", a == remote_after.end()
                                          ? text(b->second, "resource_kind")
                                          : text(a->second, "resource_kind")},
                   {"locality", "remote"},
                   {"change", "attachment_changed"},
                   {"physical_change_evidence", false},
                   {"evidence_refs",
                    Json::array({old.at("record_id"), next.at("record_id")})}});
  }
  return out;
}
std::string status(const Json &findings) {
  for (const auto &f : findings)
    if (text(f, "code").starts_with("conflicting_"))
      return "conflicting";
  for (const auto &f : findings)
    if (text(f, "code").starts_with("insufficient_"))
      return "insufficient";
  return "valid";
}
void relationships(const Json &records, Json &findings) {
  std::map<std::string, const Json *> by_id;
  std::set<std::string> superseded;
  std::map<std::string, std::string> successor_records;
  for (const auto &r : records)
    by_id.emplace(text(r, "record_id"), &r);
  for (const auto &r : records) {
    if (r.at("predecessor_record_id").is_null())
      require(integer(r.at("generation")) == 1,
              "Initial generation must be one");
    else {
      auto p = by_id.find(str(r.at("predecessor_record_id")));
      if (p == by_id.end())
        findings.push_back(finding(
            "insufficient_predecessor",
            Json::array({r.at("record_id"), r.at("predecessor_record_id")})));
      else {
        auto [branch, unique_branch] = successor_records.emplace(
            text(*p->second, "record_id"), text(r, "record_id"));
        if (!unique_branch)
          findings.push_back(
              finding("conflicting_inventory_branches",
                      Json::array({branch->second, r.at("record_id")})));
        require(integer(r.at("generation")) ==
                    integer(p->second->at("generation")) + 1,
                "Inventory predecessor generation mismatch");
        if (r.at("physical_node_id") != p->second->at("physical_node_id")) {
          require(!r.at("identity_rebind").is_null() &&
                      r.at("identity_rebind").at("predecessor_node_id") ==
                          p->second->at("physical_node_id") &&
                      r.at("identity_rebind").at("successor_node_id") ==
                          r.at("physical_node_id"),
                  "Inventory subject change lacks retained SNIV reference");
        } else
          require(r.at("identity_rebind") == p->second->at("identity_rebind"),
                  "Inventory correction cannot rewrite identity lineage");
        superseded.insert(text(*p->second, "record_id"));
      }
    }
    if (r.at("coverage") != "complete")
      findings.push_back({{"code", "partial_inventory_coverage"},
                          {"coverage", r.at("coverage")},
                          {"record_refs", Json::array({r.at("record_id")})}});
  }
  std::map<std::string, std::pair<Json, std::string>> facts;
  for (const auto &r : records) {
    if (superseded.contains(text(r, "record_id")) ||
        r.at("status") == "retired")
      continue;
    for (const auto &resource : r.at("local_resources")) {
      auto key =
          r.at("physical_node_id").dump() + resource.at("resource_id").dump();
      Json fact = {{"presence", resource.at("presence")},
                   {"capacity_kind", resource.at("capacity_kind")},
                   {"capacity", resource.at("capacity")},
                   {"instance_ref", resource.at("instance_ref")},
                   {"component_kind", resource.at("component_kind")}};
      auto [it, inserted] =
          facts.emplace(key, std::make_pair(fact, text(r, "record_id")));
      if (!inserted) {
        bool contradiction = false;
        for (const auto &[field, value] : fact.items()) {
          const auto &prior = it->second.first.at(field);
          if (!value.is_null() && !prior.is_null() && value != "unknown" &&
              prior != "unknown" && value != prior)
            contradiction = true;
        }
        if (contradiction)
          findings.push_back(
              finding("conflicting_resource_assertions",
                      Json::array({it->second.second, r.at("record_id")})));
      }
    }
  }
}
} // namespace
std::vector<Operation> operations() {
  return {{"resources_validate",
           "symphony.snrv.resources-validate-input.v1",
           "symphony.snrv.resources-validate.v1",
           {"validate", "propose"}}};
}
Json handle(std::string_view operation, const Json &payload,
            std::int64_t deadline_unix_ms) {
  try {
    deadline(deadline_unix_ms);
    require(operation == "resources_validate", "Unsupported SNRV operation");
    keys(payload,
         {"protocol", "mode", "record_limit", "inventories", "transition"});
    require(payload.at("protocol") ==
                "symphony.snrv.resources-validate-input.v1",
            "Unsupported SNRV protocol");
    choice(payload.at("mode"), {"snapshot", "transition"});
    auto maximum = limit(payload.at("record_limit"));
    const auto &records = array(payload.at("inventories"), maximum);
    std::set<std::string> seen, subjects;
    std::size_t components = 0;
    for (const auto &r : records) {
      deadline(deadline_unix_ms);
      inventory(r, maximum);
      require(seen.insert(text(r, "record_id")).second,
              "Duplicate inventory record identifier");
      subjects.insert(text(r, "physical_node_id"));
      components +=
          r.at("local_resources").size() + r.at("remote_attachments").size();
    }
    require(components <= maximum,
            "Aggregate inventory components exceed selected capacity");
    Json findings = Json::array(), proposals = Json::array(),
         diff = Json::array();
    bool new_subject = false;
    relationships(records, findings);
    if (records.empty())
      findings.push_back(
          finding("insufficient_inventory_records", Json::array()));
    if (payload.at("mode") == "snapshot")
      require(payload.at("transition").is_null(),
              "Snapshot cannot carry transition");
    else {
      const auto &t = payload.at("transition");
      keys(t, {"kind", "predecessor_record_id", "candidate_record"});
      choice(t.at("kind"),
             {"correction", "retirement", "restoration", "resource_change"});
      const Json *old = nullptr;
      for (const auto &r : records)
        if (r.at("record_id") == t.at("predecessor_record_id"))
          old = &r;
      require(old != nullptr, "Transition predecessor not supplied");
      const auto &next = t.at("candidate_record");
      inventory(next, maximum);
      new_subject = next.at("physical_node_id") != old->at("physical_node_id");
      require(next.at("predecessor_record_id") == old->at("record_id") &&
                  integer(next.at("generation")) ==
                      integer(old->at("generation")) + 1,
              "Candidate must follow exact inventory subject and predecessor");
      require(!seen.contains(text(next, "record_id")),
              "Candidate record identifier already used");
      if (next.at("physical_node_id") != old->at("physical_node_id"))
        require(t.at("kind") == "resource_change" &&
                    !next.at("identity_rebind").is_null() &&
                    next.at("identity_rebind").at("predecessor_node_id") ==
                        old->at("physical_node_id"),
                "Changed inventory subject requires supplied SNIV rebind");
      subjects.insert(text(next, "physical_node_id"));
      if (t.at("kind") == "retirement")
        require(old->at("status") == "active" && next.at("status") == "retired",
                "Inventory retirement state mismatch");
      else if (t.at("kind") == "restoration")
        require(old->at("status") == "retired" && next.at("status") == "active",
                "Inventory restoration state mismatch");
      else
        require(next.at("status") == old->at("status"),
                "Unexpected inventory status transition");
      require(records.size() + 1 <= maximum &&
                  components + next.at("local_resources").size() +
                          next.at("remote_attachments").size() <=
                      maximum,
              "Candidate exceeds aggregate capacity");
      diff = changes(*old, next);
      proposals.push_back(next);
      auto candidates = records;
      candidates.push_back(next);
      relationships(candidates, findings);
    }
    Json binding = nullptr;
    if (!proposals.empty())
      binding = proposals.front().at("identity_rebind");
    bool handoff = new_subject;
    for (const auto &c : diff)
      if (c.at("physical_change_evidence") == true)
        handoff = true;
    auto result_status = status(findings);
    if (result_status != "valid")
      proposals = Json::array();
    deadline(deadline_unix_ms);
    return {{"protocol", "symphony.snrv.resources-validate.v1"},
            {"owner", "snrv"},
            {"owner_version", version},
            {"source_digest", digest(payload)},
            {"subject_ids", ids(subjects)},
            {"status", result_status},
            {"records", records},
            {"proposed_records", proposals},
            {"findings", findings},
            {"resource_changes", diff},
            {"identity_handoff",
             {{"required", handoff},
              {"authority", "sniv"},
              {"physical_identity_changed", false},
              {"subject_rebind", binding}}}};
  } catch (const nlohmann::json::exception &) {
    fail("invalid_input", "Invalid SNRV JSON field type");
  }
}
} // namespace symphony::snv::snrv
