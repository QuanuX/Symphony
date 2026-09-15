#pragma once
#include "installed_campaign.hpp"
#include <cmath>
#include <ctime>

namespace scv_maintenance {
using namespace scv_test;
inline constexpr const char *VERSION = "0.10.0-dev";
inline std::vector<std::string> split(const std::string &text) {
  std::istringstream input(text);
  std::vector<std::string> result;
  std::string word;
  while (input >> word)
    result.push_back(word);
  return result;
}
inline J without(J value, std::initializer_list<const char *> keys) {
  for (auto key : keys)
    value.erase(key);
  return value;
}
inline std::string sha(const std::string &bytes) {
  return nt::digest(bytes).substr(7);
}
inline std::string encoded(const J &value) { return value.dump(2) + "\n"; }
inline double elapsed(std::chrono::steady_clock::time_point start) {
  return std::round(std::chrono::duration<double>(
                        std::chrono::steady_clock::now() - start)
                        .count() *
                    1000000.0) /
         1000000.0;
}
inline J read_unique(const fs::path &path) {
  std::vector<std::set<std::string>> objects;
  return J::parse(nt::read(path), [&](int, J::parse_event_t event, J &value) {
    if (event == J::parse_event_t::object_start)
      objects.emplace_back();
    else if (event == J::parse_event_t::key) {
      auto key = value.get<std::string>();
      nt::require(objects.back().insert(key).second,
                  "duplicate JSON key: " + key);
    } else if (event == J::parse_event_t::object_end)
      objects.pop_back();
    return true;
  });
}
using Instant = std::chrono::sys_time<std::chrono::microseconds>;
inline Instant instant(const std::string &text) {
  nt::require(text.size() >= 20, "invalid ISO timestamp");
  std::tm tm{};
  std::istringstream input(text.substr(0, 19));
  input >> std::get_time(&tm, "%Y-%m-%dT%H:%M:%S");
  NT_REQUIRE(!input.fail());
  auto date = std::chrono::year(tm.tm_year + 1900) / unsigned(tm.tm_mon + 1) /
              unsigned(tm.tm_mday);
  NT_REQUIRE(date.ok());
  NT_REQUIRE(tm.tm_hour >= 0 && tm.tm_hour <= 23 && tm.tm_min >= 0 &&
             tm.tm_min <= 59 && tm.tm_sec >= 0 && tm.tm_sec <= 59);
  auto result =
      Instant(std::chrono::sys_days(date)) + std::chrono::hours(tm.tm_hour) +
      std::chrono::minutes(tm.tm_min) + std::chrono::seconds(tm.tm_sec);
  std::size_t pos = 19;
  if (pos < text.size() && text[pos] == '.') {
    ++pos;
    std::string fraction;
    while (pos < text.size() &&
           std::isdigit(static_cast<unsigned char>(text[pos])))
      fraction += text[pos++];
    NT_REQUIRE(!fraction.empty());
    fraction.resize(6, '0');
    result += std::chrono::microseconds(std::stoll(fraction.substr(0, 6)));
  }
  if (text.substr(pos) == "Z")
    return result;
  NT_REQUIRE(text.size() == pos + 6 && (text[pos] == '+' || text[pos] == '-') &&
             text[pos + 3] == ':');
  int hours = std::stoi(text.substr(pos + 1, 2)),
      minutes = std::stoi(text.substr(pos + 4, 2));
  NT_REQUIRE(hours <= 23 && minutes <= 59);
  auto offset = std::chrono::hours(hours) + std::chrono::minutes(minutes);
  return text[pos] == '+' ? result - offset : result + offset;
}
inline std::string stamp(Instant value, bool utc_offset = false) {
  auto seconds = std::chrono::floor<std::chrono::seconds>(value);
  auto micros =
      std::chrono::duration_cast<std::chrono::microseconds>(value - seconds)
          .count();
  auto ticks = std::chrono::system_clock::to_time_t(seconds);
  std::tm tm{};
  NT_REQUIRE(::gmtime_r(&ticks, &tm) != nullptr);
  std::ostringstream out;
  out << std::put_time(&tm, "%Y-%m-%dT%H:%M:%S");
  if (micros)
    out << '.' << std::setw(6) << std::setfill('0') << micros;
  out << (utc_offset ? "+00:00" : "Z");
  return out.str();
}
inline std::string utc() {
  return stamp(Instant(std::chrono::floor<std::chrono::seconds>(
      std::chrono::system_clock::now())));
}
inline std::string now() {
  return stamp(std::chrono::time_point_cast<std::chrono::microseconds>(
                   std::chrono::system_clock::now()),
               true);
}
inline J claim(const std::string &id, const std::string &subject,
               const std::string &predicate, const J &scope, const J &value) {
  return {{"claim_id", id},
          {"subject", subject},
          {"predicate", predicate},
          {"scope", scope},
          {"statement_kind", "documented_fact"},
          {"dependencies", array()},
          {"value",
           {{"type", value.is_boolean() ? "boolean" : "string"},
            {"value", value},
            {"unit", nullptr}}}};
}
inline J valkey_scope() {
  return {{"provider", "do"},
          {"service", "app-platform"},
          {"component", "development-database"},
          {"engine", "valkey"}};
}
// Independently authored meanings from the retained source sentences. Never
// populated from native output or historical profile expectations.
inline J limits_expected() {
  return J::array(
      {claim("do-app-platform-amd64", "do.app-platform.container",
             "image.architecture",
             {{"provider", "do"},
              {"service", "app-platform"},
              {"operating-system", "linux"}},
             "amd64"),
       claim("do-app-platform-local-storage", "do.app-platform.container",
             "local-storage.persistent",
             {{"provider", "do"},
              {"service", "app-platform"},
              {"storage", "host-local-filesystem"}},
             false),
       claim("do-limits-dev-valkey", "do.app-platform.development-database",
             "engine.valkey.available", valkey_scope(), false)});
}
inline J labs_expected() {
  return J::array(
      {claim("do-guide-dev-valkey", "do.app-platform.development-database",
             "engine.valkey.available", valkey_scope(), true)});
}
inline J expected_claims() {
  J values = object();
  for (auto value : limits_expected()) {
    value["valid_from"] = nullptr;
    value["valid_until"] = nullptr;
    values[value["claim_id"].get<std::string>()] = value;
  }
  return values;
}
inline J expected_values() {
  J values = object();
  auto claims = expected_claims();
  for (const auto &[key, value] : claims.items())
    values[key] = value["value"];
  return values;
}
inline J projected(const J &claims) {
  J result = array();
  for (const auto &c : claims)
    result.push_back(pick(c, {"claim_id", "subject", "predicate", "scope",
                              "statement_kind", "value", "dependencies"}));
  std::sort(result.begin(), result.end(), [](const J &a, const J &b) {
    return a["claim_id"].get<std::string>() < b["claim_id"].get<std::string>();
  });
  return result;
}
inline J by_digest(const J &objects) {
  J result = object();
  for (const auto &[key, value] : objects.items())
    result[key] = value["digest"];
  return result;
}
} // namespace scv_maintenance
