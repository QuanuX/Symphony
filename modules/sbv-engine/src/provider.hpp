#pragma once
#include "detail.hpp"
#include <atomic>
#include <memory>
#include <span>
#include <symphony/sbv/provider.h>
#include <symphony/sqav/databento/dbn.hpp>
namespace symphony::sbv::detail {
struct ProviderReply {
  std::int32_t status = SBV_PROVIDER_OK;
  Json value = nullptr;
  Json error = nullptr;
};
struct NativeProviderState;
class NativeProviderInstance;
class NativeProvider final {
public:
  NativeProvider(const Json &selection, std::int64_t deadline_ms);
  const Json &descriptor() const;
  const Json &evidence() const;
  std::unique_ptr<NativeProviderInstance>
  create(std::int64_t deadline_ms,
         const std::atomic_bool *cancelled = nullptr) const;

private:
  std::shared_ptr<NativeProviderState> state_;
};
// Instances retain the loaded provider and its private stage. Integration owns
// topology: one serialized/shared instance or one instance per worker. Do not
// race destruction with calls. No borrowed event/call view survives a call.
class NativeProviderInstance final {
public:
  ~NativeProviderInstance();
  NativeProviderInstance(const NativeProviderInstance &) = delete;
  NativeProviderInstance &operator=(const NativeProviderInstance &) = delete;
  ProviderReply event(std::span<const sqav::databento::Mbo>,
                      std::size_t ordinal, std::int64_t deadline_ms,
                      const std::atomic_bool *cancelled = nullptr);
  ProviderReply finish(std::int64_t deadline_ms,
                       const std::atomic_bool *cancelled = nullptr);
  ProviderReply evaluate(std::span<const sqav::databento::Mbo>,
                         std::size_t signal_ordinal,
                         std::size_t followup_end_exclusive, const Json &signal,
                         const Json &coverage, std::uint64_t horizon_end_ns,
                         std::int64_t deadline_ms,
                         const std::atomic_bool *cancelled = nullptr);

private:
  struct State;
  explicit NativeProviderInstance(std::unique_ptr<State>);
  std::unique_ptr<State> state_;
  friend class NativeProvider;
};
Json provider_inspect(const Json &, std::int64_t);
// Validate a retained self-consistent declaration without loading any code or
// reading present-day files. This does not authenticate provider authorship.
void validate_native_provider_evidence(const Json &selection,
                                       const Json &evidence);
} // namespace symphony::sbv::detail
