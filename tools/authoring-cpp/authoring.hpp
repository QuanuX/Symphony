#pragma once

#include "symphony/knowledge/engine/digest.hpp"
#include "native_test.hpp"
#include <algorithm>
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iostream>
#include <map>
#include <nlohmann/json.hpp>
#include <regex>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string>
#include <sys/wait.h>
#include <unistd.h>
#include <vector>

namespace symphony::authoring {
namespace fs = std::filesystem;
using Json = nlohmann::ordered_json;
using Outputs = std::vector<std::pair<std::string, std::string>>;
inline void require(bool ok, const std::string &message) {
  if (!ok)
    throw std::runtime_error(message);
}
inline std::string read_bytes(const fs::path &path) {
  std::ifstream in(path, std::ios::binary);
  require(bool(in), "cannot read: " + path.string());
  return {std::istreambuf_iterator<char>(in), std::istreambuf_iterator<char>()};
}
inline void write_bytes(const fs::path &path, const std::string &body) {
  if (!path.parent_path().empty())
    fs::create_directories(path.parent_path());
  std::ofstream out(path, std::ios::binary | std::ios::trunc);
  require(bool(out), "cannot write: " + path.string());
  out.write(body.data(), static_cast<std::streamsize>(body.size()));
  require(bool(out), "write failed: " + path.string());
}
inline Json read_json(const fs::path &path,
                      std::size_t maximum = 16 * 1024 * 1024) {
  require(fs::file_size(path) <= maximum, "JSON input exceeds byte limit");
  std::vector<std::set<std::string>> keys;
  return Json::parse(
      read_bytes(path), [&](int, Json::parse_event_t event, Json &value) {
        if (event == Json::parse_event_t::object_start)
          keys.emplace_back();
        if (event == Json::parse_event_t::key)
          require(keys.back().insert(value.get<std::string>()).second,
                  "duplicate JSON key: " + value.get<std::string>());
        if (event == Json::parse_event_t::object_end)
          keys.pop_back();
        return true;
      });
}
inline void write_json(const fs::path &path, const Json &value) {
  write_bytes(path, value.dump(2, ' ', true) + "\n");
}
inline std::string string(const Json &value) {
  require(value.is_string(), "expected string");
  return value.get<std::string>();
}
inline bool same(const Json &a, const Json &b) {
  return nlohmann::json(a) == nlohmann::json(b);
}
inline std::string digest(const Json &value, bool ascii = true) {
  return symphony::knowledge::engine::tagged_sha256(
      nlohmann::json(value).dump(-1, ' ', ascii));
}
inline std::string digest_bytes(const std::string &value) {
  return symphony::knowledge::engine::tagged_sha256(value);
}
inline std::string quote(const Json &value, bool ascii = true) {
  return value.dump(-1, ' ', ascii);
}
inline std::string join(const std::vector<std::string> &values,
                        std::string_view separator) {
  std::string out;
  for (const auto &value : values) {
    if (!out.empty())
      out += separator;
    out += value;
  }
  return out;
}
inline std::vector<std::string> strings(const Json &values) {
  require(values.is_array(), "expected list");
  std::vector<std::string> out;
  for (const auto &value : values)
    out.push_back(string(value));
  return out;
}
inline std::string quoted_join(const Json &values,
                               std::string_view separator = ",",
                               std::string_view suffix = "") {
  std::vector<std::string> out;
  for (const auto &value : values)
    out.push_back(quote(value) + std::string(suffix));
  return join(out, separator);
}
inline bool contains(const Json &values, const Json &value) {
  return values.is_array() &&
         std::find(values.begin(), values.end(), value) != values.end();
}
inline std::size_t ordinal(const Json &values, const Json &value) {
  require(values.is_array(), "expected list");
  const auto it = std::find(values.begin(), values.end(), value);
  require(it != values.end(), "undeclared exact release");
  return static_cast<std::size_t>(std::distance(values.begin(), it));
}
inline Json keys(const Json &object) {
  require(object.is_object(), "expected object");
  Json result = Json::array();
  for (auto it = object.begin(); it != object.end(); ++it)
    result.push_back(it.key());
  return result;
}
inline void exact(const Json &value, const std::string &fields,
                  const std::string &label) {
  require(value.is_object(), "unexpected " + label + " fields");
  std::set<std::string> wanted, actual;
  std::istringstream in(fields);
  std::string field;
  while (in >> field)
    wanted.insert(field);
  for (auto it = value.begin(); it != value.end(); ++it)
    actual.insert(it.key());
  require(actual == wanted, "unexpected " + label + " fields");
}
inline std::string word(const Json &value, const std::string &pattern,
                        const std::string &label) {
  require(value.is_string() &&
              std::regex_match(value.get<std::string>(), std::regex(pattern)),
          "invalid " + label);
  return string(value);
}
inline void unique(const Json &values, const std::string &label) {
  const auto values_list = strings(values);
  require(
      std::set<std::string>(values_list.begin(), values_list.end()).size() ==
          values_list.size(),
      "invalid or duplicate " + label);
}
inline fs::path local(const fs::path &root, const Json &value,
                      bool exists = true) {
  const auto text = word(value, "[A-Za-z0-9_./-]+", "local path");
  const fs::path p(text);
  require(!p.is_absolute() && p.lexically_normal().generic_string() == text,
          "escaping path");
  fs::path q = fs::weakly_canonical(root);
  for (const auto &part : p) {
    require(part != "..", "escaping path");
    q /= part;
    require(!fs::is_symlink(q), "symlink path");
  }
  if (exists)
    require(fs::is_regular_file(q), "missing regular file: " + text);
  return q;
}
inline void local_refs(const Json &doc) {
  std::function<void(const Json &)> walk = [&](const Json &value) {
    if (value.is_object()) {
      if (value.contains("$ref")) {
        const auto ref = string(value.at("$ref"));
        require(ref.starts_with("#/"), "nonlocal schema reference");
        static_cast<void>(doc.at(Json::json_pointer(ref.substr(1))));
      }
      for (const auto &child : value)
        walk(child);
    } else if (value.is_array())
      for (const auto &child : value)
        walk(child);
  };
  walk(doc);
}
struct TempDir {
  fs::path path;
  TempDir() {
    auto name =
        (fs::temp_directory_path() / "symphony-authoring-XXXXXX").string();
    std::vector<char> chars(name.begin(), name.end());
    chars.push_back(0);
    auto p = ::mkdtemp(chars.data());
    require(p != nullptr, "cannot create temporary directory");
    path = p;
  }
  ~TempDir() {
    std::error_code error;
    fs::remove_all(path, error);
  }
  TempDir(const TempDir &) = delete;
  TempDir &operator=(const TempDir &) = delete;
};
struct Process {
  int status;
  std::string output;
  std::string error;
};
inline Process run(const std::vector<std::string> &args,
                   const std::string &input = "") {
  auto result = native_test::run(args, input);
  return {result.returncode, std::move(result.stdout_text),
          std::move(result.stderr_text)};
}

inline std::string gofmt(const std::string &source) {
  const auto result = run({"gofmt"}, source);
  require(result.status == 0, "gofmt failed: " + result.error);
  return result.output;
}
inline void emit(const fs::path &root, const Outputs &outputs, bool check,
                 bool guard_owner = false) {
  for (const auto &[path, body] : outputs) {
    const auto target = local(root, path, false);
    if (check)
      require(fs::is_regular_file(target) && read_bytes(target) == body,
              "generated interface drift: " + path);
    else if (guard_owner && fs::exists(target)) {
      const auto old = read_bytes(target);
      require(old.substr(0, old.find('\n')) == body.substr(0, body.find('\n')),
              "destination belongs to another author");
    }
  }
  if (!check)
    for (const auto &[path, body] : outputs)
      write_bytes(local(root, path, false), body);
}
struct Arguments {
  fs::path root = SYMPHONY_AUTHORING_ROOT;
  bool check = false, metadata_only = false, help = false;
  std::string owner, registration, engine;
  Arguments(int argc, char **argv) {
    for (int i = 1; i < argc; ++i) {
      const std::string key = argv[i];
      if (key == "--check")
        check = true;
      else if (key == "--metadata-only")
        metadata_only = true;
      else if (key == "--help" || key == "-h")
        help = true;
      else {
        require(i + 1 < argc, "missing argument: " + key);
        const std::string value = argv[++i];
        if (key == "--root")
          root = fs::weakly_canonical(value);
        else if (key == "--owner")
          owner = value;
        else if (key == "--registration")
          registration = value;
        else if (key == "--engine")
          engine = value;
        else
          throw std::runtime_error("unknown argument: " + key);
      }
    }
  }
};
template <class Function> int main_guard(Function function) {
  try {
    function();
    return 0;
  } catch (const std::exception &error) {
    std::cerr << "Authoring rejected: " << error.what() << '\n';
    return 1;
  }
}
} // namespace symphony::authoring
