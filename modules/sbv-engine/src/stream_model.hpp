#pragma once
#include "logical_value.hpp"
#include <memory>
#include <symphony/sbv/models.hpp>

namespace symphony::sbv::detail {
// Private scalable model admission. Census signals must already satisfy their
// complete census contract. External models additionally verify their exact
// signal-ID set before any evaluate call. Mathematical domains are admitted by
// the existing one-row AdmittedModel, with no new aggregate model-ID cap.
class StreamModel {
public:
  StreamModel(Json selection, logical::Value signals, std::int64_t deadline_ms);
  // The enclosing retained financial result must already be admitted. This
  // overload checks captured source correspondence and selected-array digest
  // without reopening deleted ancestors; it does not re-prove outer closure.
  StreamModel(Json selection, logical::Value signals, logical::Value outcomes,
              Json source_reference, std::int64_t deadline_ms);
  // Archived choices may themselves contain a large inline outcome array.
  // Extract only typed metadata and compare its logical array by hash.
  StreamModel(logical::Value selection, logical::Value signals,
              logical::Value outcomes, Json source_reference,
              std::int64_t deadline_ms);
  ~StreamModel();
  StreamModel(const StreamModel &) = delete;
  StreamModel &operator=(const StreamModel &) = delete;
  Json evaluate(const ModelFrame &, std::int64_t deadline_ms) const;
  std::uint64_t horizon_ns() const;
  Json evidence() const;
  const Json &source_reference() const;
  // External models only. Copy/graft this view into descendants before source
  // removal. Its shared reader cache is single-caller: use outside evaluate's
  // worker phase; evaluate itself serializes row reads and owns each row.
  const logical::Value &outcomes() const;

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
};
} // namespace symphony::sbv::detail
