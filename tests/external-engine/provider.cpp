#include "calculation.hpp"
#include <algorithm>
#include <cstdlib>
#include <cstring>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/provider.h>
namespace {
using beta_example::J;
struct Instance {
  std::size_t window;
};
void release(void *, const uint8_t *p, uint64_t) {
  std::free(const_cast<uint8_t *>(p));
}
void output(sbv_owned_bytes_v1 *out, const J &j) {
  const auto s = j.dump();
  auto *p = static_cast<uint8_t *>(std::malloc(s.size()));
  if (!p)
    throw std::bad_alloc();
  std::memcpy(p, s.data(), s.size());
  *out = {sizeof(*out), 1, p, s.size(), nullptr, release};
}
bool cancelled(const sbv_call_context_v1 *ctx) {
  return ctx && ctx->cancel_state &&
         ctx->cancel_state(ctx->token) != SBV_CALL_RUNNING;
}
template <class F> int32_t guarded(sbv_owned_bytes_v1 *err, F fn) noexcept {
  try {
    return fn();
  } catch (...) {
    try {
      output(err, J{{"code", "beta.example.invalid"},
                    {"message", "bounded empirical example refused input"},
                    {"details", J::object()}});
    } catch (...) {
    }
    return SBV_PROVIDER_INVALID_INPUT;
  }
}
J parse(const sbv_bytes_v1 &b) {
  if (b.struct_size < sizeof(b) || b.abi_version != 1 || b.size > 32768 ||
      !b.data)
    throw std::runtime_error("invalid bytes");
  return symphony::knowledge::engine::parse_bounded_json(
      std::string(reinterpret_cast<const char *>(b.data),
                  static_cast<std::size_t>(b.size)),
      32768, 4096);
}
int32_t descriptor(const sbv_call_context_v1 *ctx, sbv_owned_bytes_v1 *out,
                   sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    if (cancelled(ctx))
      return SBV_PROVIDER_CANCELLED;
    output(out, J{{"protocol", "symphony.sbv.native-provider-descriptor.v1"},
                  {"id", beta_example::id},
                  {"version", BETA_EXAMPLE_VERSION},
                  {"roles", J::array({"model"})},
                  {"input_profiles",
                   J::array({"symphony.sbv.provider-databento-mbo-event.v1"})},
                  {"config_schema", J{{"type", "object"}}},
                  {"strategy_concurrency", J::array()},
                  {"model_concurrency", J::array({"serialized_instance"})},
                  {"reproducibility", "deterministic_declared"},
                  {"cancellation", "cooperative_polling"},
                  {"dependency_disclosure",
                   "Installed public foundation static library and platform "
                   "C++ runtime only; no service, provider network, state or "
                   "administrative dependency"},
                  {"extensions", J::object()}});
    return SBV_PROVIDER_OK;
  });
}
int32_t create(const sbv_call_context_v1 *ctx, const sbv_bytes_v1 *config,
               void **instance, sbv_owned_bytes_v1 *err) {
  *instance = nullptr;
  return guarded(err, [&]() -> int32_t {
    if (cancelled(ctx))
      return SBV_PROVIDER_CANCELLED;
    const auto p = parse(*config);
    if (!p.is_object() || p.size() != 2 || p.at("version") != "1")
      throw std::runtime_error("configuration version");
    const auto n = beta_example::integer(p.at("window"));
    if (n < 1 || n > 64)
      throw std::runtime_error("window bound");
    *instance = new Instance{static_cast<std::size_t>(n)};
    return SBV_PROVIDER_OK;
  });
}
void destroy(void *p) { delete static_cast<Instance *>(p); }
int32_t model(void *p, const sbv_call_context_v1 *ctx,
              const sbv_model_frame_v1 *input, sbv_owned_bytes_v1 *out,
              sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    if (cancelled(ctx))
      return SBV_PROVIDER_CANCELLED;
    if (!p || !input || input->struct_size < sizeof(*input) ||
        input->abi_version != 1 || !input->causal_events)
      throw std::runtime_error("model frame");
    const auto &reader = *input->causal_events;
    if (reader.struct_size < sizeof(reader) || reader.abi_version != 1 ||
        !reader.read || reader.end_ordinal_exclusive < reader.first_ordinal ||
        reader.end_ordinal_exclusive - reader.first_ordinal > 100000)
      throw std::runtime_error("reader bound");
    std::vector<std::int64_t> prices;
    const auto window = static_cast<Instance *>(p)->window;
    for (auto ordinal = reader.end_ordinal_exclusive;
         ordinal > reader.first_ordinal && prices.size() < window;) {
      if (cancelled(ctx))
        return SBV_PROVIDER_CANCELLED;
      sbv_mbo_event_v1 event{};
      event.struct_size = sizeof(event);
      event.abi_version = 1;
      if (reader.read(reader.token, --ordinal, &event) != SBV_PROVIDER_OK ||
          event.source_ordinal != ordinal)
        throw std::runtime_error("reader refused");
      if (event.action == 'T')
        prices.push_back(event.price_nanos);
    }
    std::reverse(prices.begin(), prices.end());
    if (prices.empty())
      return SBV_PROVIDER_UNAVAILABLE;
    const auto result = beta_example::calculate(prices);
    const auto signal = parse(input->signal_json);
    // This empirical price distribution makes no fill/execution probability
    // claim.
    output(out,
           J{{"signal_id", signal.at("signal").at("signal_id")},
             {"support", result.at("support")},
             {"execution_probability",
              J{{"status", "unavailable"},
                {"reason",
                 "Empirical prices do not estimate execution probability"}}},
             {"evidence_reference", beta_example::evidence(result)}});
    return SBV_PROVIDER_OK;
  });
}
} // namespace
extern "C" SBV_PROVIDER_EXPORT int32_t
symphony_sbv_provider_api_v1(sbv_provider_api_v1 *api) {
  if (!api || api->struct_size < sizeof(*api) || api->abi_version != 1)
    return SBV_PROVIDER_INVALID_INPUT;
  *api = {sizeof(*api),
          1,
          SBV_PROVIDER_MODEL,
          0,
          descriptor,
          nullptr,
          nullptr,
          nullptr,
          nullptr,
          create,
          model,
          destroy};
  return SBV_PROVIDER_OK;
}
