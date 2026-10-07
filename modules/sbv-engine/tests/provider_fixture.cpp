#include <atomic>
#include <chrono>
#include <cstdlib>
#include <cstring>
#include <limits>
#include <stdexcept>
#include <string>
#include <symphony/sbv/provider.h>
#include <thread>
#include <tuple>
namespace {
std::atomic<unsigned long long> releases{}, creates{}, destroys{};
void release(void *, const uint8_t *p, uint64_t) {
  ++releases;
  std::free(const_cast<uint8_t *>(p));
}
void output(sbv_owned_bytes_v1 *out, const std::string &s) {
  auto *p = static_cast<uint8_t *>(std::malloc(s.size()));
  if (!p)
    throw std::bad_alloc();
  std::memcpy(p, s.data(), s.size());
  *out = {sizeof(*out), 1, p, s.size(), nullptr, release};
}
std::string quote(const std::string &s) {
  std::string out = "\"";
  constexpr char hex[] = "0123456789abcdef";
  for (unsigned char c : s) {
    if (c == '"' || c == '\\') {
      out += '\\';
      out += static_cast<char>(c);
    } else if (c < 32) {
      out += "\\u00";
      out += hex[c >> 4];
      out += hex[c & 15];
    } else
      out += static_cast<char>(c);
  }
  return out + '"';
}
// Deliberately small independently authored fixture parser. Host supplies valid
// JSON; this test provider accepts only simple string parameters and signal IDs
// (ordinary quoted escapes supported). It is not a general JSON implementation.
std::string field(const std::string &s, const std::string &key,
                  const std::string &fallback) {
  auto at = s.find('"' + key + '"');
  if (at == std::string::npos)
    return fallback;
  at = s.find(':', at + key.size() + 2);
  if (at == std::string::npos)
    throw std::runtime_error("fixture field");
  at = s.find_first_not_of(" \n\r\t", at + 1);
  if (at == std::string::npos || s[at++] != '"')
    throw std::runtime_error("fixture string field");
  std::string out;
  for (; at < s.size(); ++at) {
    auto c = s[at];
    if (c == '"')
      return out;
    if (c == '\\') {
      if (++at == s.size())
        break;
      c = s[at];
      if (c == 'n')
        c = '\n';
      else if (c == 'r')
        c = '\r';
      else if (c == 't')
        c = '\t';
      else if (c != '\\' && c != '"' && c != '/')
        throw std::runtime_error("fixture escape");
    }
    out += c;
  }
  throw std::runtime_error("fixture unterminated field");
}
uint64_t number(const std::string &s) {
  std::size_t count = 0;
  auto n = std::stoull(s, &count);
  if (count != s.size())
    throw std::runtime_error("fixture integer");
  return n;
}
int32_t error(sbv_owned_bytes_v1 *out, int32_t status, const char *reason) {
  output(out, "{\"code\":\"fixture\",\"message\":" + quote(reason) +
                  ",\"details\":{}}");
  return status;
}
template <class F> int32_t guarded(sbv_owned_bytes_v1 *err, F &&fn) noexcept {
  try {
    return fn();
  } catch (...) {
    try {
      return error(err, SBV_PROVIDER_FAILED, "fixture callback failure");
    } catch (...) {
      return SBV_PROVIDER_FAILED;
    }
  }
}
auto fields(const sbv_mbo_event_v1 &e) {
  return std::make_tuple(e.source_ordinal, e.publisher_id, e.instrument_id,
                         e.ts_event, e.order_id, e.price_nanos, e.size, e.flags,
                         e.channel_id, e.action, e.side, e.ts_recv,
                         e.ts_in_delta, e.sequence, e.has_ts_out, e.ts_out);
}
sbv_mbo_event_v1 blank() {
  sbv_mbo_event_v1 e{};
  e.struct_size = sizeof(e);
  e.abi_version = 1;
  e.source_ordinal = UINT64_MAX;
  return e;
}
bool bounds(const sbv_event_reader_v1 &r, bool probe_thread = true) {
  auto out = blank();
  const auto prior = fields(out);
  if (r.read(r.token, r.end_ordinal_exclusive, &out) !=
          SBV_PROVIDER_INVALID_INPUT ||
      fields(out) != prior)
    return false;
  if (r.first_ordinal && (r.read(r.token, r.first_ordinal - 1, &out) !=
                              SBV_PROVIDER_INVALID_INPUT ||
                          fields(out) != prior))
    return false;
  if (!probe_thread)
    return true;
  // Wrong-thread calls are tested while the context remains actively alive,
  // never after the callback returns or its stack context has expired.
  int32_t status = SBV_PROVIDER_OK;
  std::thread wrong([&] { status = r.read(r.token, r.first_ordinal, &out); });
  wrong.join();
  return status == SBV_PROVIDER_INVALID_INPUT && fields(out) == prior;
}
struct Instance {
  std::string config;
  uint64_t calls = 0;
  sbv_mbo_event_v1 last{};
};
int32_t create(const sbv_call_context_v1 *, const sbv_bytes_v1 *config,
               void **out, sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    std::string text(reinterpret_cast<const char *>(config->data),
                     static_cast<std::size_t>(config->size));
    if (field(text, "mode", "normal") == "create_failure")
      return error(err, SBV_PROVIDER_FAILED, "selected create failure");
    auto *instance = new Instance{std::move(text), 0, {}};
    const auto created = ++creates;
    *out = instance;
    if (field(instance->config, "mode", "normal") == "create_failure_owned" &&
        created == 1)
      return error(err, SBV_PROVIDER_FAILED,
                   "deliberately published instance on failed create");
    return SBV_PROVIDER_OK;
  });
}
void destroy(void *p) {
  ++destroys;
  delete static_cast<Instance *>(p);
}
int32_t selected_failure(const Instance &i, const sbv_call_context_v1 *ctx,
                         sbv_owned_bytes_v1 *out, sbv_owned_bytes_v1 *err) {
  const auto mode = field(i.config, "mode", "normal");
  if (mode == "provider_failure")
    return error(err, SBV_PROVIDER_FAILED, "selected provider failure");
  if (mode == "unavailable")
    return error(err, SBV_PROVIDER_UNAVAILABLE, "selected model unavailable");
  if (mode == "cancel_poll") {
    while (ctx->cancel_state(ctx->token) == SBV_CALL_RUNNING)
      std::this_thread::yield();
    return error(err, SBV_PROVIDER_CANCELLED,
                 "cooperative cancellation observed");
  }
  if (mode == "malformed_json") {
    output(out, "{invalid");
    return SBV_PROVIDER_OK;
  }
  if (mode == "bad_buffer_abi") {
    output(out, "[]");
    out->abi_version = 2;
    return SBV_PROVIDER_OK;
  }
  if (mode == "zero_size_owned") {
    output(out, "[]");
    out->size = 0;
    return SBV_PROVIDER_OK;
  }
  if (mode == "unknown_status") {
    output(out, "[]");
    return 99;
  }
  if (mode == "owned_error") {
    output(out, "[]");
    return error(err, SBV_PROVIDER_FAILED,
                 "both buffers are deliberately owned");
  }
  return -1;
}
int32_t strategy_event(void *p, const sbv_call_context_v1 *ctx,
                       const sbv_strategy_event_v1 *input,
                       sbv_owned_bytes_v1 *out, sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    auto &i = *static_cast<Instance *>(p);
    const auto fail_after = field(i.config, "fail_after", "");
    if (!fail_after.empty() && i.calls >= number(fail_after))
      return error(err, SBV_PROVIDER_FAILED,
                   "selected failure after completed events");
    ++i.calls;
    auto unusual = selected_failure(i, ctx, out, err);
    if (unusual != -1)
      return unusual;
    const auto &e = *input->event;
    i.last = e;
    auto copy = blank();
    if (input->prefix->first_ordinal != 0 ||
        input->prefix->end_ordinal_exclusive != e.source_ordinal + 1 ||
        !bounds(*input->prefix, i.calls == 1) ||
        input->prefix->read(input->prefix->token, e.source_ordinal, &copy) !=
            SBV_PROVIDER_OK ||
        fields(copy) != fields(e))
      return error(err, SBV_PROVIDER_FAILED,
                   "full MBO prefix correspondence failed");
    const auto every = number(field(i.config, "emit_every", "1")),
               emissions = number(field(i.config, "emissions_per_event", "1"));
    if (!every)
      return error(err, SBV_PROVIDER_INVALID_INPUT, "zero fixture stride");
    const auto action = field(i.config, "action", "any");
    if (e.source_ordinal % every ||
        (action != "any" &&
         action != std::string(1, static_cast<char>(e.action))))
      return SBV_PROVIDER_OK;
    auto mode = field(i.config, "mode", "normal");
    std::string text = "[";
    for (uint64_t n = 0; n < emissions; ++n) {
      if (ctx->cancel_state(ctx->token) != SBV_CALL_RUNNING)
        return error(err, SBV_PROVIDER_CANCELLED, "emission cancelled");
      if (n)
        text += ',';
      const auto id = mode == "duplicate_id"
                          ? "duplicate"
                          : "provider-" + std::to_string(e.source_ordinal) +
                                "-" + std::to_string(n);
      auto price = mode == "bad_price" ? INT64_MAX : e.price_nanos;
      text +=
          "{\"signal_id\":" + quote(id) +
          ",\"anchor_price_nanos\":" + quote(std::to_string(price)) +
          ",\"context_reference\":" +
          quote("fixture://mbo/" + std::to_string(e.order_id) + "/" +
                std::to_string(e.flags) + "/" + std::to_string(e.channel_id)) +
          "}";
    }
    output(out, text + "]");
    return SBV_PROVIDER_OK;
  });
}
int32_t finish(void *p, const sbv_call_context_v1 *, sbv_owned_bytes_v1 *out,
               sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    const auto &i = *static_cast<Instance *>(p);
    auto ext = "{\"events\":" + quote(std::to_string(i.calls));
    const auto add = [&](const char *name, auto value) {
      ext += ',' + quote(std::string("last_") + name) + ':' +
             quote(std::to_string(value));
    };
    add("source_ordinal", i.last.source_ordinal);
    add("publisher_id", i.last.publisher_id);
    add("instrument_id", i.last.instrument_id);
    add("ts_event", i.last.ts_event);
    add("order_id", i.last.order_id);
    add("price_nanos", i.last.price_nanos);
    add("size", i.last.size);
    add("flags", i.last.flags);
    add("channel_id", i.last.channel_id);
    add("action", i.last.action);
    add("side", i.last.side);
    add("ts_recv", i.last.ts_recv);
    add("ts_in_delta", i.last.ts_in_delta);
    add("sequence", i.last.sequence);
    add("has_ts_out", i.last.has_ts_out);
    add("ts_out", i.last.ts_out);
    if (field(i.config, "report_counters", "false") == "true")
      ext += ",\"releases\":" + quote(std::to_string(releases.load())) +
             ",\"creates\":" + quote(std::to_string(creates.load())) +
             ",\"destroys\":" + quote(std::to_string(destroys.load()));
    output(out, "{\"diagnostics\":[\"full MBO and invocation prefix checks "
                "passed\"],\"extensions\":" +
                    ext + "}}");
    return SBV_PROVIDER_OK;
  });
}
int32_t model(void *p, const sbv_call_context_v1 *ctx,
              const sbv_model_frame_v1 *input, sbv_owned_bytes_v1 *out,
              sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    const auto &i = *static_cast<Instance *>(p);
    auto unusual = selected_failure(i, ctx, out, err);
    if (unusual != -1)
      return unusual;
    if (!bounds(*input->causal_events) || !bounds(*input->followup_events) ||
        !input->causal_events->end_ordinal_exclusive ||
        input->causal_events->end_ordinal_exclusive !=
            input->followup_events->first_ordinal)
      return error(err, SBV_PROVIDER_FAILED,
                   "model reader role boundaries failed");
    auto anchor = blank();
    if (input->causal_events->read(input->causal_events->token,
                                   input->causal_events->end_ordinal_exclusive -
                                       1,
                                   &anchor) != SBV_PROVIDER_OK)
      return error(err, SBV_PROVIDER_FAILED, "model anchor read failed");
    if (input->followup_events->first_ordinal !=
        input->followup_events->end_ordinal_exclusive) {
      auto end = blank();
      if (input->followup_events->read(
              input->followup_events->token,
              input->followup_events->end_ordinal_exclusive - 1,
              &end) != SBV_PROVIDER_OK ||
          end.ts_recv > input->horizon_end_ns)
        return error(err, SBV_PROVIDER_FAILED, "model horizon read failed");
    }
    const auto signal =
        std::string(reinterpret_cast<const char *>(input->signal_json.data),
                    static_cast<std::size_t>(input->signal_json.size));
    const auto id = field(signal, "signal_id", "");
    const int64_t variant =
        field(i.config, "model_variant", "1") == "2" ? 2 : 1;
    if (anchor.price_nanos < INT64_MIN + variant ||
        anchor.price_nanos > INT64_MAX - variant)
      return error(err, SBV_PROVIDER_UNAVAILABLE,
                   "fixture anchor arithmetic unavailable");
    const auto mode = field(i.config, "mode", "normal");
    const auto left_num = mode == "signed_coefficients" ? "-1" : "1";
    const auto right_den =
        mode == "invalid_probability" || mode == "signed_coefficients" ? "3"
                                                                       : "2";
    output(out, "{\"signal_id\":" + quote(id) +
                    ",\"support\":[{\"price_nanos\":" +
                    quote(std::to_string(anchor.price_nanos - variant)) +
                    ",\"weight\":{\"numerator\":" + quote(left_num) +
                    ",\"denominator\":\"2\"}},{"
                    "\"price_nanos\":" +
                    quote(std::to_string(anchor.price_nanos + variant)) +
                    ",\"weight\":{\"numerator\":\"1\",\"denominator\":" +
                    quote(right_den) +
                    "}}],\"execution_probability\":{\"status\":\"supplied\","
                    "\"value\":{\"numerator\":" +
                    quote(variant == 1 ? "1" : "3") +
                    ",\"denominator\":" + quote(variant == 1 ? "2" : "4") +
                    "}},\"evidence_reference\":\"fixture-full-mbo-prefix-and-"
                    "followup-checked\"}");
    return SBV_PROVIDER_OK;
  });
}
int32_t descriptor(const sbv_call_context_v1 *, sbv_owned_bytes_v1 *out,
                   sbv_owned_bytes_v1 *err) {
  return guarded(err, [&]() -> int32_t {
    output(
        out,
        R"({"protocol":"symphony.sbv.native-provider-descriptor.v1","id":"sbv-test-provider","version":"1","roles":["strategy","model"],"input_profiles":["symphony.sbv.provider-databento-mbo-event.v1"],"config_schema":{"type":"object","description":"Fixture string parameters only","maxProperties":1000},"strategy_concurrency":["serialized_instance"],"model_concurrency":["serialized_instance","per_worker_instances","shared_reentrant_instance"],"reproducibility":"deterministic_declared","cancellation":"cooperative_polling","dependency_disclosure":"Fixture links the platform C++ runtime; report_counters deliberately exposes process-local diagnostics when selected","extensions":{"fixture":"independently compiled without engine/JSON dependency"}})");
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
          SBV_PROVIDER_STRATEGY | SBV_PROVIDER_MODEL,
          0,
          descriptor,
          create,
          strategy_event,
          finish,
          destroy,
          create,
          model,
          destroy};
#if SBV_FIXTURE_BAD_ABI == 1
  api->struct_size -= 1;
#elif SBV_FIXTURE_BAD_ABI == 2
  api->abi_version = 2;
#elif SBV_FIXTURE_BAD_ABI == 3
  api->required_capability_bits = UINT64_C(1);
#elif SBV_FIXTURE_BAD_ABI == 4
  api->strategy_event = nullptr;
#endif
  return SBV_PROVIDER_OK;
}
