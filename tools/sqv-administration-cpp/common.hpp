#pragma once
#include <algorithm>
#include <array>
#include <charconv>
#include <cstdint>
#include <initializer_list>
#include <limits>
#include <span>
#include <string>
#include <string_view>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <vector>
namespace symphony::sqv_admin {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
[[noreturn]] inline void
refuse(const char *code = "sqv.admin.invalid",
       const char *message = "SQV administration contract refused",
       int status = 2) {
  throw engine::Error(code, message, status);
}
inline void require(bool b) {
  if (!b)
    refuse();
}
inline void fields(const Json &j, std::initializer_list<const char *> names) {
  require(j.is_object() && j.size() == names.size());
  for (const auto *name : names)
    require(j.contains(name));
}
inline std::string text(const Json &j, const char *k,
                        std::size_t maximum = 4096) {
  const auto &v = j.at(k);
  require(v.is_string() && v.get_ref<const std::string &>().size() <= maximum);
  return v.get<std::string>();
}
// Decimal strings preserve the whole unsigned 64-bit domain in every client.
inline std::uint64_t decimal(std::string_view s) {
  require(!s.empty() && s.size() <= 20 && (s.size() == 1 || s.front() != '0'));
  for (const char c : s)
    require(c >= '0' && c <= '9');
  std::uint64_t n = 0;
  const auto r = std::from_chars(s.data(), s.data() + s.size(), n);
  require(r.ec == std::errc{} && r.ptr == s.data() + s.size());
  return n;
}
inline std::uint64_t u64(const Json &j, const char *k) {
  return decimal(text(j, k, 20));
}
template <class T> T narrow(std::uint64_t n) {
  require(n <= std::numeric_limits<T>::max());
  return static_cast<T>(n);
}
inline bool boolean(const Json &j, const char *k) {
  require(j.at(k).is_boolean());
  return j.at(k).get<bool>();
}
inline std::string hex(std::string_view raw) {
  constexpr char digits[] = "0123456789abcdef";
  std::string out;
  out.reserve(raw.size() * 2);
  for (const unsigned char c : raw) {
    out.push_back(digits[c >> 4]);
    out.push_back(digits[c & 15]);
  }
  return out;
}
inline std::string hex(std::span<const std::uint8_t> raw) {
  return hex(
      std::string_view(reinterpret_cast<const char *>(raw.data()), raw.size()));
}
inline Json hex_chunks(std::string_view raw) {
  Json out = Json::array();
  for (std::size_t at = 0; at < raw.size(); at += 16384)
    out.push_back(hex(raw.substr(at, 16384)));
  return out;
}
inline std::string unhex(std::string_view s, std::size_t maximum) {
  require(s.size() % 2 == 0 && s.size() / 2 <= maximum);
  std::string out;
  out.reserve(s.size() / 2);
  auto digit = [](char c) -> unsigned {
    if (c >= '0' && c <= '9')
      return static_cast<unsigned>(c - '0');
    if (c >= 'a' && c <= 'f')
      return static_cast<unsigned>(c - 'a' + 10);
    refuse();
  };
  for (std::size_t i = 0; i < s.size(); i += 2)
    out.push_back(static_cast<char>(digit(s[i]) * 16 + digit(s[i + 1])));
  return out;
}
inline std::string bytes(const Json &j, const char *k,
                         std::size_t maximum = 4096) {
  return unhex(text(j, k, maximum * 2), maximum);
}
template <std::size_t N>
std::array<std::uint8_t, N> fixed(const Json &j, const char *k) {
  const auto s = bytes(j, k, N);
  require(s.size() == N);
  std::array<std::uint8_t, N> out{};
  std::copy(s.begin(), s.end(), out.begin());
  return out;
}
template <class Status> void accepted(Status s) {
  if (s == Status::ok)
    return;
  if (s == Status::no_memory || s == Status::internal_error)
    refuse("sqv.admin.unavailable", "Native administration unavailable", 5);
  refuse("sqv.admin.rejected", "Native administration validation refused");
}
inline Json seal(Json j, const char *field) {
  j[field] = engine::tagged_sha256(j.dump());
  return j;
}
inline void deadline(std::int64_t end) {
  if (engine::unix_time_ms() >= end)
    refuse("request.deadline_expired", "SQV administration deadline expired",
           3);
}
} // namespace symphony::sqv_admin
