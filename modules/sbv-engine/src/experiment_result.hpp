#pragma once
#include "stream_census.hpp"
#include <algorithm>
#include <cerrno>
#include <filesystem>
#include <sys/stat.h>

namespace symphony::sbv::detail::trial {
inline std::string protocol(std::string operation) {
  std::replace(operation.begin(), operation.end(), '_', '-');
  return "symphony.sbv." + operation + ".v1";
}
inline bool partitioned(const Json &request) {
  return request.contains("output");
}
inline Json root_reference(const Json &receipt, const Json &options) {
  if (receipt.contains("storage"))
    return {{"kind", "bundle"},
            {"reference", receipt.at("storage").at("reference")},
            {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
            {"read_options", options}};
  return {{"path", receipt.at("path")},
          {"expected_sha256", receipt.at("content_sha256")},
          {"pointer", ""}};
}
inline void read_options(const Json &options) {
  keys(options, {"max_page_bytes", "cache_bytes"});
  (void)u64(options.at("cache_bytes"));
  need(options.at("max_page_bytes").is_null() ||
           u64(options.at("max_page_bytes")) > 0,
       "positive selected trial page limit or null required");
}
inline logical::Value verify(const Json &receipt, const Json &request,
                             const std::string &operation, const Json &options,
                             std::int64_t end) {
  need(receipt.at("protocol") == protocol(operation),
       "child receipt operation mismatch");
  need(receipt.at("status") != "recovery_required",
       "incomplete child is not a result");
  if (partitioned(request)) {
    const auto &storage = receipt.at("storage"), &output = request.at("output");
    need(storage.at("kind") == "partitioned" &&
             storage.at("reference").at("manifest_path") ==
                 str(output.at("bundle_path")) + "/manifest.json" &&
             storage.at("reference").at("content_sha256") ==
                 receipt.at("content_sha256") &&
             storage.at("workspace_path") == output.at("workspace_path") &&
             storage.at("write_options") == output.at("write_options"),
         "child partitioned output correspondence mismatch");
    read_options(options);
  } else {
    need(!receipt.contains("storage") &&
             receipt.at("path") == request.at("output_path"),
         "child legacy output correspondence mismatch");
    const auto bytes = read_file(str(receipt.at("path")), end);
    need(e::sha256_hex(bytes) == str(receipt.at("file_sha256")) &&
             dec(bytes.size()) == str(receipt.at("bytes")),
         "child result file changed");
  }
  auto result =
      select_logical_result(root_reference(receipt, options), end).value;
  need(result.at("status").materialize() == receipt.at("status") &&
           result.at("sections").at("summary").materialize() ==
               receipt.at("summary"),
       "child receipt status/summary mismatch");
  return result;
}
inline Json binding(const Json &receipt, const Json &options,
                    const Json &pointer) {
  auto out = root_reference(receipt, options);
  if (out.contains("selector"))
    out["selector"]["pointer"] = pointer;
  else
    out["pointer"] = pointer;
  return out;
}
inline bool selects(const logical::Value &root, const std::string &pointer) {
  (void)Json::json_pointer(pointer);
  auto value = root;
  std::size_t at = 0;
  while (at < pointer.size()) {
    const auto next = pointer.find('/', at + 1);
    auto edge = pointer.substr(
        at + 1, next == std::string::npos ? next : next - at - 1);
    std::string decoded;
    for (std::size_t i = 0; i < edge.size(); ++i) {
      if (edge[i] == '~') {
        decoded += edge[++i] == '0' ? '~' : '/';
      } else
        decoded += edge[i];
    }
    if (value.kind() == "object") {
      if (!value.contains(decoded))
        return false;
      value = value.at(decoded);
    } else if (value.kind() == "array") {
      std::uint64_t index{};
      const auto [last, error] = std::from_chars(
          decoded.data(), decoded.data() + decoded.size(), index);
      if (decoded.empty() || (decoded.size() > 1 && decoded[0] == '0') ||
          error != std::errc{} || last != decoded.data() + decoded.size() ||
          index >= value.size())
        return false;
      value = value.at(index);
    } else
      return false;
    if (next == std::string::npos)
      break;
    at = next;
  }
  return true;
}
inline void recovery(const Json &receipt, const Json &request,
                     const std::string &op) {
  need(receipt.at("protocol") == protocol(op) &&
           receipt.at("status") == "recovery_required" &&
           receipt.at("request_sha256") == e::sha256_hex(request.dump()) &&
           receipt.at("automatic_retry") == false,
       "child recovery is not bound to this exact request");
}
inline bool appeared(const Json &request) {
  auto exists = [](const std::string &path) {
    struct stat st{};
    if (::lstat(path.c_str(), &st) == 0)
      return true;
    need(errno == ENOENT, "child destination state unavailable");
    return false;
  };
  if (partitioned(request))
    return exists(str(request.at("output").at("bundle_path"))) ||
           exists(str(request.at("output").at("workspace_path")));
  return exists(str(request.at("output_path")));
}
} // namespace symphony::sbv::detail::trial
