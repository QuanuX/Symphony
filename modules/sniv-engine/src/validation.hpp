#pragma once
#include <algorithm>
#include <chrono>
#include <limits>
#include <set>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/temporal.hpp>
#include <symphony/snv/common.hpp>
#include <vector>
namespace symphony::snv::owner_validation {
namespace kve = symphony::knowledge::engine;
[[noreturn]] inline void fail(std::string code, std::string message) {
  throw kve::Error(std::move(code), std::move(message), 2);
}
inline void require(bool okay, std::string_view message) {
  if (!okay)
    fail("invalid_input", std::string(message));
}
inline void keys(const Json &j, std::initializer_list<std::string_view> names) {
  require(j.is_object(), "Expected an object");
  require(j.size() == names.size(), "Missing or unknown object field");
  for (auto name : names)
    require(j.contains(std::string(name)), "Missing required object field");
}
inline bool utf8(std::string_view s) {
  for (std::size_t i = 0; i < s.size();) {
    auto b = static_cast<unsigned char>(s[i]);
    if (b < 0x20 || b == 0x7f)
      return false;
    if (b < 0x80) {
      ++i;
      continue;
    }
    unsigned n = 0;
    std::uint32_t cp = 0;
    if (b >= 0xc2 && b <= 0xdf) {
      n = 2;
      cp = b & 31;
    } else if (b >= 0xe0 && b <= 0xef) {
      n = 3;
      cp = b & 15;
    } else if (b >= 0xf0 && b <= 0xf4) {
      n = 4;
      cp = b & 7;
    } else
      return false;
    if (i + n > s.size())
      return false;
    for (unsigned k = 1; k < n; ++k) {
      auto c = static_cast<unsigned char>(s[i + k]);
      if ((c & 0xc0) != 0x80)
        return false;
      cp = (cp << 6) | (c & 63);
    }
    if ((n == 3 && cp < 0x800) || (n == 4 && cp < 0x10000) ||
        (cp >= 0xd800 && cp <= 0xdfff) || cp > 0x10ffff)
      return false;
    i += n;
  }
  return true;
}
inline std::string str(const Json &j, std::size_t max = 256) {
  require(j.is_string(), "Expected a string");
  const auto &s = j.get_ref<const std::string &>();
  require(!s.empty() && s.size() <= max && utf8(s),
          "Invalid bounded UTF-8 text");
  return s;
}
inline std::string text(const Json &j, std::string_view key,
                        std::size_t max = 256) {
  return str(j.at(std::string(key)), max);
}
inline void nullable_text(const Json &j) {
  if (!j.is_null())
    (void)str(j);
}
inline std::uint64_t integer(const Json &j,
                             std::uint64_t maximum = 9007199254740991ULL) {
  require(j.is_number_integer() &&
              (j.is_number_unsigned() || j.get<std::int64_t>() >= 0),
          "Expected an unsigned safe integer");
  auto value = j.get<std::uint64_t>();
  require(value <= maximum, "Integer is out of range");
  return value;
}
inline void choice(const Json &j,
                   std::initializer_list<std::string_view> names) {
  auto s = str(j);
  require(std::find(names.begin(), names.end(), s) != names.end(),
          "Unsupported enum value");
}
inline const Json &array(const Json &j, std::size_t maximum = 2048) {
  require(j.is_array() && j.size() <= maximum, "Expected bounded array");
  return j;
}
inline void unique_strings(const Json &j, std::size_t maximum = 2048) {
  std::set<std::string> seen;
  for (const auto &v : array(j, maximum))
    require(seen.insert(str(v)).second, "Duplicate identifier");
}
inline void timestamp(const Json &j) {
  if (!j.is_null())
    require(kve::is_utc_seconds(str(j, 20)),
            "Timestamp must be whole-second UTC");
}
inline bool digest_string(const std::string &s) {
  if (s.size() != 71 || s.substr(0, 7) != "sha256:")
    return false;
  return std::all_of(s.begin() + 7, s.end(), [](char c) {
    return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
  });
}
inline void source(const Json &j) {
  keys(j, {"source_id", "kind", "source_ref", "source_revision", "observed_at",
           "evidence_digest", "method_id", "method_version"});
  (void)text(j, "source_id");
  (void)text(j, "source_ref", 1024);
  choice(j.at("kind"),
         {"provider", "firmware", "probe", "user", "artifact", "shv", "scv"});
  nullable_text(j.at("source_revision"));
  timestamp(j.at("observed_at"));
  require(digest_string(text(j, "evidence_digest", 71)),
          "Evidence digest must be lowercase tagged SHA-256");
  (void)text(j, "method_id");
  (void)text(j, "method_version");
}
inline std::set<std::string> sources(const Json &j) {
  std::set<std::string> seen;
  for (const auto &s : array(j, 128)) {
    source(s);
    require(seen.insert(text(s, "source_id")).second,
            "Duplicate source identifier");
  }
  return seen;
}
inline void references(const Json &j, const std::set<std::string> &seen,
                       bool nonempty = true) {
  unique_strings(j, 128);
  require(!nonempty || !j.empty(), "Evidence reference list is empty");
  for (const auto &id : j)
    require(seen.contains(str(id)), "Unknown source reference");
}
inline void deadline(std::int64_t deadline_unix_ms) {
  auto now = std::chrono::duration_cast<std::chrono::milliseconds>(
                 std::chrono::system_clock::now().time_since_epoch())
                 .count();
  if (deadline_unix_ms <= now)
    throw kve::Error("deadline_exceeded", "Native owner deadline expired", 3);
}
inline std::size_t limit(const Json &j) {
  auto n = integer(j, 2048);
  require(n == 512 || n == 1024 || n == 2048,
          "Record limit must be 512, 1024 or 2048");
  return static_cast<std::size_t>(n);
}
inline std::string digest(const Json &j) {
  try {
    return kve::tagged_sha256(j.dump());
  } catch (const nlohmann::json::exception &) {
    fail("invalid_input", "Invalid JSON representation");
  }
}
inline Json finding(std::string_view code, const Json &refs) {
  return {{"code", code}, {"record_refs", refs}};
}
inline Json ids(const std::set<std::string> &values) {
  Json out = Json::array();
  for (const auto &v : values)
    out.push_back(v);
  return out;
}
} // namespace symphony::snv::owner_validation
