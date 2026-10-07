#ifdef __APPLE__
#define SBV_TEST_TMP "/private/tmp"
#else
#define SBV_TEST_TMP "/tmp"
#endif
#include "logical_value.hpp"
#include "row_spool.hpp"
#include <atomic>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <symphony/knowledge/engine/digest.hpp>
#include <thread>
#include <unistd.h>
namespace l = symphony::sbv::logical;
namespace s = symphony::sbv::result_store;
namespace k = symphony::knowledge::engine;
using Json = s::Json;
std::uint64_t checks = 0;
void require(bool x, const char *s) {
  ++checks;
  if (!x)
    throw std::runtime_error(s);
}
template <class F> void rejects(F f, const char *s) {
  bool failed = false;
  try {
    f();
  } catch (const std::exception &) {
    failed = true;
  }
  require(failed, s);
}
Json body() {
  Json sections = Json::object();
  for (auto key : {"summary", "signals", "execution", "distributions",
                   "studies", "replay", "comparisons", "search", "resources",
                   "diagnostics", "choices", "provenance"})
    sections[key] = {{"data", nullptr},
                     {"reason", "not selected"},
                     {"status", "not_selected"}};
  return {{"protocol", "symphony.sbv.result.v1"},
          {"origin", "logical-graft-test"},
          {"status", "completed"},
          {"sections", sections}};
}
int main() {
  try {
    char pattern[] = SBV_TEST_TMP "/sbv-logical-XXXXXX";
    auto p = ::mkdtemp(pattern);
    require(p, "tempdir");
    std::string base = p;
    Json value = {
        {"empty", Json::array()},
        {"nested",
         {{"é/~", Json::array({true, nullptr, "18446744073709551615"})}}},
        {"z", "end"}};
    l::Value original(value);
    require(original.sha256() == k::sha256_hex(value.dump()),
            "independent JSON hash");
    require(original.materialize() == value, "owned JSON exact");
    auto nested = original.at("nested").at("é/~");
    require(nested.at(std::uint64_t{2}).materialize() == "18446744073709551615",
            "owned nested array");
    auto object = original.with("new", l::Value("\n\t深")).without("z");
    auto expected = value;
    expected["new"] = "\n\t深";
    expected.erase("z");
    require(object.materialize() == expected, "immutable object graft");
    require(original.materialize() == value, "original unchanged");
    require(object.sha256() == k::sha256_hex(expected.dump()),
            "graft independent hash");
    auto lifetime = [&] {
      l::Value temporary(Json::array({Json{{"alive", "yes"}}}));
      return temporary.at(std::uint64_t{0});
    }();
    require(lifetime.at("alive").materialize() == "yes",
            "JSON child owns ancestor lifetime");
    auto cursor = nested.children(1, 2);
    require(cursor.next()->value.materialize().is_null(), "range null");
    require(cursor.next()->edge == "2" && !cursor.next(), "range terminal");
    rejects([&] { nested.children(4); }, "range starts beyond end");
    rejects([&] { nested.children(1, 3); }, "range extends beyond end");
    rejects([&] { nested.at("bad"); }, "typed child mismatch");
    rejects([&] { original.at("missing"); }, "missing member");
    require(object.contains("new") && !object.contains("z"),
            "object membership");
    rejects([&] { original.materialize({std::uint64_t{2}, {}}); },
            "byte budget");
    rejects([&] { original.materialize({{}, std::uint64_t{2}}); },
            "node budget");
    auto count = original.export_json([](std::string_view) {});
    require(original.materialize(
                {std::stoull(count.at("bytes").get<std::string>()),
                 std::stoull(count.at("nodes").get<std::string>())}) == value,
            "exact selected budgets");
    rejects(
        [&] {
          original.export_json(
              [](std::string_view) { throw std::runtime_error("sink"); });
        },
        "sink failure");
    rejects(
        [&] {
          original.sha256({}, [] { throw std::runtime_error("cancel"); });
        },
        "checkpoint cancellation");
    rejects([&] { l::Value(123).sha256(); }, "numeric literal rejected");
    rejects([&] { l::Value(std::string(65537, 'x')).sha256(); },
            "string profile enforced");
    Json deep = nullptr;
    for (int i = 0; i < 65; ++i)
      deep = Json::array({deep});
    rejects([&] { l::Value(deep).sha256(); }, "depth profile enforced");
    bool cancelled = false;
    std::string prefix;
    rejects(
        [&] {
          l::Value("end").export_json(
              [&](std::string_view b) {
                prefix.append(b);
                cancelled = true;
              },
              {},
              [&] {
                if (cancelled)
                  throw std::runtime_error("cancel");
              });
        },
        "sink cancellation withholds final byte");
    require(prefix == "\"end", "incomplete scalar prefix only");
    original.visit({});
    require(true, "omitted visitor callbacks accepted");
    s::RowSpool spool(base + "/scratch", {2048, 2});
    Json rows = Json::array();
    for (unsigned i = 0; i < 257; ++i) {
      Json row = {{"id", std::to_string(i)}, {"value", std::string(97, 'a')}};
      rows.push_back(row);
      spool.append(row);
    }
    auto closed = spool.close();
    l::Value array(closed.rows);
    require(array.size() == 257 && array.kind() == "array",
            "owned native array");
    require(array.sha256() == k::sha256_hex(rows.dump()),
            "native array exact digest");
    auto child = array.at(std::uint64_t{256});
    require(child.materialize() == rows.back(), "native tail row");
    auto skeleton = body();
    auto result = l::Value(skeleton).with(
        "sections", l::Value(skeleton.at("sections"))
                        .with("signals", l::Value(Json{{"data", nullptr},
                                                       {"reason", ""},
                                                       {"status", "available"}})
                                             .with("data", array)));
    auto full = skeleton;
    full["sections"]["signals"] = {
        {"data", rows}, {"reason", ""}, {"status", "available"}};
    require(result.sha256() == k::sha256_hex(full.dump()),
            "composed body independent hash");
    s::ResultWriter writer(base + "/result", {2048, 2});
    result.write(writer);
    auto receipt = writer.finish();
    require(receipt.reference.content_sha256 == result.sha256(),
            "writer matches logical body");
    std::filesystem::remove_all(base + "/scratch");
    s::ResultReader reader(receipt.reference);
    reader.verify_closure();
    l::Value sealed(reader.select(""));
    full["content_sha256"] = receipt.reference.content_sha256;
    require(sealed.materialize() == full,
            "portable graft after scratch deletion");
    require(sealed.without("content_sha256").sha256() ==
                receipt.reference.content_sha256,
            "explicit body hash domain");
    require(sealed.sha256() == k::sha256_hex(full.dump()),
            "full hash includes virtual digest");
    auto selected = sealed.at("sections").at("signals").at("data");
    auto c = selected.children();
    unsigned ordinal = 0;
    while (auto v = c.next()) {
      require(v->edge == std::to_string(ordinal) &&
                  v->value.materialize() == rows.at(ordinal),
              "ordered native row");
      ++ordinal;
    }
    require(ordinal == 257, "all rows");
    auto composed_array = l::Value::array(
        {l::Value("header"), selected, l::Value(Json::array())});
    require(composed_array.materialize() ==
                Json::array({"header", rows, Json::array()}),
            "composed array preserves streamed child");
    auto array_slice = composed_array.children(1, 1);
    require(array_slice.next()->value.sha256() == selected.sha256() &&
                !array_slice.next(),
            "composed array cursor");
    auto independent_a = selected.independent_reader({std::nullopt, 0});
    auto independent_b = selected.independent_reader({std::nullopt, 0});
    std::atomic<bool> readers_ok = true;
    auto consume = [&](l::Value owned) {
      try {
        auto scan = owned.children();
        std::size_t index = 0;
        while (auto value = scan.next()) {
          if (value->value.materialize() != rows.at(index++))
            readers_ok = false;
        }
        if (index != rows.size())
          readers_ok = false;
      } catch (...) {
        readers_ok = false;
      }
    };
    {
      std::jthread a(consume, independent_a), b(consume, independent_b);
    }
    require(readers_ok,
            "independent readers retain exact concurrent row order");
    std::cout
        << Json{{"checks", std::to_string(checks)},
                {"status", "passed"},
                {"directory", base},
                {"claim_scope",
                 "external logical composition and reader/spool foundation"}}
               .dump()
        << "\n";
  } catch (const std::exception &e) {
    std::cerr << e.what() << "\n";
    return 1;
  }
}
