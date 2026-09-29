#pragma once
#include "common.hpp"
#include <symphony/sqmv/metadata.hpp>
namespace symphony::sqmv::administration_support {
using namespace symphony::sqv_admin;
inline Limits limits(const Json &j) {
  fields(j, {"max_manifest_bytes", "max_field_bytes", "max_evidence_refs"});
  return {narrow<std::uint32_t>(u64(j, "max_manifest_bytes")),
          narrow<std::uint32_t>(u64(j, "max_field_bytes")),
          narrow<std::uint16_t>(u64(j, "max_evidence_refs"))};
}
inline EvidenceRole role(std::string_view name) {
  constexpr std::array<std::string_view, 8> names{
      "schema", "layout",   "access",  "source",
      "time",   "coverage", "lineage", "units"};
  auto it = std::find(names.begin(), names.end(), name);
  require(it != names.end());
  return static_cast<EvidenceRole>(it - names.begin() + 1);
}
inline const char *role_name(EvidenceRole r) {
  constexpr std::array<const char *, 8> names{"schema",  "layout", "access",
                                              "source",  "time",   "coverage",
                                              "lineage", "units"};
  auto n = static_cast<unsigned>(r);
  require(n >= 1 && n <= names.size());
  return names[n - 1];
}
inline Description description(const Json &j) {
  fields(j, {"dataset_id", "dataset_revision", "schema_version",
             "layout_version", "access_scope", "producer_ref", "evidence"});
  Description d{bytes(j, "dataset_id"),
                bytes(j, "dataset_revision"),
                bytes(j, "schema_version"),
                bytes(j, "layout_version"),
                bytes(j, "access_scope"),
                bytes(j, "producer_ref"),
                {}};
  const auto &evidence = j.at("evidence");
  require(evidence.is_array() && evidence.size() <= 128);
  for (const auto &e : evidence) {
    fields(e, {"role", "producer_ref", "evidence_ref"});
    d.evidence.push_back({role(text(e, "role", 16)), bytes(e, "producer_ref"),
                          bytes(e, "evidence_ref")});
  }
  return d;
}
inline Json describe(const Description &d,
                     std::string_view projection = "all") {
  Json refs = Json::array();
  for (const auto &e : d.evidence)
    if (projection == "all" || role_name(e.role) == projection)
      refs.push_back({{"role", role_name(e.role)},
                      {"producer_ref", hex(e.producer_ref)},
                      {"evidence_ref", hex(e.evidence_ref)}});
  return {{"dataset_id", hex(d.dataset_id)},
          {"dataset_revision", hex(d.dataset_revision)},
          {"schema_version", hex(d.schema_version)},
          {"layout_version", hex(d.layout_version)},
          {"access_scope", hex(d.access_scope)},
          {"producer_ref", hex(d.producer_ref)},
          {"evidence", refs}};
}
inline Json binding(const sqfv::Binding &b) {
  return {{"metadata_ref", hex(b.metadata_ref)},
          {"dataset_revision", hex(b.dataset_revision)},
          {"schema_version", hex(b.schema_version)},
          {"layout_version", hex(b.layout_version)},
          {"access_scope", hex(b.access_scope)}};
}
inline Json chunks(ByteView raw) {
  Json out = Json::array();
  for (std::size_t i = 0; i < raw.size(); i += 16384)
    out.push_back(
        hex(raw.subspan(i, std::min<std::size_t>(16384, raw.size() - i))));
  return out;
}
inline std::string decode_chunks(const Json &j) {
  require(j.is_array() && !j.empty() && j.size() <= 4);
  std::string raw;
  for (std::size_t i = 0; i < j.size(); ++i) {
    require(j[i].is_string());
    auto part = unhex(j[i].get_ref<const std::string &>(), 16384);
    require(!part.empty() && (i + 1 == j.size() || part.size() == 16384));
    raw += part;
  }
  return raw;
}
} // namespace symphony::sqmv::administration_support
