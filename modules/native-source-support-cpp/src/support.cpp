#include <algorithm>
#include <chrono>
#include <set>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/source/json.hpp>
namespace symphony::source {
bool valid_limits(const JsonLimits &l) noexcept {
  return l.bytes > 0 && l.bytes <= (64U << 20) && l.values > 0 &&
         l.values <= 1048576 && l.string_bytes > 0 &&
         l.string_bytes <= l.bytes && l.depth > 0 && l.depth <= 64;
}
bool token(std::string_view s, std::size_t max) noexcept {
  return !s.empty() && s.size() <= max &&
         std::ranges::all_of(s, [](unsigned char c) {
           return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                  (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' ||
                  c == ':';
         });
}
bool date(std::string_view s) noexcept {
  if (s.size() != 10 || s[4] != '-' || s[7] != '-')
    return false;
  unsigned v[3]{};
  unsigned part = 0;
  for (std::size_t i = 0; i < s.size(); ++i) {
    if (i == 4 || i == 7) {
      ++part;
      continue;
    }
    if (s[i] < '0' || s[i] > '9')
      return false;
    v[part] = v[part] * 10 + unsigned(s[i] - '0');
  }
  return v[0] >= 1 && v[0] <= 9999 &&
         std::chrono::year_month_day{std::chrono::year(int(v[0])),
                                     std::chrono::month(v[1]),
                                     std::chrono::day(v[2])}
             .ok();
}
bool decimal(std::string_view s) noexcept {
  if (s.empty() || s.size() > 128)
    return false;
  std::size_t p = (s[0] == '-' || s[0] == '+') ? 1 : 0;
  bool digits = false;
  while (p < s.size() && s[p] >= '0' && s[p] <= '9') {
    digits = true;
    ++p;
  }
  if (p < s.size() && s[p] == '.') {
    ++p;
    while (p < s.size() && s[p] >= '0' && s[p] <= '9') {
      digits = true;
      ++p;
    }
  }
  if (!digits)
    return false;
  if (p < s.size() && (s[p] == 'e' || s[p] == 'E')) {
    ++p;
    if (p < s.size() && (s[p] == '-' || s[p] == '+'))
      ++p;
    auto b = p;
    while (p < s.size() && s[p] >= '0' && s[p] <= '9')
      ++p;
    if (p == b)
      return false;
  }
  return p == s.size();
}
std::string encode(std::string_view s) {
  constexpr char h[] = "0123456789ABCDEF";
  std::string o;
  for (unsigned char c : s) {
    if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
        (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~')
      o += char(c);
    else {
      o += '%';
      o += h[c >> 4];
      o += h[c & 15];
    }
  }
  return o;
}
std::string digest(std::string_view s) {
  return symphony::knowledge::engine::sha256_hex(s);
}
std::string HttpRequest::reference() const {
  return "source-request-v1-" +
         digest(endpoint + "\n" + (post ? "POST" : "GET") + "\n" +
                std::to_string(static_cast<unsigned>(credential)) + "\n" +
                parameters);
}
Status parse_json(Bytes b, const JsonLimits &l, nlohmann::json &out) noexcept {
  if (!valid_limits(l))
    return Status::invalid_argument;
  if (b.size() > l.bytes)
    return Status::limit;
  if (b.empty())
    return Status::malformed;
  try {
    std::vector<std::set<std::string>> keys;
    std::uint32_t count = 0;
    auto cb = [&](int depth, nlohmann::json::parse_event_t event,
                  nlohmann::json &parsed) {
      using E = nlohmann::json::parse_event_t;
      if (depth < 0 || static_cast<unsigned>(depth) > l.depth)
        throw Status::limit;
      if (event == E::object_start || event == E::array_start ||
          event == E::key || event == E::value)
        if (++count > l.values)
          throw Status::limit;
      if (event == E::object_start)
        keys.emplace_back();
      if (event == E::object_end)
        keys.pop_back();
      if (parsed.is_string() &&
          parsed.get_ref<const std::string &>().size() > l.string_bytes)
        throw Status::limit;
      if (event == E::key) {
        if (keys.empty() ||
            !keys.back().insert(parsed.get<std::string>()).second)
          throw Status::malformed;
      }
      return true;
    };
    auto parsed = nlohmann::json::parse(b.begin(), b.end(), cb, true, false);
    out = std::move(parsed);
    return Status::ok;
  } catch (Status s) {
    return s;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::source
