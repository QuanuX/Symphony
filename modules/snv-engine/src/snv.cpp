#include <algorithm>
#include <map>
#include <set>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/sciv.hpp>
#include <symphony/snv/scnv.hpp>
#include <symphony/snv/sniv.hpp>
#include <symphony/snv/snrv.hpp>
#include <symphony/snv/snv.hpp>

namespace symphony::snv::snv {
namespace engine = symphony::knowledge::engine;
namespace {
[[noreturn]] void fail(std::string_view code, std::string_view message) {
  throw engine::Error("snv." + std::string(code), std::string(message), 2);
}
void require(bool value, std::string_view message) {
  if (!value)
    fail("invalid_input", message);
}
void keys(const Json &j, std::initializer_list<std::string_view> required) {
  require(j.is_object() && j.size() == required.size(),
          "exact object fields required");
  for (const auto key : required)
    require(j.contains(key), "required field missing");
}
std::string text(const Json &j, std::size_t max = 128) {
  require(j.is_string(), "string required");
  const auto s = j.get<std::string>();
  require(!s.empty() && s.size() <= max, "text bound violated");
  require(std::none_of(s.begin(), s.end(),
                       [](unsigned char c) { return c < 32 || c == 127; }),
          "control character prohibited");
  return s;
}
std::int64_t integer(const Json &j, std::int64_t max = 9007199254740991LL) {
  require(j.is_number_integer() && !j.is_number_float(),
          "exact integer required");
  const auto v = j.get<std::int64_t>();
  require(v >= 0 && v <= max, "integer out of bounds");
  return v;
}
std::string digest(const Json &j) { return engine::tagged_sha256(j.dump()); }
Json seal(Json j) {
  j["digest"] = digest(j);
  return j;
}
void check_seal(const Json &j) {
  require(j.is_object() && j.contains("digest") && j.at("digest").is_string(),
          "sealed object required");
  auto copy = j;
  copy.erase("digest");
  require(j.at("digest") == digest(copy), "object digest mismatch");
}
void deadline(std::int64_t d) {
  if (engine::unix_time_ms() > d)
    fail("deadline_exceeded", "deadline exceeded");
}
Json owner_call(std::string_view owner, std::string_view op, const Json &p,
                std::int64_t d) {
  if (owner == "sniv")
    return sniv::handle(op, p, d);
  if (owner == "snrv")
    return snrv::handle(op, p, d);
  if (owner == "sciv")
    return sciv::handle(op, p, d);
  if (owner == "scnv")
    return scnv::handle(op, p, d);
  fail("unsupported_owner", "unsupported owner");
}
Json replay(const Json &bundle, std::int64_t d) {
  keys(bundle, {"protocol", "bundle_id", "artifacts", "relations"});
  require(bundle.at("protocol") == "symphony.snv.bundle.v1",
          "unsupported bundle protocol");
  require(bundle.dump().size() <= 262144, "bundle byte profile exceeded");
  (void)text(bundle.at("bundle_id"));
  require(bundle.at("artifacts").is_array() &&
              bundle.at("artifacts").size() <= 4,
          "one artifact per owner maximum");
  require(bundle.at("relations").is_array() &&
              bundle.at("relations").size() <= 2048,
          "relation bound violated");
  Json views = Json::object(), subjects = Json::object(),
       findings = Json::array();
  Json identity_records = Json::array(), episodes = Json::array(),
       inventories = Json::array(), name_records = Json::array(),
       cluster_evidence = Json::object();
  std::map<std::string, Json> payloads;
  std::map<std::string, std::set<std::string>> typed;
  auto add = [&](std::string_view owner, std::string_view kind,
                 const Json &id) {
    if (!id.is_null())
      typed[std::string(owner) + ":" + std::string(kind)].insert(text(id));
  };
  for (const auto &artifact : bundle.at("artifacts")) {
    deadline(d);
    keys(artifact, {"owner", "owner_version", "operation", "source_utf8",
                    "source_digest"});
    const auto owner = text(artifact.at("owner"));
    require(!views.contains(owner), "duplicate owner artifact");
    require(artifact.at("owner_version") == version,
            "unsupported embedded owner version");
    const auto op = text(artifact.at("operation"));
    require(artifact.at("source_utf8").is_string(),
            "original UTF8 source required");
    const auto source = artifact.at("source_utf8").get<std::string>();
    require(source.size() <= 65536,
            "owner artifact source byte profile exceeded");
    require(artifact.at("source_digest") == engine::tagged_sha256(source),
            "original source bytes digest mismatch");
    const auto payload =
        engine::parse_bounded_json(source, 1U << 20, max_json_values);
    const auto result = owner_call(owner, op, payload, d);
    require(result.at("owner") == "symphony-" + owner ||
                result.at("owner") == owner,
            "owner result identity mismatch");
    require(result.at("owner_version") == artifact.at("owner_version") &&
                result.at("source_digest") == digest(payload),
            "owner replay binding mismatch");
    views[owner] = result;
    subjects[owner] = result.at("subject_ids");
    payloads[owner] = payload;
    if (owner == "sniv" && result.contains("incarnation_ids"))
      for (const auto &id : result.at("incarnation_ids"))
        subjects[owner].push_back(id);
    if (owner == "sniv") {
      identity_records = payload.at("physical_records");
      episodes = payload.at("participations");
      for (const auto &record : result.at("proposed_records")) {
        if (record.contains("incarnation_id"))
          episodes.push_back(record);
        else
          identity_records.push_back(record);
      }
      for (const auto &record : identity_records)
        add(owner, "node", record.at("physical_node_id"));
      for (const auto &record : episodes) {
        add(owner, "node", record.at("physical_node_id"));
        add(owner, "incarnation", record.at("incarnation_id"));
      }
    }
    if (owner == "snrv") {
      inventories = payload.at("inventories");
      for (const auto &record : result.at("proposed_records"))
        inventories.push_back(record);
      for (const auto &inventory : inventories) {
        for (const auto &resource : inventory.at("local_resources")) {
          subjects[owner].push_back(resource.at("resource_id"));
          add(owner, "resource", resource.at("resource_id"));
        }
        for (const auto &resource : inventory.at("remote_attachments")) {
          subjects[owner].push_back(resource.at("attachment_id"));
          add(owner, "attachment", resource.at("attachment_id"));
        }
      }
    }
    if (owner == "sciv") {
      cluster_evidence =
          result.contains("candidate_evidence")
              ? result.at("candidate_evidence")
              : (payload.contains("evidence") ? payload.at("evidence")
                                              : payload);
      const auto &evidence = cluster_evidence;
      add(owner, "cluster", evidence.at("cluster").at("cluster_id"));
      add(owner, "system", evidence.at("cluster").at("system_id"));
      for (const auto &r : evidence.at("memberships"))
        add(owner, "membership", r.at("record_id"));
      for (const auto &r : evidence.at("observations")) {
        add(owner, "connection", r.at("record_id"));
        add(owner, "bus", r.at("bus_id"));
      }
    }
    if (owner == "scnv") {
      name_records = payload.at("records");
      if (result.contains("proposed_records"))
        for (const auto &record : result.at("proposed_records"))
          name_records.push_back(record);
      for (const auto &r : name_records)
        add(owner, "name_association", r.at("association_id"));
    }
  }
  auto reference = [&](const Json &ref) {
    keys(ref, {"owner", "kind", "id"});
    auto owner = text(ref.at("owner"));
    auto id = text(ref.at("id"));
    auto kind = text(ref.at("kind"));
    require(owner == "sniv" || owner == "snrv" || owner == "sciv" ||
                owner == "scnv",
            "unknown reference owner");
    const std::map<std::string, std::set<std::string>> kinds{
        {"sniv", {"node", "incarnation"}},
        {"snrv", {"resource", "attachment"}},
        {"sciv", {"cluster", "system", "bus", "membership", "connection"}},
        {"scnv", {"name_association"}}};
    require(kinds.at(owner).contains(kind),
            "reference kind does not belong to owner");
    const bool found = typed[owner + ":" + kind].contains(id);
    if (!found)
      findings.push_back(
          {{"code", subjects.contains(owner) ? "unresolved_subject_reference"
                                             : "missing_owner_view"},
           {"reference", ref}});
    return found;
  };
  std::set<std::string> relationship_ids;
  for (const auto &relation : bundle.at("relations")) {
    keys(relation, {"relationship_id", "kind", "from", "to", "source_ref"});
    require(
        relationship_ids.insert(text(relation.at("relationship_id"))).second,
        "duplicate relationship ID");
    (void)text(relation.at("kind"));
    (void)text(relation.at("source_ref"), 1024);
    reference(relation.at("from"));
    reference(relation.at("to"));
  }
  // References in an owner input remain unresolved when its companion is
  // absent.
  if (payloads.contains("snrv"))
    for (const auto &r : inventories)
      reference({{"owner", "sniv"},
                 {"kind", "node"},
                 {"id", r.at("physical_node_id")}});
  if (payloads.contains("sciv")) {
    const auto &evidence = cluster_evidence;
    for (const auto &n : evidence.at("nodes")) {
      reference({{"owner", "sniv"}, {"kind", "node"}, {"id", n.at("node_id")}});
      if (n.contains("incarnation_id")) {
        reference({{"owner", "sniv"},
                   {"kind", "incarnation"},
                   {"id", n.at("incarnation_id")}});
        if (payloads.contains("sniv"))
          for (const auto &episode : episodes) {
            if (episode.at("incarnation_id") == n.at("incarnation_id"))
              require(episode.at("physical_node_id") == n.at("node_id") &&
                          episode.at("system_id") == n.at("system_id"),
                      "incarnation cross-owner subject/system mismatch");
          }
      }
    }
  }
  if (payloads.contains("scnv"))
    for (const auto &r : name_records) {
      const auto kind = r.at("subject").at("kind").get<std::string>();
      if (kind == "relationship") {
        const auto id = text(r.at("subject").at("id"));
        if (!relationship_ids.contains(id) &&
            !typed["sciv:membership"].contains(id) &&
            !typed["sciv:connection"].contains(id))
          findings.push_back({{"code", "unresolved_relationship_reference"},
                              {"subject", r.at("subject")}});
      } else {
        const auto owner = (kind == "node" || kind == "incarnation") ? "sniv"
                           : kind == "resource"                      ? "snrv"
                                                                     : "sciv";
        reference({{"owner", owner},
                   {"kind", kind},
                   {"id", r.at("subject").at("id")}});
      }
      if (r.at("scope").at("kind") == "cluster")
        reference({{"owner", "sciv"},
                   {"kind", "cluster"},
                   {"id", r.at("scope").at("id")}});
    }
  if (payloads.contains("snrv")) {
    const auto &resource_input = payloads.at("snrv");
    for (const auto &inventory : inventories)
      if (!inventory.at("identity_rebind").is_null()) {
        const auto &binding = inventory.at("identity_rebind");
        bool found = false;
        if (payloads.contains("sniv")) {
          for (const auto &record : identity_records)
            if (record.at("record_id") == binding.at("sniv_record_ref")) {
              require(
                  record.at("physical_node_id") ==
                          binding.at("successor_node_id") &&
                      record.at("predecessor_node_id") ==
                          binding.at("predecessor_node_id") &&
                      inventory.at("physical_node_id") ==
                          binding.at("successor_node_id"),
                  "resource identity rebind differs from exact SNIV record");
              found = true;
            }
        }
        if (!found)
          findings.push_back(
              {{"code", "unresolved_identity_rebind"},
               {"binding", binding},
               {"inventory_record_id", inventory.at("record_id")}});
      }
    if (resource_input.at("mode") == "transition" &&
        views.at("snrv").at("identity_handoff").at("required") == true) {
      if (!payloads.contains("sniv") ||
          payloads.at("sniv").at("mode") != "transition" ||
          payloads.at("sniv").at("transition").at("kind") !=
              "physical_change") {
        findings.push_back(
            {{"code", "insufficient_identity_handoff"},
             {"resource_source_digest", views.at("snrv").at("source_digest")}});
      } else {
        const auto &identity_input = payloads.at("sniv");
        const auto &it = identity_input.at("transition");
        const auto &rt = resource_input.at("transition");
        const Json *old_inventory = nullptr;
        for (const auto &r : resource_input.at("inventories"))
          if (r.at("record_id") == rt.at("predecessor_record_id"))
            old_inventory = &r;
        const Json *old_identity = nullptr;
        for (const auto &r : identity_input.at("physical_records"))
          if (r.at("record_id") == it.at("predecessor_record_id"))
            old_identity = &r;
        require(old_inventory && old_identity &&
                    old_inventory->at("physical_node_id") ==
                        old_identity->at("physical_node_id"),
                "resource handoff predecessor subject mismatch");
        const auto &proposed = views.at("sniv").at("proposed_records");
        if (views.at("snrv").at("proposed_records").empty())
          findings.push_back({{"code", "insufficient_resource_handoff"},
                              {"resource_source_digest",
                               views.at("snrv").at("source_digest")}});
        else if (proposed.empty())
          findings.push_back({{"code", "insufficient_materiality_handoff"},
                              {"identity_source_digest",
                               views.at("sniv").at("source_digest")}});
        else {
          require(proposed.size() == 1 &&
                      proposed.at(0).at("physical_node_id") ==
                          rt.at("candidate_record").at("physical_node_id"),
                  "resource candidate and classified physical subject differ");
          const auto &identity = proposed.at(0);
          const auto &resource = views.at("snrv").at("proposed_records").at(0);
          require(resource == rt.at("candidate_record"),
                  "resource handoff candidate differs from admitted proposal");
          const bool new_subject = identity.at("physical_node_id") !=
                                   old_identity->at("physical_node_id");
          if (new_subject) {
            const auto &binding = resource.at("identity_rebind");
            require(!binding.is_null() &&
                        binding.at("sniv_record_ref") ==
                            identity.at("record_id") &&
                        binding.at("predecessor_node_id") ==
                            old_identity->at("physical_node_id") &&
                        binding.at("successor_node_id") ==
                            identity.at("physical_node_id"),
                    "resource handoff must bind the exact classified SNIV "
                    "proposal");
          }
          const bool replaced_subject =
              new_subject && (it.at("change_kind") == "replacement" ||
                              it.at("change_kind") == "provider_resource");
          const bool retired_subject = it.at("change_kind") == "retirement" &&
                                       rt.at("kind") == "retirement" &&
                                       identity.at("status") == "retired" &&
                                       resource.at("status") == "retired";
          for (const auto &change : views.at("snrv").at("resource_changes"))
            if (change.at("physical_change_evidence") == true) {
              bool bound =
                  replaced_subject ||
                  (retired_subject && change.at("change") == "removed");
              if (it.at("change_kind") == "hardware_change")
                for (const auto &claim : it.at("changes")) {
                  if (claim.at("component_kind") !=
                          change.at("component_kind") ||
                      claim.at("change") != change.at("change") ||
                      claim.at("locality") != "local")
                    continue;
                  bool before = false, after = false;
                  for (const auto &ref : claim.at("evidence_refs")) {
                    before |= ref == old_inventory->at("record_id");
                    after |= ref == rt.at("candidate_record").at("record_id");
                  }
                  if (before && after)
                    bound = true;
                }
              require(bound, "physical resource change lacks exact versioned "
                             "SNIV materiality/source binding");
            }
          findings.push_back(
              {{"code", "verified_resource_identity_handoff"},
               {"resource_source_digest", views.at("snrv").at("source_digest")},
               {"identity_source_digest",
                views.at("sniv").at("source_digest")}});
        }
      }
    }
  }
  for (auto &[owner, ids] : subjects.items()) {
    (void)owner;
    std::set<std::string> unique;
    for (const auto &id : ids)
      unique.insert(text(id));
    ids = Json::array();
    for (const auto &id : unique)
      ids.push_back(id);
  }
  Json coverage = Json::object();
  for (const auto *owner : {"sniv", "snrv", "sciv", "scnv"})
    coverage[owner] = views.contains(owner) ? "present" : "missing";
  return {
      {"protocol", "symphony.snv.replay.v1"},
      {"owner", engine_id},
      {"owner_version", version},
      {"bundle_id", bundle.at("bundle_id")},
      {"bundle_digest", digest(bundle)},
      {"coverage", coverage},
      {"partial", views.size() != 4 ||
                      std::any_of(findings.begin(), findings.end(),
                                  [](const Json &f) {
                                    return f.at("code") !=
                                           "verified_resource_identity_handoff";
                                  })},
      {"views", views},
      {"subject_ids", subjects},
      {"findings", findings},
      {"embedded_dependencies", embedded_dependencies()}};
}
Json export_manifest(const Json &bundle) {
  const auto bytes = bundle.dump();
  constexpr std::size_t chunk_size = 16384;
  Json chunks = Json::array();
  for (std::size_t i = 0; i < bytes.size(); i += chunk_size) {
    // Hex chunks preserve UTF8 byte boundaries without lossy string slicing.
    const auto chunk = bytes.substr(i, chunk_size);
    chunks.push_back({{"index", i / chunk_size},
                      {"byte_count", chunk.size()},
                      {"digest", engine::tagged_sha256(chunk)}});
  }
  return seal({{"protocol", "symphony.snv.export-manifest.v1"},
               {"writer", engine_id},
               {"writer_version", version},
               {"bundle_digest", digest(bundle)},
               {"byte_count", bytes.size()},
               {"encoding", "hex"},
               {"chunk_size", chunk_size},
               {"chunks", chunks},
               {"embedded_dependencies", embedded_dependencies()}});
}
std::string hex(std::string_view bytes) {
  constexpr char chars[] = "0123456789abcdef";
  std::string result;
  result.reserve(bytes.size() * 2);
  for (const unsigned char c : bytes) {
    result += chars[c >> 4];
    result += chars[c & 15];
  }
  return result;
}
std::string unhex(const Json &j) {
  require(j.is_string(), "hex bytes required");
  const auto s = j.get<std::string>();
  require(s.size() % 2 == 0 && s.size() <= 32768, "hex chunk bound violated");
  std::string out;
  auto nibble = [](char c) {
    if (c >= '0' && c <= '9')
      return c - '0';
    if (c >= 'a' && c <= 'f')
      return c - 'a' + 10;
    fail("invalid_input", "noncanonical hex");
  };
  for (std::size_t i = 0; i < s.size(); i += 2)
    out += static_cast<char>((nibble(s[i]) << 4) | nibble(s[i + 1]));
  return out;
}
Json import_bundle(const Json &records, std::int64_t d) {
  keys(records, {"manifest", "chunks"});
  const auto &manifest = records.at("manifest");
  check_seal(manifest);
  keys(manifest,
       {"protocol", "writer", "writer_version", "bundle_digest", "byte_count",
        "encoding", "chunk_size", "chunks", "embedded_dependencies", "digest"});
  require(manifest.at("protocol") == "symphony.snv.export-manifest.v1" &&
              manifest.at("writer") == engine_id &&
              manifest.at("writer_version") == version,
          "unsupported export writer");
  require(manifest.at("embedded_dependencies") == embedded_dependencies(),
          "incompatible export reducers");
  require(manifest.at("encoding") == "hex" &&
              manifest.at("chunk_size") == 16384,
          "unsupported export representation");
  require(records.at("chunks").is_array() &&
              records.at("chunks").size() == manifest.at("chunks").size() &&
              records.at("chunks").size() <= 64,
          "incomplete export chunks");
  std::string bytes;
  for (std::size_t i = 0; i < records.at("chunks").size(); ++i) {
    deadline(d);
    const auto &c = records.at("chunks").at(i);
    const auto &spec = manifest.at("chunks").at(i);
    keys(c, {"protocol", "manifest_digest", "bundle_digest", "index", "hex",
             "byte_count", "digest"});
    require(c.at("protocol") == "symphony.snv.export-chunk.v1" &&
                c.at("manifest_digest") == manifest.at("digest") &&
                c.at("bundle_digest") == manifest.at("bundle_digest"),
            "mixed export revision");
    const auto raw = unhex(c.at("hex"));
    require(c.at("index") == i && spec.at("index") == i &&
                c.at("digest") == spec.at("digest") &&
                c.at("byte_count") == spec.at("byte_count") &&
                raw.size() == static_cast<std::size_t>(
                                  integer(c.at("byte_count"), 16384)) &&
                c.at("digest") == engine::tagged_sha256(raw),
            "export chunk binding mismatch");
    bytes += raw;
    require(bytes.size() <= 1U << 20, "import bytes exceeded");
  }
  require(bytes.size() == static_cast<std::size_t>(
                              integer(manifest.at("byte_count"), 1U << 20)),
          "export length mismatch");
  const auto bundle =
      engine::parse_bounded_json(bytes, 1U << 20, max_json_values);
  require(manifest.at("bundle_digest") == digest(bundle) &&
              bundle.dump() == bytes,
          "export canonical bundle mismatch");
  (void)replay(bundle, d);
  return bundle;
}
Json inspect(const Json &p, std::int64_t d) {
  keys(p, {"protocol", "mode", "bundle", "baseline", "offset", "limit"});
  require(p.at("protocol") == "symphony.snv.inspect-input.v1",
          "unsupported inspect protocol");
  const auto mode = text(p.at("mode"));
  const auto offset = integer(p.at("offset"), 2048);
  const auto limit = integer(p.at("limit"), 2048);
  require(limit > 0, "positive page limit required");
  const auto result = replay(p.at("bundle"), d);
  Json out = result;
  out["protocol"] = "symphony.snv.inspect.v1";
  out["mode"] = mode;
  out["source_digest"] = digest(p);
  if (mode == "replay" || mode == "projection") {
    require(p.at("baseline").is_null() && offset == 0,
            "unsupported projection page/baseline");
    return out;
  }
  if (mode == "diff") {
    const auto before = replay(p.at("baseline"), d);
    Json changed = Json::array();
    for (const auto *owner : {"sniv", "snrv", "sciv", "scnv"}) {
      const auto previous = before.at("views").contains(owner)
                                ? before.at("views").at(owner)
                                : Json(nullptr);
      const auto current = result.at("views").contains(owner)
                               ? result.at("views").at(owner)
                               : Json(nullptr);
      if (previous != current)
        changed.push_back(
            {{"owner", owner}, {"before", previous}, {"after", current}});
    }
    out["baseline_digest"] = before.at("bundle_digest");
    out["changes"] = changed;
    return out;
  }
  require(p.at("baseline").is_null(), "unexpected baseline");
  if (mode == "history") {
    Json rows = Json::array();
    for (const auto &artifact : p.at("bundle").at("artifacts")) {
      const auto owner = artifact.at("owner").get<std::string>();
      auto source = engine::parse_bounded_json(
          artifact.at("source_utf8").get<std::string>(), 65536,
          max_json_values);
      if (owner == "sciv" && source.contains("evidence"))
        source = source.at("evidence");
      for (const auto *field :
           {"physical_records", "participations", "inventories", "nodes",
            "memberships", "observations", "records"}) {
        if (source.contains(field))
          for (const auto &record : source.at(field))
            rows.push_back({{"owner", owner},
                            {"kind", field},
                            {"source_digest", artifact.at("source_digest")},
                            {"record", record}});
      }
      const auto &view = result.at("views").at(owner);
      if (view.contains("proposed_records"))
        for (const auto &record : view.at("proposed_records"))
          rows.push_back({{"owner", owner},
                          {"kind", "proposal"},
                          {"source_digest", artifact.at("source_digest")},
                          {"record", record}});
      if (owner == "sciv" && view.contains("candidate_evidence")) {
        std::set<std::string> retained;
        for (const auto *field : {"memberships", "observations"})
          for (const auto &record : source.at(field))
            retained.insert(text(record.at("record_id")));
        for (const auto *field : {"memberships", "observations"})
          for (const auto &record : view.at("candidate_evidence").at(field))
            if (!retained.contains(text(record.at("record_id"))))
              rows.push_back({{"owner", owner},
                              {"kind", "proposal"},
                              {"source_digest", artifact.at("source_digest")},
                              {"record", record}});
      }
    }
    Json page = Json::array();
    for (auto i = static_cast<std::size_t>(offset);
         i < rows.size() && page.size() < static_cast<std::size_t>(limit); ++i)
      page.push_back(rows.at(i));
    out["history"] = page;
    out["total"] = rows.size();
    out["next_offset"] = offset + static_cast<std::int64_t>(page.size());
    return out;
  }
  const auto manifest = export_manifest(p.at("bundle"));
  if (mode == "export_manifest") {
    out.erase("views");
    out["export_manifest"] = manifest;
    return out;
  }
  if (mode == "export_chunk") {
    require(static_cast<std::size_t>(offset) < manifest.at("chunks").size(),
            "export chunk not found");
    const auto raw = p.at("bundle").dump().substr(
        static_cast<std::size_t>(offset) * 16384, 16384);
    out.erase("views");
    out["export_chunk"] = {{"protocol", "symphony.snv.export-chunk.v1"},
                           {"manifest_digest", manifest.at("digest")},
                           {"bundle_digest", manifest.at("bundle_digest")},
                           {"index", offset},
                           {"hex", hex(raw)},
                           {"byte_count", raw.size()},
                           {"digest", engine::tagged_sha256(raw)}};
    return out;
  }
  fail("unsupported_mode", "unsupported inspection mode");
}
Json state_plan(const Json &p, std::int64_t d) {
  keys(p, {"protocol", "view", "operation_id", "expected_state_digest",
           "prior_head", "change_kind", "bundle", "reason", "migration"});
  require(p.at("protocol") == "symphony.snv.state-plan-input.v1",
          "unsupported state plan protocol");
  keys(p.at("view"), {"tops_id", "view_id"});
  (void)text(p.at("view").at("tops_id"));
  (void)text(p.at("view").at("view_id"));
  (void)text(p.at("operation_id"));
  (void)text(p.at("reason"), 1024);
  const auto kind = text(p.at("change_kind"));
  require(kind == "select" || kind == "unselect", "unsupported state change");
  std::int64_t generation = 1;
  if (p.at("prior_head").is_null())
    require(p.at("expected_state_digest").is_null(),
            "initial expected state must be null");
  else {
    const auto &head = p.at("prior_head");
    check_seal(head);
    keys(head,
         {"protocol", "view", "generation", "previous_digest", "bundle_digest",
          "tombstone", "operation_id", "owner_version", "digest"});
    require(head.at("protocol") == "symphony.snv.head.v1" &&
                head.at("owner_version") == version &&
                head.at("view") == p.at("view") &&
                head.at("digest") == p.at("expected_state_digest"),
            "prior head identity/version mismatch");
    generation = integer(head.at("generation"), 9007199254740990LL) + 1;
  }
  Json bundle_digest = nullptr;
  if (kind == "select")
    bundle_digest = replay(p.at("bundle"), d).at("bundle_digest");
  else
    require(p.at("bundle").is_null() && !p.at("prior_head").is_null(),
            "unselect requires prior view and no bundle");
  if (!p.at("migration").is_null()) {
    keys(p.at("migration"),
         {"from_owner_version", "to_owner_version", "source_bundle_digest",
          "method", "method_version"});
    require(p.at("migration").at("to_owner_version") == version,
            "incompatible migration target");
    (void)text(p.at("migration").at("from_owner_version"));
    (void)text(p.at("migration").at("method"));
    (void)text(p.at("migration").at("method_version"));
    require(!p.at("prior_head").is_null() &&
                p.at("migration").at("from_owner_version") ==
                    p.at("prior_head").at("owner_version"),
            "migration source owner version mismatch");
    require(!p.at("prior_head").is_null() &&
                p.at("migration").at("source_bundle_digest") ==
                    p.at("prior_head").at("bundle_digest"),
            "migration source mismatch");
  }
  auto head = seal({{"protocol", "symphony.snv.head.v1"},
                    {"view", p.at("view")},
                    {"generation", generation},
                    {"previous_digest", p.at("expected_state_digest")},
                    {"bundle_digest", bundle_digest},
                    {"tombstone", kind == "unselect"},
                    {"operation_id", p.at("operation_id")},
                    {"owner_version", version}});
  return seal({{"protocol", "symphony.snv.state-plan.v1"},
               {"operation_id", p.at("operation_id")},
               {"expected_state_digest", p.at("expected_state_digest")},
               {"change_kind", kind},
               {"reason", p.at("reason")},
               {"migration", p.at("migration")},
               {"head", head},
               {"input_digest", digest(p)}});
}
} // namespace
Json embedded_dependencies() {
  return Json::array(
      {Json{{"engine_id", sniv::engine_id}, {"version", sniv::version}},
       Json{{"engine_id", snrv::engine_id}, {"version", snrv::version}},
       Json{{"engine_id", sciv::engine_id}, {"version", sciv::version}},
       Json{{"engine_id", scnv::engine_id}, {"version", scnv::version}}});
}
std::vector<Operation> operations() {
  return {{"snv_inspect",
           "symphony.snv.inspect-input.v1",
           "symphony.snv.inspect.v1",
           {"inspect", "query"}},
          {"snv_evidence_plan",
           "symphony.snv.evidence-plan-input.v1",
           "symphony.snv.evidence-plan.v1",
           {"propose"}},
          {"snv_state_plan",
           "symphony.snv.state-plan-input.v1",
           "symphony.snv.state-plan.v1",
           {"propose"}},
          {"snv_state_reduce",
           "symphony.snv.state-reduce-input.v1",
           "symphony.snv.state-transition.v1",
           {"apply", "recover"}}};
}
Json handle(std::string_view operation, const Json &p, std::int64_t d) {
  deadline(d);
  if (operation == "snv_inspect")
    return inspect(p, d);
  if (operation == "snv_state_plan")
    return state_plan(p, d);
  if (operation == "snv_evidence_plan") {
    keys(p, {"protocol", "operation_id", "mode", "bundle", "export_records"});
    require(p.at("protocol") == "symphony.snv.evidence-plan-input.v1",
            "unsupported evidence plan");
    (void)text(p.at("operation_id"));
    const auto mode = text(p.at("mode"));
    Json bundle;
    if (mode == "bundle") {
      require(p.at("export_records").is_null(), "unexpected export records");
      bundle = p.at("bundle");
    } else if (mode == "import") {
      require(p.at("bundle").is_null(), "unexpected import bundle");
      bundle = import_bundle(p.at("export_records"), d);
    } else
      fail("unsupported_mode", "unsupported evidence input mode");
    const auto result = replay(bundle, d);
    return seal({{"protocol", "symphony.snv.evidence-plan.v1"},
                 {"operation_id", p.at("operation_id")},
                 {"bundle", bundle},
                 {"bundle_digest", result.at("bundle_digest")},
                 {"replay", result},
                 {"source_digest", digest(p)}});
  }
  if (operation == "snv_state_reduce") {
    keys(p, {"protocol", "input", "plan"});
    require(p.at("protocol") == "symphony.snv.state-reduce-input.v1",
            "unsupported reduce protocol");
    const auto expected = state_plan(p.at("input"), d);
    require(expected == p.at("plan"), "plan replay mismatch");
    return seal(
        {{"protocol", "symphony.snv.state-transition.v1"},
         {"operation_id", expected.at("operation_id")},
         {"expected_state_digest", expected.at("expected_state_digest")},
         {"head", expected.at("head")},
         {"plan_digest", expected.at("digest")},
         {"effect", "proposed_only"}});
  }
  fail("unsupported_operation", "unsupported operation");
}
} // namespace symphony::snv::snv
