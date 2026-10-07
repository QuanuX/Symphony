#ifdef __APPLE__
#define SBV_TEST_TMP "/private/tmp"
#else
#define SBV_TEST_TMP "/tmp"
#endif
#include "result_store.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <string>
#include <unistd.h>

namespace s = symphony::sbv::result_store;
namespace k = symphony::knowledge::engine;
using Json = s::Json;
namespace fs = std::filesystem;
std::uint64_t checks = 0;
void require(bool yes, const char *message) {
  ++checks; if (!yes) throw std::runtime_error(message);
}
template<class F> void rejects(F &&f, const char *message) {
  bool rejected = false;
  try { f(); } catch (const std::exception &) { rejected = true; }
  require(rejected, message);
}
std::uint64_t integer(const Json &j) { return std::stoull(j.get<std::string>()); }
std::uint64_t nodes(const Json &v) {
  std::uint64_t n = 1;
  if (v.is_structured()) for (auto i = v.begin(); i != v.end(); ++i)
    n += nodes(i.value()) + (v.is_object() ? 1 : 0);
  return n;
}
void emit(s::ResultWriter &w, const Json &v) {
  if (!v.is_structured()) { w.value(v); return; }
  if (v.is_object()) w.begin_object(); else w.begin_array();
  for (auto i = v.begin(); i != v.end(); ++i) {
    if (v.is_object()) w.key(i.key());
    emit(w, i.value());
  }
  w.end();
}
Json body(std::size_t size) {
  Json sections = Json::object();
  for (const auto *name : {"summary", "signals", "execution", "distributions",
       "studies", "replay", "comparisons", "search", "resources", "diagnostics",
       "choices", "provenance"})
    sections[name] = {{"data", nullptr}, {"reason", "not selected"}, {"status", "not_selected"}};
  Json rows = Json::array();
  for (std::size_t i = 0; i < size; ++i)
    rows.push_back({{"id", "signal-" + std::to_string(i)}, {"ordinal", std::to_string(i)},
      {"padding", std::string(211, 'p')}, {"unknown", {{"/é~", Json::array({true, nullptr, "深"})}}}});
  sections["signals"] = {{"data", rows}, {"reason", ""}, {"status", "available"}};
  return {{"origin", "independent-reader-fixture"}, {"protocol", "symphony.sbv.result.v1"},
          {"sections", sections}, {"status", "completed"}};
}
s::Receipt write(const std::string &path, const Json &v) {
  s::ResultWriter w(path, {2048, 2}); emit(w, v); return w.finish();
}
s::Receipt descendant(const std::string &path, const s::NodeHandle &value) {
  s::ResultWriter w(path, {2048, 2}); auto skeleton = body(0);
  w.begin_object();
  for (auto field = skeleton.begin(); field != skeleton.end(); ++field) {
    w.key(field.key());
    if (field.key() != "sections") { emit(w, field.value()); continue; }
    w.begin_object();
    for (auto sec = field.value().begin(); sec != field.value().end(); ++sec) {
      w.key(sec.key());
      if (sec.key() != "replay") { emit(w, sec.value()); continue; }
      w.begin_object(); w.key("data"); (void)w.append_subtree(value);
      w.key("reason"); w.value(""); w.key("status"); w.value("available"); w.end();
    }
    w.end();
  }
  w.end(); return w.finish();
}
fs::path last_signal_leaf(const fs::path &directory, std::string last_id) {
  for (const auto &entry : fs::directory_iterator(directory)) {
    if (entry.path().filename() == "manifest.json") continue;
    std::ifstream input(entry.path()); Json page; input >> page;
    if (page.value("level", "") != "0") continue;
    for (const auto &row : page.at("entries"))
      if (row.at("node").value("kind", "") == "inline" &&
          row.at("node").at("value").is_object() &&
          row.at("node").at("value").value("id", "") == last_id)
        return entry.path();
  }
  throw std::runtime_error("expected final signal leaf missing");
}
int main(int argc, char **argv) {
  try {
    std::string base;
    if (argc == 2) { base = argv[1]; require(fs::create_directory(base), "fresh evidence root"); }
    else if (argc == 1) {
      char pattern[] = SBV_TEST_TMP "/sbv-reader-XXXXXX";
      const auto created = ::mkdtemp(pattern);
      require(created != nullptr, "fresh mkdtemp root"); base = created;
    } else throw std::runtime_error("usage: reader-test [new-evidence-directory]");
    const auto specimen = body(129);
    const auto receipt = write(base + "/source", specimen);
    auto complete = specimen; complete["content_sha256"] = receipt.reference.content_sha256;
    require(receipt.reference.content_sha256 == k::sha256_hex(specimen.dump()), "independent body hash");
    auto source = [&] {
      s::ResultReader r(receipt.reference, {std::nullopt, 0});
      require(r.verify_closure().at("verification_extent") == "full_logical_closure", "outer admission");
      return r.select("");
    }();
    require(source.read_value() == complete, "root handle lifetime and unknown fields");
    const auto root_hash = source.hash();
    require(root_hash.at("canonical_sha256") == k::sha256_hex(complete.dump()), "complete root byte hash");
    require(integer(root_hash.at("nodes")) == nodes(complete), "root counted nodes independent oracle");
    require(integer(root_hash.at("canonical_bytes")) == complete.dump().size(), "root exact byte counts");
    require(root_hash.at("verification_extent") == "full_logical_closure", "root closure scope");
    auto digest = source.child("content_sha256");
    require(digest.describe().at("node_id") == "1" && digest.read_value() == receipt.reference.content_sha256,
            "virtual digest identity");
    auto root_cursor = source.children(); std::size_t root_index = 0;
    for (auto i = complete.begin(); i != complete.end(); ++i) {
      auto value = root_cursor.next(); require(value.has_value(), "root cursor row");
      require(value->describe().at("edge") == i.key() && value->read_value() == i.value(), "root cursor order/value");
      ++root_index;
    }
    require(root_index == 5 && !root_cursor.next() && root_cursor.progress().at("complete") == true, "root cursor completion");
    auto array = source.child("sections").child("signals").child("data");
    const auto expected = specimen.at("sections").at("signals").at("data");
    require(array.hash().at("canonical_sha256") == k::sha256_hex(expected.dump()), "array hash independent oracle");
    require(array.hash().at("verification_extent") == "selected_subtree", "subtree scope explicit");
    for (const std::uint64_t first : {0ULL, 1ULL, 64ULL, 126ULL}) {
      auto cursor = array.children(first, 3);
      for (std::uint64_t offset = 0; offset < 3; ++offset) {
        auto value = cursor.next(); require(value.has_value(), "selected row exists");
        require(value->read_value() == expected.at(first + offset), "ordered range exact rows");
        s::ResultReader r(receipt.reference, {std::nullopt, 0});
        const auto id = integer(value->describe().at("node_id"));
        require(r.select_node(id).read_value() == value->read_value(), "node ID parity");
        require(r.query_node(id, 0, 1, 10000).at("selected_node_id") == value->describe().at("node_id"), "R3.1 locator parity");
      }
      require(!cursor.next() && cursor.progress().at("complete") == true, "exact bounded range end");
    }
    auto empty_range = array.children(129, 0);
    require(!empty_range.next(), "empty tail range");
    rejects([&] { (void)array.children(128, 2); }, "range not silently clipped");
    rejects([&] { (void)digest.children(); }, "scalar cursor refused");
    auto row = array.child(std::uint64_t{4});
    const auto bytes = expected.at(4).dump().size();
    const auto count = nodes(expected.at(4));
    require(row.read_value({bytes, count}) == expected.at(4), "exact optional materialization limits");
    std::uint64_t callbacks = 0; s::ValueVisitor observer;
    observer.begin_object = [&] { ++callbacks; };
    rejects([&] { (void)row.visit(observer, {bytes - 1, count}); }, "byte budget refuses");
    rejects([&] { (void)row.visit(observer, {bytes, count - 1}); }, "node budget refuses");
    require(callbacks == 0, "declared budget refusal precedes callbacks");
    std::string exported;
    const auto export_receipt = row.export_json([&](std::string_view p) { exported += p; });
    require(export_receipt.at("canonical_sha256") ==
            k::sha256_hex(exported), "subtree export receipt hash");
    require(exported == expected.at(4).dump(), "canonical subtree export parity");
    std::uint64_t writes = 0;
    rejects([&] { row.export_json([&](std::string_view) { if (++writes == 2) throw std::runtime_error("sink refused"); }); }, "sink failure propagated");
    require(writes == 2, "no callback after failed sink");
    bool cancelled = false;
    s::ResultReader interrupted(receipt.reference, {}, [&] { if (cancelled) throw std::runtime_error("cancelled"); });
    auto cancel_cursor = interrupted.select("/sections/signals/data").children();
    require(cancel_cursor.next().has_value(), "cursor first before cancellation");
    cancelled = true;
    rejects([&] { cancel_cursor.next(); }, "cursor cancellation");
    cancelled = false;
    rejects([&] { cancel_cursor.next(); }, "failed cursor cannot resume implicitly");
    require(cancel_cursor.progress().at("failed") == true, "failed cursor evidence");
    auto cancelled_row = interrupted.select("/sections/signals/data/0");
    exported.clear();
    rejects([&] { cancelled_row.export_json([&](std::string_view p) { exported += p; cancelled = true; }); }, "cancel during export");
    require(exported != expected.at(0).dump() && (exported.empty() || exported.back() != '}'), "withheld terminal on cancellation");
    cancelled = false;
    const auto copied = descendant(base + "/descendant", source);
    fs::remove_all(base + "/source");
    s::ResultReader final(copied.reference, {std::nullopt, 0});
    require(final.verify_closure().at("verification_extent") == "full_logical_closure", "descendant self-contained closure");
    require(final.select("/sections/replay/data").read_value() == complete, "embedded full result remains exact without ancestor");

    const auto damaged = write(base + "/damaged", specimen);
    const auto missing = last_signal_leaf(base + "/damaged", "signal-128");
    const auto saved = missing.string() + ".held"; fs::rename(missing, saved);
    s::ResultReader bounded(damaged.reference, {std::nullopt, 0});
    auto selected = bounded.select("/sections/signals/data");
    auto one = selected.children(0, 1);
    require(one.next()->read_value() == expected.at(0) && !one.next(), "range does not open missing later dependency");
    rejects([&] { selected.child(std::uint64_t{128}).read_value(); }, "missing selected dependency rejected");
    s::ResultWriter bad_copy(base + "/failed-copy", {2048, 2});
    bad_copy.begin_object(); bad_copy.key("a");
    rejects([&] { bad_copy.append_subtree(selected); }, "copy missing dependency rejected");
    rejects([&] { bad_copy.end(); }, "copy failure poisons writer");
    require(!fs::exists(base + "/failed-copy/manifest.json"), "failed copy unpublished");
    fs::rename(saved, missing);
    { std::ofstream corrupt(missing, std::ios::binary | std::ios::app); corrupt << ' '; }
    s::ResultReader corrupt(damaged.reference, {std::nullopt, 0});
    rejects([&] { corrupt.select("/sections/signals/data").hash(); }, "corrupt dependency rejected");
    const auto undercounted = write(base + "/undercounted", specimen);
    Json manifest;
    { std::ifstream input(undercounted.reference.manifest_path); input >> manifest; }
    manifest["root"]["bytes"] = "1000";
    const auto replacement = manifest.dump();
    { std::ofstream output(undercounted.reference.manifest_path, std::ios::binary | std::ios::trunc); output << replacement; }
    auto forged = undercounted.reference;
    forged.manifest_sha256 = k::sha256_hex(replacement);
    s::ResultReader understated(forged, {std::nullopt, 0});
    rejects([&] { understated.select("").read_value({1200, std::nullopt}); }, "actual byte budget checked despite forged metrics");
    std::string provisional;
    rejects([&] { understated.select("").export_json([&](std::string_view p) { provisional += p; }); }, "false root metrics rejected");
    require(provisional != complete.dump() && provisional.size() + 1 == complete.dump().size(),
            "final closure withheld for metric disagreement");
    std::cout << Json{{"protocol", "research.sbv.reader-extension-test.v1"}, {"status", "passed"},
      {"assertions", std::to_string(checks)}, {"evidence_root", base}, {"source_receipt", receipt.json()},
      {"descendant_receipt", copied.json()}, {"root_hash", root_hash}}.dump() << '\n';
    return 0;
  } catch (const std::exception &e) { std::cerr << "after " << checks << " checks: " << e.what() << '\n'; return 1; }
}
