#include "shv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <cctype>
#include <chrono>
namespace symphony::knowledge::shv {
[[noreturn]] void invalid(const std::string &why) {
  throw engine::Error("shv.invalid", why, 4);
}
void fields(const Json &j, std::initializer_list<const char *> names) {
  if (!j.is_object() || j.size() != names.size())
    invalid("unexpected object fields");
  for (auto n : names)
    if (!j.contains(n))
      invalid(std::string("missing field: ") + n);
}
std::string text(const Json &j, const char *k, std::size_t m) {
  if (!j.contains(k) || !j.at(k).is_string())
    invalid(std::string("expected string: ") + k);
  auto s = j.at(k).get<std::string>();
  if (s.empty() || s.size() > m || s.find('\0') != s.npos)
    invalid("text outside bounds");
  return s;
}
std::string ident(const Json &j, const char *k) {
  auto s = text(j, k, 128);
  if (!std::all_of(s.begin(), s.end(), [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
               (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-';
      }))
    invalid("identifier outside grammar");
  return s;
}
void array(const Json &j, std::size_t n) {
  if (!j.is_array() || j.size() > n)
    invalid("array outside bounds");
}
Json seal(Json j, const std::string &k) {
  j.erase(k);
  j[k] = engine::tagged_sha256(j.dump());
  return j;
}
void check_seal(const Json &j) {
  if (!j.is_object() || !j.contains("digest") || seal(j) != j)
    invalid("artifact seal mismatch");
}
void deadline(const engine::Request &r) {
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    throw engine::Error("request.deadline", "deadline exceeded", 4);
}
bool date_valid(const std::string &s) {
  if (s.size() != 10 || s[4] != '-' || s[7] != '-')
    return false;
  for (std::size_t i = 0; i < s.size(); ++i)
    if (i != 4 && i != 7 && (s[i] < '0' || s[i] > '9'))
      return false;
  int y = std::stoi(s.substr(0, 4)), m = std::stoi(s.substr(5, 2)),
      d = std::stoi(s.substr(8, 2));
  return y >= 1 &&
         std::chrono::year_month_day{
             std::chrono::year{y}, std::chrono::month{static_cast<unsigned>(m)},
             std::chrono::day{static_cast<unsigned>(d)}}
             .ok();
}
void summary(const Json &s) {
  fields(s, {"id", "manufacturer", "model", "hardware_class", "introduced"});
  ident(s, "id");
  ident(s, "hardware_class");
  text(s, "manufacturer", 256);
  text(s, "model", 256);
  if (!s.at("introduced").is_null()) {
    const auto &d = s.at("introduced");
    fields(d, {"from", "through"});
    auto a = text(d, "from", 10), b = text(d, "through", 10);
    if (!date_valid(a) || !date_valid(b) || a > b)
      invalid("invalid introduction interval");
  }
}
} // namespace symphony::knowledge::shv
