#pragma once
#include "detail.hpp"
#include <memory>
#include <symphony/sqav/databento/dbn.hpp>
namespace symphony::sbv::detail {
// Nonmovable: metadata.dataset borrows the owned dataset_name, never DBN bytes.
struct Dataset {
  std::string path, sha256, dataset_name;
  sqav::databento::Metadata metadata;
  std::vector<sqav::databento::Mbo> events;
  std::uint64_t source_bytes{}, metadata_bytes{}, load_buffer_bytes{};
  Json selected_limits = Json::object();
  Json memory_budget = nullptr;
  Json limits_evidence() const;
  Json resident_identity = nullptr;
  bool book_compatible = true, locked = false;
  Json source_selection = Json::object(), source_delivery = nullptr;
  std::string load_buffer_scope;
  Dataset() = default;
  Dataset(const Dataset &) = delete;
  ~Dataset();
  void bind(const Json &) const;
  Json evidence(bool resident) const;
};
// Exactly one file identity tuple or retained_source, with no I/O.
Json dataset_source_selection(const Json &);
// Validate retained result choices/delivery against normalized provenance
// without reopening any source, original path, store or provider.
Json dataset_result_identity(const Json &choices, const Json &provenance,
                             const Json &resources);
std::unique_ptr<Dataset> load_dataset(const Json &, std::int64_t,
                                      bool lock = false);
} // namespace symphony::sbv::detail
