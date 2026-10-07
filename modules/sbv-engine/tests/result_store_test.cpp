#include "../src/result_store.hpp"
#include "../src/streaming_sha256.hpp"
#include <algorithm>
#include <array>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <limits>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <unistd.h>

namespace s = symphony::sbv::result_store;
namespace e = symphony::knowledge::engine;
using Json = s::Json;
std::uint64_t checks = 0;
void test(bool v, const char *message) {
  ++checks;
  if (!v)
    throw std::runtime_error(message);
}
template <class F> void rejects(F f, const char *message) {
  bool rejected = false;
  try {
    f();
  } catch (const std::exception &) {
    rejected = true;
  }
  test(rejected, message);
}
std::string read(const std::string &path) {
  std::ifstream in(path, std::ios::binary);
  return std::string(std::istreambuf_iterator<char>(in), {});
}
void write(const std::string &path, const std::string &bytes) {
  std::ofstream out(path, std::ios::binary);
  out << bytes;
  if (!out)
    throw std::runtime_error("test fixture write failed");
}
Json body() {
  Json sections = Json::object();
  for (const auto *name : {"summary", "signals", "execution", "distributions",
                           "studies", "replay", "comparisons", "search",
                           "resources", "diagnostics", "choices", "provenance"})
    sections[name] = {{"data", nullptr},
                      {"reason", "not selected by fixture"},
                      {"status", "not_selected"}};
  sections["unknown"] = {
      {"data",
       {{"", Json::array({"-9223372036854775808", "18446744073709551615", true,
                          nullptr})},
        {"a/~", {{"深", "line\n\t\x1b"}, {"unicode", "é😀\u2028"}}}}},
      {"reason", ""},
      {"status", "available"}};
  return {{"origin", "independent-test"},
          {"protocol", "symphony.sbv.result.v1"},
          {"sections", sections},
          {"status", "completed"}};
}
void object(s::ResultWriter &w, const Json &v) {
  w.begin_object();
  for (auto i = v.begin(); i != v.end(); ++i) {
    w.key(i.key());
    w.value(i.value());
  }
  w.end();
}
std::string streaming_hash(const std::string &input, std::size_t chunk) {
  symphony::sbv::hash_detail::Sha256 h;
  for (std::size_t at = 0; at < input.size();
       at += std::min(chunk, input.size() - at)) {
    const auto n = std::min(chunk, input.size() - at);
    test(h.update(reinterpret_cast<const std::uint8_t *>(input.data() + at), n),
         "SHA update refused");
  }
  std::uint8_t out[32];
  h.finish(out);
  constexpr char hex[] = "0123456789abcdef";
  std::string result;
  for (auto b : out) {
    result += hex[b >> 4];
    result += hex[b & 15];
  }
  return result;
}
void streamed(s::ResultWriter &w, std::uint64_t rows, std::size_t padding) {
  w.begin_object();
  w.key("origin");
  w.value("stream-fixture");
  w.key("protocol");
  w.value("symphony.sbv.result.v1");
  w.key("sections");
  w.begin_object();
  const auto base = body().at("sections");
  for (auto it = base.begin(); it != base.end(); ++it) {
    w.key(it.key());
    if (it.key() != "signals") {
      w.value(it.value());
      continue;
    }
    w.begin_object();
    w.key("data");
    w.begin_array();
    for (std::uint64_t i = 0; i < rows; ++i)
      w.value(Json{{"id", std::to_string(i)},
                   {"payload", std::string(padding, 'x')},
                   {"signed", "-9223372036854775808"},
                   {"time", "18446744073709551615"}});
    w.end();
    w.key("reason");
    w.value("");
    w.key("status");
    w.value("available");
    w.end();
  }
  w.end();
  w.key("status");
  w.value("completed");
  w.end();
}
int main(int argc, char **argv) {
  try {
    if (argc > 1 && std::string(argv[1]) == "large") {
      test(argc == 3, "large requires fresh absolute bundle path");
      s::ResultWriter writer(argv[2]);
      streamed(writer, 500000, 256);
      auto receipt = writer.finish();
      test(receipt.logical_body_bytes > (128U << 20),
           "large canonical bytes threshold");
      test(receipt.logical_body_nodes > 4000000,
           "large logical nodes threshold");
      s::ResultReader reader(receipt.reference);
      auto query = reader.query("/sections/signals/data", 499990, 10, 512000);
      test(query.at("nodes").size() == 10 && query.at("complete") == true,
           "large late query");
      std::cout << Json{{"checks", std::to_string(checks)},
                        {"receipt", receipt.json()},
                        {"late_query", query}}
                       .dump()
                << '\n';
      return 0;
    }
    const auto base = std::string("/private/tmp/sbv-result-kernel-") +
                      std::to_string(::getpid());
    std::filesystem::create_directory(base);
    for (const auto &text : {std::string{}, std::string("abc"),
                             std::string(1000, 'a'), std::string("hé😀")})
      for (const auto chunk :
           {std::size_t(1), std::size_t(7), std::size_t(63), std::size_t(64),
            std::size_t(65), std::size_t(257)})
        test(streaming_hash(text, chunk) == e::sha256_hex(text),
             "extracted SHA parity");
    test(streaming_hash("abc", 2) ==
             "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
         "known SHA vector");
    symphony::sbv::hash_detail::Sha256 overflow;
    std::uint8_t byte = 0;
    test(!overflow.update(&byte, std::numeric_limits<std::size_t>::max()),
         "SHA length bound");
    auto source = body();
    const auto logical = e::sha256_hex(source.dump());
    s::ResultWriter writer(base + "/small");
    object(writer, source);
    auto receipt = writer.finish();
    test(receipt.reference.content_sha256 == logical,
         "legacy canonical body hash parity");
    source["content_sha256"] = logical;
    s::ResultReader reader(receipt.reference);
    std::string exported;
    auto verified =
        reader.export_json([&](std::string_view x) { exported += x; });
    test(exported == source.dump(), "legacy complete exported bytes parity");
    test(verified.at("verification_extent") == "full_logical_closure",
         "closure extent");
    auto root = reader.query_node(0, 0, 20, 512000);
    test(root.at("nodes").size() == 5, "virtual root digest included");
    test(root.at("nodes")[0].at("node_id") == "1", "virtual digest ID");
    auto nested = reader.query("/sections/unknown/data/a~1~0", 0, 8, 512000);
    test(nested.at("nodes").size() == 2, "pointer escape traversal");
    for (const auto &row : nested.at("nodes")) {
      auto same = reader.query_node(
          std::stoull(row.at("node_id").get<std::string>()), 0, 1, 512000);
      test(same.at("nodes")[0] == row, "preorder locator parity");
    }
    auto file = s::export_file(reader, base + "/small-export.json", "json");
    test(read(base + "/small-export.json") == source.dump() + "\n",
         "native file export exactbytes");
    test(file.at("completion_suffix") == "}\n", "JSON completion suffix");
    auto imported = s::import_json(base + "/small-export.json",
                                   file.at("file_sha256"), base + "/imported");
    test(imported.at("reference").at("content_sha256") == logical,
         "stream import parity");
    auto nd = s::export_file(reader, base + "/small-export.ndjson", "ndjson");
    test(read(base + "/small-export.ndjson")
             .ends_with(nd.at("completion_suffix").get<std::string>()),
         "NDJSON completion suffix");
    auto compact_source = body();
    compact_source["sections"]["studies"] = {
        {"data", {{std::string(100, 'k'), {{"v", std::string(1000, 'x')}}}}},
        {"reason", ""},
        {"status", "available"}};
    compact_source["content_sha256"] = e::sha256_hex(compact_source.dump());
    const auto compact_bytes = compact_source.dump();
    write(base + "/compact-input.json", compact_bytes);
    auto compact_receipt = s::import_json(
        base + "/compact-input.json", e::sha256_hex(compact_bytes),
        base + "/compact-fallback", {1600, 2});
    test(compact_receipt.at("reference").at("content_sha256") ==
             compact_source.at("content_sha256"),
         "large parent key uses indexed compaction fallback");
    auto buffered_body = body();
    buffered_body["sections"]["signals"] = {
        {"data", Json::array()}, {"reason", ""}, {"status", "available"}};
    for (int i = 0; i < 40; ++i)
      buffered_body["sections"]["signals"]["data"].push_back(
          std::string(4096, static_cast<char>('a' + i % 26)));
    s::ResultWriter buffered_writer(base + "/buffered");
    object(buffered_writer, buffered_body);
    const auto buffered_receipt = buffered_writer.finish();
    buffered_body["content_sha256"] = buffered_receipt.reference.content_sha256;
    s::ResultReader buffered_reader(buffered_receipt.reference);
    auto buffered_export =
        s::export_file(buffered_reader, base + "/buffered.json", "json");
    const auto buffered_bytes = read(base + "/buffered.json");
    test(buffered_bytes == buffered_body.dump() + "\n" &&
             buffered_bytes.size() > 65536,
         "buffered multi-write JSON parity");
    test(buffered_export.at("file_sha256") == e::sha256_hex(buffered_bytes),
         "buffered exported exact-byte digest");
    rejects([&] { s::ResultWriter duplicate(base + "/small"); },
            "existing bundle refused");
    rejects([&] { s::ResultWriter long_path("/" + std::string(4085, 'a')); },
            "derived manifest path preflight");
    rejects(
        [&] { s::export_file(reader, base + "/small-export.json", "json"); },
        "existing export refused");
    rejects([&] { reader.query_node(0, 0, 1, 20); },
            "too-small page refuses progress");
    rejects(
        [&] {
          s::ResultReader wrong(
              {receipt.reference.manifest_path, std::string(64, '0'), logical});
        },
        "manifest digest mismatch rejected");
    for (const auto n : {3, 5}) {
      auto small = body();
      small["sections"].erase("unknown");
      const auto path = base + "/fringe" + std::to_string(n);
      s::ResultWriter w(path, {1400, 2});
      // Stream every container; retain one large scalar per array leaf.
      w.begin_object();
      for (auto it = small.begin(); it != small.end(); ++it) {
        w.key(it.key());
        if (it.key() != "sections") {
          w.value(it.value());
          continue;
        }
        w.begin_object();
        for (auto section = it.value().begin(); section != it.value().end();
             ++section) {
          w.key(section.key());
          if (section.key() != "signals") {
            w.value(section.value());
            continue;
          }
          w.begin_object();
          w.key("data");
          w.begin_array();
          for (int k = 0; k < n; ++k)
            w.value(std::string(700, static_cast<char>('a' + k)));
          w.end();
          w.key("reason");
          w.value("");
          w.key("status");
          w.value("available");
          w.end();
        }
        w.end();
      }
      w.end();
      auto r = w.finish();
      s::ResultReader read_fringe(r.reference);
      auto q = read_fringe.query("/sections/signals/data", 0, 10, 100000);
      test(q.at("nodes").size() == static_cast<std::size_t>(n),
           "right-fringe index traversal");
      test(read_fringe.verify_closure().at("verification_extent") ==
               "full_logical_closure",
           "right-fringe closure");
    }
    rejects(
        [&] {
          s::ResultWriter bad(base + "/numeric");
          auto b = body();
          b["sections"]["signals"]["data"] = 1;
          object(bad, b);
          bad.finish();
        },
        "numeric literal refused");
    rejects(
        [&] {
          s::ResultWriter bad(base + "/semantics");
          auto b = body();
          b["sections"].erase("signals");
          object(bad, b);
          bad.finish();
        },
        "required section refused");
    test(!std::filesystem::exists(base + "/semantics/manifest.json"),
         "semantic failure publishes no root");
    bool cancelled = false;
    s::ResultWriter cancel(base + "/cancel", {}, [&] {
      if (cancelled)
        throw std::runtime_error("caller cancellation");
    });
    cancelled = true;
    rejects([&] { object(cancel, body()); }, "caller cancellation propagated");
    test(!std::filesystem::exists(base + "/cancel/manifest.json"),
         "cancel publishes no root");
    std::cout << Json{{"status", "passed"},
                      {"checks", std::to_string(checks)},
                      {"fixture_directory", base}}
                     .dump()
              << '\n';
    return 0;
  } catch (const s::StoreError &x) {
    std::cerr << Json{{"code", x.code},
                      {"message", x.what()},
                      {"recovery", x.recovery}}
                     .dump()
              << '\n';
    return 2;
  } catch (const std::exception &x) {
    std::cerr << x.what() << '\n';
    return 3;
  }
}
