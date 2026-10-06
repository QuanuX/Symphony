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
  Dataset() = default;
  Dataset(const Dataset &) = delete;
  ~Dataset();
  void bind(const Json &) const;
  Json evidence(bool resident) const;
};
std::unique_ptr<Dataset> load_dataset(const Json &, std::int64_t,
                                      bool lock = false);
} // namespace symphony::sbv::detail
