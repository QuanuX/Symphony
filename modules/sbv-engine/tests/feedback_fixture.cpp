// Independently linked provider fixture: public C ABI plus a header-only
// parser.
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <nlohmann/json.hpp>
#include <string>
#include <symphony/sbv/feedback.h>
#include <thread>
using J = nlohmann::json;
namespace {
void log(const char *s) noexcept {
  try {
    if (auto p = std::getenv("SBV_FEEDBACK_LIFETIME_LOG"))
      std::ofstream(p, std::ios::app) << s << '\n';
  } catch (...) {
  }
}
void release(void *, const uint8_t *p, uint64_t) {
  log("release");
  std::free(const_cast<uint8_t *>(p));
}
void output(sbv_owned_bytes_v1 *out, const std::string &s) {
  auto p = static_cast<uint8_t *>(std::malloc(s.size()));
  if (!p)
    throw std::bad_alloc();
  std::memcpy(p, s.data(), s.size());
  *out = {sizeof(*out), 1, p, s.size(), nullptr, release};
}
J parse(const sbv_bytes_v1 *p) { return J::parse(p->data, p->data + p->size); }
int32_t error(sbv_owned_bytes_v1 *out, int32_t status) {
  output(out, J{{"code", "feedback.fixture"},
                {"message", "explicit fixture failure"},
                {"details", J::object()}}
                  .dump());
  return status;
}
template <class F> int32_t guard(sbv_owned_bytes_v1 *err, F fn) noexcept {
  try {
    return fn();
  } catch (...) {
    try {
      return error(err, SBV_PROVIDER_FAILED);
    } catch (...) {
      return SBV_PROVIDER_FAILED;
    }
  }
}
int32_t descriptor(const sbv_call_context_v1 *, sbv_owned_bytes_v1 *out,
                   sbv_owned_bytes_v1 *err) {
  return guard(err, [&]() -> int32_t {
    output(out,
           J{{"protocol", "symphony.sbv.native-feedback-descriptor.v1"},
             {"id", "sbv-feedback-fixture"},
             {"version", "1"},
             {"role", "feedback"},
             {"input_profile", "symphony.sbv.account-feedback-call.v1"},
             {"concurrency", "serialized_instance"},
             {"state_model", "explicit_returned_memory"},
             {"config_schema", J::object()},
             {"reproducibility", "deterministic_declared"},
             {"cancellation", "cooperative_polling"},
             {"dependency_disclosure",
              "Fixture links only platform runtime; JSON header compiled in"},
             {"extensions", J::object()}}
               .dump());
    return SBV_PROVIDER_OK;
  });
}
int32_t create(const sbv_call_context_v1 *, const sbv_bytes_v1 *config,
               void **instance, sbv_owned_bytes_v1 *err) {
  return guard(err, [&]() -> int32_t {
    auto j = parse(config);
    *instance = new J(j);
    log("create");
    if (j.value("fail_create", false))
      return error(err, SBV_PROVIDER_FAILED);
    return SBV_PROVIDER_OK;
  });
}
void destroy(void *p) {
  log("destroy");
  delete static_cast<J *>(p);
}
int32_t decide(void *, const sbv_call_context_v1 *ctx,
               const sbv_bytes_v1 *input, sbv_owned_bytes_v1 *out,
               sbv_owned_bytes_v1 *err) {
  return guard(err, [&]() -> int32_t {
    auto call = parse(input);
    auto context = call.at("context");
    auto mode = context.value("mode", std::string{});
    if (mode == "job_barrier") {
      std::ofstream(context.at("entered").get<std::string>()) << "entered\n";
      while (!std::ifstream(context.at("release").get<std::string>()).good()) {
        if (ctx->cancel_state(ctx->token) != SBV_CALL_RUNNING)
          return error(err, SBV_PROVIDER_CANCELLED);
        std::this_thread::sleep_for(std::chrono::milliseconds(2));
      }
    }
    if (mode == "wait_cancel") {
      while (ctx->cancel_state(ctx->token) == SBV_CALL_RUNNING)
        std::this_thread::yield();
      return error(err, SBV_PROVIDER_CANCELLED);
    }
    if (mode == "unavailable")
      return error(err, SBV_PROVIDER_UNAVAILABLE);
    if (mode == "failed")
      return error(err, SBV_PROVIDER_FAILED);
    if (mode == "cancelled")
      return error(err, SBV_PROVIDER_CANCELLED);
    if (mode == "malformed") {
      output(out, "{");
      return SBV_PROVIDER_OK;
    }
    if (call.size() != 7 || call.at("state").size() != 5 ||
        call.at("protocol") != "symphony.sbv.account-feedback-call.v1")
      return error(err, SBV_PROVIDER_INVALID_INPUT);
    auto q = [](const std::string &n) {
      return J{{"numerator", n}, {"denominator", "1"}};
    };
    auto position = call.at("state").at("positions").at("A");
    J action = {{"event_id", call.at("occurrence").at("event_id")},
                {"time_ns", call.at("time_ns")},
                {"kind", "fill_intent"},
                {"instrument_id", "A"},
                {"quantity", q(position.at("numerator") == "0" ? "1" : "-1")},
                {"price", q("10")},
                {"fee", q("0")},
                {"guards", J::array()}};
    if (context.contains("action"))
      action = context.at("action");
    if (mode == "no_action")
      action = nullptr;
    if (mode == "wrong_time")
      action["time_ns"] = "999";
    if (mode == "recursive")
      action = {{"event_id", call.at("occurrence").at("event_id")},
                {"time_ns", call.at("time_ns")},
                {"kind", "feedback_trigger"},
                {"context", J::object()}};
    auto count =
        std::stoull(call.at("memory").value("calls", std::string("0"))) + 1;
    J memory = {{"calls", std::to_string(count)},
                {"prior_position", position},
                {"previous_disposition",
                 call.at("previous_event").is_null()
                     ? J(nullptr)
                     : call.at("previous_event").at("disposition")}};
    if (mode == "bad_memory")
      memory = J::array();
    output(out, J{{"protocol", "symphony.sbv.feedback-decision.v1"},
                  {"action", action},
                  {"memory", memory},
                  {"extensions", J::object()}}
                    .dump());
    return SBV_PROVIDER_OK;
  });
}
} // namespace
extern "C" SBV_PROVIDER_EXPORT int32_t
symphony_sbv_feedback_api_v1(sbv_feedback_api_v1 *p) {
  if (!p || p->struct_size != sizeof(*p) || p->abi_version != 1)
    return SBV_PROVIDER_INVALID_INPUT;
  *p = {sizeof(*p), 1, 0, descriptor, create, decide, destroy};
#ifdef SBV_FEEDBACK_BAD_ABI
  p->abi_version = 2;
#endif
  return SBV_PROVIDER_OK;
}
