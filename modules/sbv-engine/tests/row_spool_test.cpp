#ifdef __APPLE__
#define SBV_TEST_TMP "/private/tmp"
#else
#define SBV_TEST_TMP "/tmp"
#endif
#include "row_spool.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <filesystem>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace s = symphony::sbv::result_store;
namespace k = symphony::knowledge::engine;
using Json = s::Json;
std::uint64_t checks = 0;
void require(bool yes, const char *message) {
  ++checks;
  if (!yes) throw std::runtime_error(message);
}
template<class Function> void rejects(Function &&f, const char *message) {
  bool rejected = false;
  try { f(); } catch (const std::exception &) { rejected = true; }
  require(rejected, message);
}
Json row(std::uint64_t index) {
  return {{"id", "signal-" + std::to_string(index)},
          {"index", std::to_string(index)},
          {"nested", {{"unanticipated", Json::array({true, nullptr, "é/深~"})}}}};
}
Json body() {
  Json sections = Json::object();
  for (const auto *name : {"summary", "signals", "execution", "distributions",
                           "studies", "replay", "comparisons", "search",
                           "resources", "diagnostics", "choices", "provenance"})
    sections[name] = {{"data", nullptr}, {"reason", "not selected"},
                      {"status", "not_selected"}};
  return {{"origin", "independent-spool-consumer"},
          {"protocol", "symphony.sbv.result.v1"}, {"sections", sections},
          {"status", "completed"}};
}
s::Receipt descendant(const std::string &path, const s::NodeHandle &array,
                      s::WriteOptions options = {}) {
  s::ResultWriter out(path, options);
  const auto specimen = body();
  out.begin_object();
  for (auto field = specimen.begin(); field != specimen.end(); ++field) {
    out.key(field.key());
    if (field.key() != "sections") { out.value(field.value()); continue; }
    out.begin_object();
    for (auto section = field.value().begin(); section != field.value().end(); ++section) {
      out.key(section.key());
      if (section.key() != "signals" && section.key() != "replay") {
        out.value(section.value()); continue;
      }
      out.begin_object(); out.key("data"); out.append_subtree(array);
      out.key("reason"); out.value(""); out.key("status");
      out.value("available"); out.end();
    }
    out.end();
  }
  out.end();
  return out.finish();
}

