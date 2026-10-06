#pragma once
#include <charconv>
#include <set>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
namespace symphony::sbv::detail {
namespace e = knowledge::engine;
inline constexpr std::size_t artifact_bytes = 128U << 20,
                             artifact_values = 4000000;
inline void need(bool b, const char *message) {
  if (!b)
    throw e::Error("sbv.contract", message, 2);
}
inline void deadline(std::int64_t d) {
  if (d != e::no_deadline && e::unix_time_ms() >= d)
    throw e::Error("deadline.exceeded", "SBV deadline exceeded", 3);
}
inline std::string process_response(Json response) {
  response["protocol"] = e::process_protocol_v2;
  return e::serialize_response(std::move(response));
}
inline void keys(const Json &v, std::initializer_list<const char *> names) {
  need(v.is_object() && v.size() == names.size(), "unexpected object fields");
  for (auto n : names)
    need(v.contains(n), "required field missing");
}
inline void keys_optional(const Json &v,
                          std::initializer_list<const char *> required,
                          std::initializer_list<const char *> optional) {
  need(v.is_object(), "object required");
  for (auto name : required)
    need(v.contains(name), "required field missing");
  for (const auto &[name, value] : v.items()) {
    (void)value;
    bool allowed = false;
    for (auto key : required)
      allowed |= name == key;
    for (auto key : optional)
      allowed |= name == key;
    need(allowed, "unexpected object field");
  }
}
inline std::string str(const Json &v) {
  need(v.is_string(), "string required");
  return v.get<std::string>();
}
inline std::uint64_t u64(const Json &v) {
  const auto s = str(v);
  std::uint64_t n{};
  auto [p, ec] = std::from_chars(s.data(), s.data() + s.size(), n);
  need(!s.empty() && (s.size() == 1 || s[0] != '0') && ec == std::errc{} &&
           p == s.data() + s.size(),
       "canonical unsigned decimal required");
  return n;
}
inline std::int64_t i64(const Json &v) {
  const auto s = str(v);
  std::int64_t n{};
  auto [p, ec] = std::from_chars(s.data(), s.data() + s.size(), n);
  need(ec == std::errc{} && p == s.data() + s.size() && std::to_string(n) == s,
       "canonical signed decimal required");
  return n;
}
inline std::string dec(auto n) { return std::to_string(n); }
inline Json section(Json v, const std::string &status = "available",
                    const std::string &reason = "") {
  return {{"status", status}, {"reason", reason}, {"data", std::move(v)}};
}
inline Json missing(const std::string &why) {
  return {{"status", "unavailable"}, {"reason", why}, {"value", nullptr}};
}
inline Json base(const std::string &origin) {
  Json s = Json::object();
  for (auto name : {"summary", "signals", "execution", "distributions",
                    "studies", "replay", "comparisons", "search", "resources",
                    "diagnostics", "choices", "provenance"})
    s[name] = section(nullptr, "not_selected",
                      "not selected or implemented by this operation");
  return {{"protocol", result_protocol},
          {"origin", origin},
          {"status", "completed"},
          {"sections", s}};
}
std::string read_file(const std::string &, std::int64_t);
void require_new_file(const std::string &);
void create_file(const std::string &, const std::string &, std::int64_t);
Json persist(Json, const Json &, const std::string &, std::int64_t);
struct Dataset;
Json run(const Json &, std::int64_t, const Dataset * = nullptr);
Json dataset_control(const std::string &, const Json &, std::int64_t);
int resident_worker();
Json compose(const Json &, std::int64_t);
Json compose_joint(const Json &, std::int64_t);
Json evaluate(const Json &, std::int64_t, const Dataset * = nullptr);
Json fit(const Json &, std::int64_t);
Json predict(const Json &, std::int64_t);
Json split(const Json &, std::int64_t);
Json resample(const Json &, std::int64_t);
Json experiment(const Json &, std::int64_t);
Json analyze(const Json &, std::int64_t);
Json compare(const Json &, std::int64_t);
Json result_select(const Json &, std::int64_t);
Json backend_plan(const Json &, std::int64_t);
Json live_plan(const Json &, std::int64_t);
Json allocation_economics(const Json &, std::int64_t);
Json liquidity(const Json &, std::int64_t);
Json book(const Json &, std::int64_t, const Dataset * = nullptr);
Json economics(const Json &, std::int64_t);
Json economic_studies();
Json economic_transforms();
} // namespace symphony::sbv::detail
