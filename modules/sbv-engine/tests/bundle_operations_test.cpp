#include "../src/detail.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/sbv/sdk.hpp>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool b,
           std::source_location location = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("bundle assertion " +
                             std::to_string(location.line()));
}
template <class F> void rejects(F &&f) {
  bool refused = false;
  try {
    f();
  } catch (const std::exception &) {
    refused = true;
  }
  check(refused);
}
J request(const std::string &op, const J &reference) {
  auto slug = op;
  std::replace(slug.begin(), slug.end(), '_', '-');
  return {{"protocol", "symphony.sbv." + slug + "-input.v1"},
          {"reference", reference},
          {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}},
          {"extensions", {{"caller_metadata", "preserved"}}}};
}
J invoke(const std::string &op, const J &p) {
  return s::dispatch(op, p, e::no_deadline);
}
J sdk(const std::string &op, const J &p) {
  J input{{"protocol", e::process_protocol_v2},
          {"request_id", std::string(128, 'a')},
          {"correlation_id", std::string(128, 'b')},
          {"operation", op},
          {"target_engine", "symphony-sbv"},
          {"deadline_unix_ms", nullptr},
          {"payload", p}};
  const auto response = s::sdk::process(input.dump());
  if (response.status != 0)
    throw std::runtime_error(response.json);
  auto parsed = J::parse(response.json);
  const auto sha = parsed.at("response_digest");
  parsed.erase("response_digest");
  check(sha == e::tagged_sha256(parsed.dump()));
  return parsed.at("result");
}
int main() try {
#if defined(__APPLE__)
  char temp[] = "/private/tmp/sbv-bundle-ops-XXXXXX";
#else
  char temp[] = "/tmp/sbv-bundle-ops-XXXXXX";
#endif
  auto *created = ::mkdtemp(temp);
  check(created != nullptr);
  const std::filesystem::path dir(created);
  struct Cleanup {
    std::filesystem::path path;
    ~Cleanup() {
      std::error_code ec;
      std::filesystem::remove_all(path, ec);
    }
  } cleanup{dir};
  auto body = d::base("bundle_operation_test");
  J rows = J::array();
  for (unsigned i = 0; i < 6000; ++i)
    rows.push_back(std::to_string(i));
  body["sections"]["signals"] = d::section(rows);
  body["sections"]["studies"] =
      d::section(J{{"line\nkey", "<>&é\x1b"}, {"empty", J::array()}});
  const auto logical = s::seal_result(body);
  const auto input_path = (dir / "input.json").string();
  const auto bytes = logical.dump();
  d::create_file(input_path, bytes, e::no_deadline);
  J p{{"protocol", "symphony.sbv.bundle-import-input.v1"},
      {"input_path", input_path},
      {"expected_file_sha256", e::sha256_hex(bytes)},
      {"bundle_path", (dir / "bundle").string()},
      {"write_options", {{"page_bytes", "4096"}, {"index_fanout", "3"}}},
      {"extensions", J::object()}};
  auto bad = p;
  bad["unknown"] = true;
  rejects([&] { invoke("bundle_import", bad); });
  check(!std::filesystem::exists(dir / "bundle"));
  bad = p;
  bad["extensions"]["large"] = J::array();
  for (unsigned i = 0; i < 32500; ++i)
    bad["extensions"]["large"].push_back("x");
  rejects([&] { invoke("bundle_import", bad); });
  check(!std::filesystem::exists(dir / "bundle"));
  const auto imported = sdk("bundle_import", p);
  check(imported.at("status") == "complete" &&
        imported.at("verification_extent") == "full_logical_closure");
  const auto ref = imported.at("reference");
  check(ref.at("content_sha256") == logical.at("content_sha256"));
  auto inspect = request("bundle_inspect", ref);
  auto metadata = sdk("bundle_inspect", inspect);
  check(metadata.at("verification_extent") == "manifest_only" &&
        metadata.at("root_children") == "5");
  check(metadata.at("extensions") == inspect.at("extensions"));
  auto verify = request("bundle_verify", ref);
  auto verified = sdk("bundle_verify", verify);
  check(verified.at("verification_extent") == "full_logical_closure");
  for (auto key : {"logical_body_bytes", "logical_body_nodes",
                   "logical_body_values", "exported_value_nodes"})
    check(verified.at(key) == imported.at(key));
  auto q = request("bundle_query", ref);
  q.update(J{{"selector", {{"kind", "node_id"}, {"node_id", "0"}}},
             {"cursor", nullptr},
             {"row_limit", "1"},
             {"byte_limit", "8192"}});
  auto first = sdk("bundle_query", q);
  check(first.at("nodes").size() == 1 &&
        first.at("nodes").at(0).at("node_id") == "1" &&
        !first.at("complete").get<bool>());
  q["cursor"] = first.at("next_cursor");
  q["row_limit"] = "3";
  q["byte_limit"] = "16384";
  auto second = sdk("bundle_query", q);
  check(second.at("nodes").size() == 3 && second.at("offset") == "1");
  bad = q;
  bad["selector"]["node_id"] = "1";
  rejects([&] { invoke("bundle_query", bad); });
  bad = q;
  bad["cursor"]["reference"]["manifest_sha256"] = std::string(64, '0');
  rejects([&] { invoke("bundle_query", bad); });
  q["cursor"] = second.at("next_cursor");
  auto last = sdk("bundle_query", q);
  check(last.at("nodes").size() == 1 && last.at("complete").get<bool>() &&
        last.at("next_cursor").is_null());
  q["cursor"] = nullptr;
  q["selector"] = {{"kind", "pointer"}, {"pointer", "/sections/signals/data"}};
  q["row_limit"] = "18446744073709551615";
  q["byte_limit"] = "18446744073709551615";
  std::uint64_t count = 0;
  do {
    auto page = sdk("bundle_query", q);
    check(page.at("offset") == std::to_string(count));
    for (const auto &node : page.at("nodes"))
      check(node.at("value") == std::to_string(count++));
    q["cursor"] = page.at("next_cursor");
  } while (!q.at("cursor").is_null());
  check(count == 6000);
  q["selector"] = {{"kind", "pointer"},
                   {"pointer", "/sections/studies/data/line\nkey"}};
  q["row_limit"] = "1";
  q["byte_limit"] = "16384";
  auto scalar = sdk("bundle_query", q);
  check(scalar.at("nodes").at(0).at("value") == "<>&é\x1b");
  q["selector"] = {{"kind", "pointer"},
                   {"pointer", "/sections/studies/data/empty"}};
  auto empty = sdk("bundle_query", q);
  check(empty.at("nodes").empty() && empty.at("complete") == true);
  q["byte_limit"] = "1";
  rejects([&] { invoke("bundle_query", q); });
  auto export_request = request("bundle_export", ref);
  export_request["output_path"] = (dir / "output.json").string();
  export_request["format"] = "json";
  auto exported = sdk("bundle_export", export_request);
  check(exported.at("completion_suffix") == "}\n");
  const auto exported_bytes =
      d::read_file(export_request.at("output_path"), e::no_deadline);
  check(exported_bytes == bytes + "\n" &&
        exported.at("file_sha256") == e::sha256_hex(exported_bytes));
  const auto refused = sdk("bundle_export", export_request);
  check(refused.at("status") == "recovery_required" &&
        refused.at("code") == "sbv.bundle_export_incomplete");
  check(refused.at("request_sha256") == e::sha256_hex(export_request.dump()));
  check(refused.at("recovery").at("published") == false);
  check(d::read_file(export_request.at("output_path"), e::no_deadline) ==
        exported_bytes);
  export_request["output_path"] = (dir / "output.ndjson").string();
  export_request["format"] = "ndjson";
  exported = sdk("bundle_export", export_request);
  auto ndjson = d::read_file(export_request.at("output_path"), e::no_deadline);
  check(ndjson.ends_with(exported.at("completion_suffix").get<std::string>()));
  std::uint64_t nodes = 0;
  bool started = false, ended = false;
  for (std::size_t pos = 0; pos < ndjson.size();) {
    const auto end = ndjson.find('\n', pos);
    check(end != std::string::npos);
    const auto record = J::parse(ndjson.substr(pos, end - pos));
    pos = end + 1;
    if (record.at("event") == "begin") {
      check(!started);
      started = true;
    } else if (record.at("event") == "node") {
      check(started && !ended);
      check(record.at("node_id") == std::to_string(nodes++));
    } else {
      check(record.at("status") == "complete" &&
            record.at("nodes") == std::to_string(nodes));
      ended = true;
    }
  }
  check(started && ended &&
        exported.at("exported_value_nodes") == std::to_string(nodes));
  std::cout << checks << " bundle operation/SDK checks passed\n";
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