int main() {
  try {
#if defined(__APPLE__)
    const std::string scratch = "/private/tmp";
#else
    const std::string scratch = "/tmp";
#endif
    const auto base = scratch + "/sbv-row-spool-" + std::to_string(::getpid());
    require(std::filesystem::create_directory(base), "fresh test scratch");
    auto small = [&] {
      s::RowSpool spool(base + "/small", {2048, 2}, {std::nullopt, 0});
      require(spool.size() == 0, "empty initial count");
      Json expected = Json::array();
      for (std::uint64_t i = 0; i < 31; ++i) {
        auto value = row(i); spool.append(value); expected.push_back(value);
        value["id"] = "changed-after-append";
      }
      require(!std::filesystem::exists(base + "/small/manifest.json"),
              "append has no scratch root");
      auto closed = spool.close();
      require(closed.row_count == 31 && spool.size() == 31, "exact closed count");
      require(closed.rows.read_value() == expected, "owned row bytes match");
      require(closed.array_verification.at("canonical_sha256") ==
                  k::sha256_hex(expected.dump()), "independent canonical array hash");
      require(spool.recovery().at("financial_result_published") == false,
              "scratch does not publish financial census");
      rejects([&] { spool.close(); }, "close only once");
      rejects([&] { spool.append(row(32)); }, "append after close refused");
      require(spool.recovery().at("state") == "closed",
              "refused post-close mutation keeps sealed state");
      return closed;
    }();
    require(small.rows.child(30).read_value() == row(30),
            "closed handle survives reader and spool destruction");
    auto rows = small.rows.children(28, 3);
    std::uint64_t ordinal = 28;
    while (auto value = rows.next()) {
      require(value->read_value() == row(ordinal++), "ordered tail cursor");
    }
    require(ordinal == 31, "tail range complete");
    rejects([&] { (void)small.rows.read_value({10, std::nullopt}); },
            "caller materialization byte budget");

    s::RowSpool copied(base + "/copied", {2048, 2});
    copied.append(small.rows.child(std::uint64_t{0})); copied.append(small.rows.child(30));
    const auto copied_rows = copied.close();
    require(copied_rows.rows.read_value() == Json::array({row(0), row(30)}),
            "owned handle append preserves rows");

    const auto final = descendant(base + "/descendant", small.rows, {2048, 2});
    std::filesystem::remove_all(base + "/small");
    s::ResultReader final_reader(final.reference, {std::nullopt, 0});
    require(final_reader.verify_closure().at("verification_extent") ==
                "full_logical_closure", "self-contained descendant after scratch removal");
    require(final_reader.select("/sections/signals/data").hash().at("canonical_sha256") ==
                small.array_verification.at("canonical_sha256"), "copied subtree identity");
    require(final_reader.select("/sections/replay/data").hash().at("canonical_sha256") ==
                small.array_verification.at("canonical_sha256"), "repeated logical subtree identity");
    const auto rebound = final_reader.select("/sections/signals/data").describe();
    require(rebound.at("reference").at("manifest_sha256") == final.reference.manifest_sha256,
            "descendant locator bound to new reference");

    s::RowSpool empty(base + "/empty"); const auto closed_empty = empty.close();
    require(closed_empty.row_count == 0 && closed_empty.rows.read_value() == Json::array(),
            "zero-row spool");
    require(closed_empty.array_verification.at("canonical_sha256") == k::sha256_hex("[]"),
            "zero-row array identity");
    Json structured = Json::array();
    for (std::size_t i = 0; i < 20; ++i)
      structured.push_back(std::string(300, static_cast<char>('a' + i)));
    s::RowSpool wide_row(base + "/wide-row", {2048, 2});
    wide_row.append(structured);
    require(wide_row.close().rows.child(std::uint64_t{0}).read_value() == structured,
            "single structured row spans selected page budget");
    Json deep = "depth-boundary";
    for (std::size_t depth = 0; depth < 60; ++depth)
      deep = Json::array({deep});
    s::RowSpool depth_ok(base + "/depth-ok");
    depth_ok.append(deep);
    require(depth_ok.close().rows.child(std::uint64_t{0}).read_value() == deep,
            "row depth matches portable signals position");
    s::RowSpool depth_bad(base + "/depth-bad");
    rejects([&] { depth_bad.append(Json::array({deep})); },
            "depth beyond existing portable profile rejected");
    rejects([&] { depth_bad.close(); }, "depth failure cannot publish prefix");
    {
      s::RowSpool unfinished(base + "/unfinished");
      for (std::uint64_t i = 0; i < 50; ++i) unfinished.append(row(i));
      // Simulate a provider failing at finish: the caller deliberately never seals.
    }
    require(!std::filesystem::exists(base + "/unfinished/manifest.json"),
            "provider failure cannot seal an emitted prefix");
    bool cancelled = false;
    s::RowSpool cancel(base + "/cancel", {}, {}, [&] {
      if (cancelled) throw std::runtime_error("selected cancellation");
    });
    cancel.append(row(0)); cancelled = true;
    rejects([&] { cancel.append(row(1)); }, "append cancellation"); cancelled = false;
    rejects([&] { cancel.close(); }, "failed append poisons prefix");
    require(cancel.recovery().at("state") == "failed" &&
                !std::filesystem::exists(base + "/cancel/manifest.json"),
            "cancelled spool retained incomplete");
    s::RowSpool invalid(base + "/invalid"); invalid.append(row(0));
    rejects([&] { invalid.append(Json{{"raw_number", 1}}); }, "numeric row rejected");
    rejects([&] { invalid.close(); }, "invalid partially written row poisons spool");
    require(!std::filesystem::exists(base + "/invalid/manifest.json"),
            "invalid row no root");
    rejects([&] { s::RowSpool occupied(base + "/empty"); }, "no overwrite");
    s::RowSpool cancelled_after_root(base + "/cancelled-after-root", {}, {}, [&] {
      if (std::filesystem::exists(base + "/cancelled-after-root/manifest.json"))
        throw std::runtime_error("cancellation after private publication");
    });
    cancelled_after_root.append(row(0));
    rejects([&] { cancelled_after_root.close(); }, "late private-close cancellation");
    require(cancelled_after_root.recovery().at("scratch").at("manifest_published") == true &&
                cancelled_after_root.recovery().at("financial_result_published") == false,
            "private root publication remains truthful after later failure");

    s::RowSpool many(base + "/many");
    constexpr std::uint64_t total = 65537;
    for (std::uint64_t i = 0; i < total; ++i) many.append(row(i));
    const auto large = many.close();
    require(large.row_count == total, "cross old signal and row counts without a cap");
    auto cursor = large.rows.children(); std::uint64_t seen = 0;
    while (auto item = cursor.next()) {
      const auto owned = item->read_value();
      if (owned.at("index") != std::to_string(seen) || owned != row(seen))
        throw std::runtime_error("large row sequence changed");
      ++seen;
    }
    require(seen == total, "all large rows retained and ordered");
    auto last = large.rows.children(total - 1, 1).next();
    require(last && last->read_value() == row(total - 1), "indexed final row accessible");
    std::cout << Json{{"status", "passed"}, {"checks", std::to_string(checks)},
                      {"large_rows", std::to_string(total)},
                      {"scratch", base}, {"large_scratch_receipt", large.scratch_receipt.json()},
                      {"cursor_progress", cursor.progress()}}.dump() << '\n';
    return 0;
  } catch (const std::exception &error) {
    std::cerr << error.what() << '\n'; return 1;
  }
}
