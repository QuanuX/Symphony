#include "shv.hpp"
#include <algorithm>
#include <charconv>
#include <map>
#include <vector>
namespace symphony::knowledge::shv {
namespace {
bool space(unsigned char c) {
  return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f';
}
std::string lower(std::string s) {
  for (auto &c : s)
    if (c >= 'A' && c <= 'Z')
      c = static_cast<char>(c - 'A' + 'a');
  return s;
}
std::string normalize(const std::string &s) {
  std::string out;
  bool pending = false;
  for (unsigned char c : s) {
    if (space(c)) {
      pending = !out.empty();
      continue;
    }
    if (c < 32 || c == 127)
      invalid("control in HTML field");
    if (pending)
      out += ' ';
    out += static_cast<char>(c);
    pending = false;
  }
  return out;
}
std::string utf8(unsigned n) {
  if (n == 0 || n > 0x10ffff || (n >= 0xd800 && n <= 0xdfff))
    invalid("invalid HTML numeric entity");
  std::string s;
  if (n < 0x80)
    s += static_cast<char>(n);
  else if (n < 0x800) {
    s += static_cast<char>(0xc0 | (n >> 6));
    s += static_cast<char>(0x80 | (n & 63));
  } else if (n < 0x10000) {
    s += static_cast<char>(0xe0 | (n >> 12));
    s += static_cast<char>(0x80 | ((n >> 6) & 63));
    s += static_cast<char>(0x80 | (n & 63));
  } else {
    s += static_cast<char>(0xf0 | (n >> 18));
    s += static_cast<char>(0x80 | ((n >> 12) & 63));
    s += static_cast<char>(0x80 | ((n >> 6) & 63));
    s += static_cast<char>(0x80 | (n & 63));
  }
  return s;
}
std::string decode(const std::string &s) {
  static const std::map<std::string, std::string> named = {
      {"amp", "&"},  {"lt", "<"},    {"gt", ">"},    {"quot", "\""},
      {"apos", "'"}, {"nbsp", " "},  {"reg", "®"},   {"trade", "™"},
      {"copy", "©"}, {"ndash", "–"}, {"mdash", "—"}, {"times", "×"},
      {"micro", "µ"}};
  std::string out;
  for (std::size_t i = 0; i < s.size();) {
    if (s[i] != '&') {
      out += s[i++];
      continue;
    }
    auto end = s.find(';', i + 1);
    if (end == s.npos || end - i > 16)
      invalid("unsupported HTML entity");
    auto key = s.substr(i + 1, end - i - 1);
    if (key.starts_with('#')) {
      unsigned n = 0;
      int base = 10;
      std::size_t start = 1;
      if (key.size() > 1 && (key[1] == 'x' || key[1] == 'X')) {
        start = 2;
        base = 16;
      }
      auto [p, e] =
          std::from_chars(key.data() + start, key.data() + key.size(), n, base);
      if (e != std::errc{} || p != key.data() + key.size() ||
          start == key.size())
        invalid("invalid numeric entity");
      out += utf8(n);
    } else {
      auto it = named.find(key);
      if (it == named.end())
        invalid("unsupported named entity");
      out += it->second;
    }
    i = end + 1;
  }
  return normalize(out);
}
} // namespace
// Finite, non-executing extraction profile. The caller names exact, unique
// element IDs for the product heading and specification fields. Content in
// navigation, scripts, and similarly named custom elements is not evidence.
std::string html_text(const std::string &raw, const std::string &model,
                      const std::string &heading_section,
                      const std::string &field_section) {
  if (raw.find('\0') != raw.npos)
    invalid("NUL in HTML");
  try {
    (void)Json(raw)
        .dump(); // Validate the entire UTF-8 source, including ignored text.
  } catch (const Json::exception &) {
    invalid("invalid UTF-8 in HTML source");
  }
  if (heading_section == field_section)
    invalid("HTML section IDs must differ");
  const auto folded = lower(raw);
  std::vector<std::pair<std::string, std::string>> pairs;
  struct Scope {
    std::string id, tag;
    std::size_t depth = 0, count = 0;
  };
  auto section = [](const std::string &selector) {
    auto split = selector.find('#');
    if (split == 0 || split == selector.npos || split + 1 == selector.size() ||
        selector.find('#', split + 1) != selector.npos)
      invalid("section must be exact tag#id");
    auto tag = selector.substr(0, split);
    for (std::size_t i = 0; i < tag.size(); ++i)
      if (!((tag[i] >= 'a' && tag[i] <= 'z') ||
            (i > 0 && ((tag[i] >= '0' && tag[i] <= '9') || tag[i] == '-'))))
        invalid("invalid section tag");
    return Scope{selector.substr(split + 1), tag, 0, 0};
  };
  Scope hs = section(heading_section), fs = section(field_section);
  std::string active, buffer, label, heading;
  std::size_t headings = 0;
  bool waiting = false;
  for (std::size_t pos = 0; pos < raw.size();) {
    if (raw[pos] != '<') {
      auto end = raw.find('<', pos);
      if (end == raw.npos)
        end = raw.size();
      if (!active.empty()) {
        if (buffer.size() + end - pos > 65536)
          invalid("HTML field exceeds bound");
        buffer.append(raw, pos, end - pos);
      }
      pos = end;
      continue;
    }
    if (raw.compare(pos, 4, "<!--") == 0) {
      auto end = raw.find("-->", pos + 4);
      if (end == raw.npos)
        invalid("unterminated HTML comment");
      pos = end + 3;
      continue;
    }
    std::size_t end = pos + 1;
    char quote = 0;
    for (; end < raw.size(); ++end) {
      char c = raw[end];
      if (quote) {
        if (c == quote)
          quote = 0;
      } else if (c == '\'' || c == '\"')
        quote = c;
      else if (c == '>')
        break;
    }
    if (end == raw.size())
      invalid("unterminated HTML tag");
    std::size_t start = pos + 1;
    bool closing = start < end && raw[start] == '/';
    if (closing)
      ++start;
    auto n = start;
    while (n < end && !space(raw[n]) && raw[n] != '/' && raw[n] != '>')
      ++n;
    auto tag = folded.substr(start, n - start);
    pos = end + 1;
    if (tag.empty() || tag.starts_with('!') || tag.starts_with('?'))
      continue;
    bool self_closing = end > start && raw[end - 1] == '/';
    if (!closing && (tag == "script" || tag == "style")) {
      auto close = pos;
      for (;;) {
        close = folded.find("</" + tag, close);
        if (close == raw.npos)
          invalid("unterminated ignored HTML block");
        auto after = close + 2 + tag.size();
        if (after < raw.size() &&
            (space(raw[after]) || raw[after] == '>' || raw[after] == '/'))
          break;
        close = after;
      }
      auto finish = raw.find('>', close);
      if (finish == raw.npos)
        invalid("invalid ignored HTML block");
      pos = finish + 1;
      continue;
    }
    std::string element_id;
    bool has_id = false;
    if (!closing) {
      while (n < end) {
        while (n < end && (space(raw[n]) || raw[n] == '/'))
          ++n;
        if (n == end)
          break;
        auto a = n;
        while (n < end && !space(raw[n]) && raw[n] != '=' && raw[n] != '/')
          ++n;
        auto attr = folded.substr(a, n - a);
        if (attr.empty())
          invalid("invalid HTML attribute");
        while (n < end && space(raw[n]))
          ++n;
        std::string value;
        if (n < end && raw[n] == '=') {
          ++n;
          while (n < end && space(raw[n]))
            ++n;
          if (n == end)
            invalid("missing HTML attribute value");
          if (raw[n] == '\'' || raw[n] == '\"') {
            auto q = raw[n++];
            auto v = n;
            while (n < end && raw[n] != q)
              ++n;
            if (n == end)
              invalid("unterminated HTML attribute");
            value = raw.substr(v, n - v);
            ++n;
          } else {
            auto v = n;
            while (n < end && !space(raw[n]))
              ++n;
            value = raw.substr(v, n - v);
          }
        }
        if (attr == "id") {
          if (has_id)
            invalid("duplicate HTML id attribute");
          has_id = true;
          element_id = value;
        }
      }
      for (auto *scope : {&hs, &fs}) {
        if (scope->depth && tag == scope->tag && !self_closing)
          ++scope->depth;
        if (has_id && element_id == scope->id && tag == scope->tag) {
          if (scope == &hs && fs.depth)
            invalid("heading section inside field section");
          if (++scope->count != 1 || self_closing || tag == "h1" ||
              tag == "dt" || tag == "dd")
            invalid("ambiguous HTML section");
          scope->tag = tag;
          scope->depth = 1;
        }
      }
    }
    const bool selected = (tag == "h1" && hs.depth && !fs.depth) ||
                          (fs.depth && (tag == "dt" || tag == "dd"));
    if (selected) {
      if (!closing) {
        if (!active.empty() || self_closing)
          invalid("nested HTML profile field");
        active = tag;
        buffer.clear();
        if (tag == "dt" && waiting)
          invalid("HTML term missing definition");
        if (tag == "dd" && !waiting)
          invalid("HTML definition without term");
      } else {
        if (active != tag)
          invalid("unmatched HTML profile field");
        auto value = decode(buffer);
        active.clear();
        buffer.clear();
        if (tag == "h1") {
          heading = value;
          ++headings;
        } else if (tag == "dt") {
          if (value.empty())
            invalid("empty HTML term");
          label = value;
          waiting = true;
        } else {
          pairs.emplace_back(label, value);
          waiting = false;
          if (pairs.size() > 256)
            invalid("too many HTML fields");
        }
      }
    } else if (!active.empty())
      buffer += ' ';
    if (closing)
      for (auto *scope : {&hs, &fs})
        if (scope->depth && tag == scope->tag) {
          if (--scope->depth == 0 && (!active.empty() || waiting))
            invalid("unclosed HTML section field");
        }
  }
  if (hs.depth || fs.depth || hs.count != 1 || fs.count != 1 ||
      !active.empty() || waiting || headings != 1 || heading != model)
    invalid("exact product sections, heading or paired fields unavailable");
  std::set<std::string> seen;
  std::string out;
  for (const auto &[key, value] : pairs) {
    if (!seen.insert(key).second)
      invalid("duplicate HTML field label");
    out += key;
    out += '\n';
    out += value;
    out += '\n';
  }
  return out;
}
} // namespace symphony::knowledge::shv
