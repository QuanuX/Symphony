#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/consumer_source.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace store = s::result_store;
using J = s::Json;
std::uint64_t checks = 0;
void check(bool good,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!good)
    throw std::runtime_error("partitioned execution assertion " +
                             std::to_string(loc.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
store::Reference bundle_reference(const J &r) {
  auto j = r.at("storage").at("reference");
  return {j.at("manifest_path"), j.at("manifest_sha256"),
          j.at("content_sha256")};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates = read(argv[1]).at("templates");
  const bool large = std::getenv("SYMPHONY_SBV_EXECUTION_LARGE");
  const bool checkpoint_only = std::getenv("SYMPHONY_SBV_CHECKPOINT_ONLY");
  char tmp[] = "/private/tmp/sbv-execution-partitioned-XXXXXX";
  check(::mkdtemp(tmp));
  const std::string root = tmp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  const J options{{"max_page_bytes", nullptr}, {"cache_bytes", "33554432"}};
  unsigned serial = 0;
  auto output = [&](const std::string &name) {
    auto path = root + "/" + name + "-" + d::dec(serial++);
    return J{
        {"kind", "partitioned"},
        {"bundle_path", path},
        {"workspace_path", path + "-work"},
        {"write_options", {{"page_bytes", "32768"}, {"index_fanout", "16"}}},
        {"scratch_read_options", options}};
  };
  auto publish = [&](J body, const std::string &name) {
    if (body["sections"]["summary"]["status"] != "available")
      body["sections"]["summary"] = d::section(J{{"fixture", name}});
    auto receipt = d::partitioned_result(
        {{"output", output(name)}}, "book", e::no_deadline,
        [&](d::PartitionedOutput &) { return s::logical::Value(body); });
    check(receipt.at("status") != "recovery_required");
    return receipt;
  };
  auto tag = [&](const J &receipt, const std::string &pointer = "") {
    return J{{"kind", "bundle"},
             {"reference", receipt.at("storage").at("reference")},
             {"selector", {{"kind", "pointer"}, {"pointer", pointer}}},
             {"read_options", options}};
  };
  auto call = [&](const std::string &op, J p) {
    p.erase("output_path");
    p["output"] = output(op);
    std::cout << "begin " << op << ' ' << serial << std::endl;
    auto r = s::dispatch(op, p, e::no_deadline);
    if (r.at("status") == "recovery_required")
      throw std::runtime_error(r.dump());
    return r;
  };
  auto body_of = [&](const J &receipt) {
    store::ResultReader reader(bundle_reference(receipt),
                               {std::nullopt, 33554432});
    return reader.select_node(0).read_value();
  };
  auto legacy = [&](const std::string &op, J p) {
    p.erase("output");
    p["output_path"] = root + "/legacy-" + d::dec(serial++) + ".json";
    s::dispatch(op, p, e::no_deadline);
    return read(p.at("output_path"));
  };
  const auto public_bytes = public_fixture::fixture();
  const std::uint64_t t = 1609160400000000000ULL;
  // Snapshot reset, two priced levels, then no-change event ends. The
  // checkpoint case instead adds distinct orders and selects only its last two
  // events.
  auto fixture = [&](std::uint64_t n, bool checkpoint) {
    const auto id = d::dec(serial++);
    std::vector<unsigned char> bytes(public_bytes.begin(),
                                     public_bytes.begin() + 360);
    J signals = J::array(), ids = J::array();
    const auto count = checkpoint ? n + 2 : n + 3;
    for (std::uint64_t i = 0; i < count; ++i) {
      const auto offset = bytes.size();
      bytes.insert(bytes.end(), public_bytes.begin() + 360,
                   public_bytes.begin() + 416);
      auto put = [&](unsigned o, std::uint64_t value, unsigned width) {
        for (unsigned j = 0; j < width; ++j)
          bytes[offset + o + j] = static_cast<unsigned char>(value >> (j * 8));
      };
      put(8, t + i * 100, 8);
      put(40, t + i * 100, 8);
      put(52, i, 4);
      put(16, i, 8);
      const bool add = i > 0 && (checkpoint ? i <= n : i < 3);
      bytes[offset + 38] = i == 0 ? 'R' : add ? 'A' : 'N';
      bytes[offset + 39] = add ? (i % 2 ? 'B' : 'A') : 'N';
      put(24, add ? (i % 2 ? 99 : 101) : 0, 8);
      put(32, add ? (checkpoint ? 1 : 10) : 0, 4);
      bytes[offset + 36] = (checkpoint ? i < n : i < 2)     ? 40
                           : (checkpoint ? i == n : i == 2) ? 168
                                                            : 128;
      bytes[offset + 37] = 0;
      if (checkpoint ? i >= n : i >= 3) {
        const auto signal = "s" + d::dec(i);
        ids.push_back(signal);
        signals.push_back({{"signal_id", signal},
                           {"source_ordinal", d::dec(i)},
                           {"available_ns", d::dec(t + i * 100)},
                           {"anchor_price_nanos", "100"},
                           {"causal_end_ordinal_exclusive", d::dec(i + 1)},
                           {"context_reference", "future caller context"}});
      }
    }
    const std::string raw(reinterpret_cast<const char *>(bytes.data()),
                          bytes.size()),
        path = root + "/" + id + ".dbn", sha = e::sha256_hex(raw);
    d::create_file(path, raw, e::no_deadline);
    J producer{{"id", "independent-mechanical-execution"},
               {"version", "1"},
               {"artifact_sha256", ""},
               {"reproducibility", "uncaptured"}};
    J census{{"protocol", "symphony.sbv.external-census.v1"},
             {"source_sha256", sha},
             {"mode", "causal_declared"},
             {"producer", producer},
             {"signals", signals}};
    auto b = d::base("independent constant displayed liquidity; mechanical, "
                     "not calibrated market evidence");
    b["sections"]["signals"] = d::section(signals);
    b["sections"]["choices"] =
        d::section({{"protocol", "symphony.sbv.evaluate-input.v1"},
                    {"source_sha256", sha},
                    {"dataset", "GLBX.MDP3"},
                    {"census", census}});
    b["sections"]["summary"] =
        d::section({{"closed_census", true},
                    {"signal_count", d::dec(signals.size())},
                    {"causality", "causal_declared"},
                    {"census_sha256", e::sha256_hex(census.dump())}});
    b["sections"]["provenance"] = d::section({{"source_sha256", sha},
                                              {"dataset", "GLBX.MDP3"},
                                              {"instrument_id", "5482"},
                                              {"census_producer", producer}});
    const auto sealed = s::seal_result(b);
    const auto legacy_path = root + "/" + id + "-census.json";
    d::create_file(legacy_path, sealed.dump(), e::no_deadline);
    auto request = templates.at("book");
    request["source_path"] = path;
    request["source_sha256"] = sha;
    request["dataset"] = "GLBX.MDP3";
    request["census_result"] = {
        {"path", legacy_path},
        {"expected_sha256", sealed.at("content_sha256")}};
    request["signal_ids"] = ids;
    request["initial_state"] = {{"mode", "uninitialized"}};
    request["on_anomaly"] = "reject";
    request["frames"] = {
        {"cadence", "signals"}, {"levels_per_side", "1"}, {"maximum", nullptr}};
    request["replay"] = {
        {"before_ns", "0"}, {"after_ns", "0"}, {"retain_events", true}};
    request["emit_checkpoint"] = true;
    return std::tuple{request, publish(b, "census"), ids};
  };
  if (!checkpoint_only) {
    const std::uint64_t n = large ? 4097 : 5;
    auto [book, census, ids] = fixture(n, false);
    auto original = book;
    const auto legacy_book = large ? J() : legacy("book", book);
    book["census_result"] = tag(census);
    book["signal_ids"] = {{"kind", "all"}};
    auto br = call("book", book);
    store::ResultReader book_reader(bundle_reference(br),
                                    {std::nullopt, 33554432});
    check(book_reader.select("/sections/book_frames/data")
              .describe()["children"] == d::dec(n));
    check(book_reader
              .select("/sections/book_frames/data/0/data/asks/0/price_nanos")
              .read_value() == "101");
    check(book_reader.select("/sections/signals/data/0/context_reference")
              .read_value() == "future caller context");
    if (!large) {
      const auto actual = body_of(br);
      for (auto name : {"signals", "book_frames", "book_checkpoint", "replay"})
        check(actual["sections"][name] == legacy_book["sections"][name]);
      auto selection = d::base("caller signal ids");
      selection["sections"]["ids"] = d::section(ids);
      auto ir = publish(selection, "ids");
      book["signal_ids"] = {{"kind", "referenced_ids"},
                            {"source", tag(ir, "/sections/ids/data")},
                            {"order", "census"}};
      auto selected = call("book", book);
      check(body_of(selected)["sections"]["book_frames"] ==
            actual["sections"]["book_frames"]);
    }
    J orders = J::array(), selections = J::array();
    for (std::uint64_t i = 0; i < n; ++i) {
      auto o = templates.at("liquidity").at("orders")[0];
      o["order_id"] = "o" + d::dec(i);
      o["signal_id"] = ids[i];
      o["frame_source_ordinal"] = d::dec(i + 3);
      o["quantity"] = "1";
      o["activation_probability"] = {
          {"status", "supplied"},
          {"value", {{"numerator", "3"}, {"denominator", "5"}}}};
      orders.push_back(o);
      auto a = templates.at("allocation_economics").at("selections")[0];
      a["order_id"] = o["order_id"];
      a["valuation"]["mark_price_nanos"] = "110";
      a["valuation"]["mark_available_ns"] = d::dec(t + (n + 4) * 100);
      a["valuation"]["mark_evidence"] = "mechanical constant oracle";
      a["valuation"]["price_unit_nanos"] = "1";
      a["valuation"]["pnl_unit"] = "mechanical unit";
      selections.push_back(a);
    }
    auto inputs = d::base("caller order and valuation rows");
    inputs["sections"]["orders"] = d::section(orders);
    inputs["sections"]["selections"] = d::section(selections);
    auto inputs_receipt = publish(inputs, "inputs");
    auto liquidity = templates.at("liquidity");
    liquidity.erase("path");
    liquidity.erase("expected_sha256");
    liquidity["source"] = tag(br);
    liquidity["orders"] = {
        {"kind", "referenced_rows"},
        {"source", tag(inputs_receipt, "/sections/orders/data")}};
    liquidity["workers"] = "4";
    auto lr = call("liquidity", liquidity);
    store::ResultReader lr_reader(bundle_reference(lr),
                                  {std::nullopt, 33554432});
    check(lr_reader.select("/sections/execution/data").describe()["children"] ==
          d::dec(n));
    check(lr_reader.select("/sections/execution/data/0/data/filled_quantity")
              .read_value() == "1");
    check(lr_reader
              .select("/sections/execution/data/0/data/"
                      "notional_price_nanos_times_quantity")
              .read_value() == "101");
    if (!large) {
      auto q = liquidity;
      q["orders"] = orders;
      auto old = legacy("liquidity", q), modern = body_of(lr);
      for (auto name :
           {"execution", "studies", "signals", "book_frames", "replay"})
        check(old["sections"][name] == modern["sections"][name]);
      q["workers"] = "1";
      auto one = call("liquidity", q);
      check(body_of(one)["sections"]["execution"] ==
            modern["sections"]["execution"]);
      q["liquidity_mode"] = "shared_snapshot";
      for (auto &o : q["orders"]) {
        o["frame_source_ordinal"] = d::dec(n + 2);
        o["activation_probability"]["value"] = {{"numerator", "1"},
                                                {"denominator", "1"}};
      }
      q["workers"] = "65";
      auto shared = call("liquidity", q);
      check(
          body_of(shared)["sections"]["resources"]["data"]["actual_workers"] ==
          "1");
    }
    // Source-independent downstream consumption: the owned liquidity result
    // retains expanded order choices and their exact selected-value identity.
    std::filesystem::remove_all(bundle_reference(br).manifest_path.substr(
        0, bundle_reference(br).manifest_path.rfind('/')));
    std::filesystem::remove_all(bundle_reference(census).manifest_path.substr(
        0, bundle_reference(census).manifest_path.rfind('/')));
    std::filesystem::remove(original.at("source_path").get<std::string>());
    auto allocation = templates.at("allocation_economics");
    allocation.erase("path");
    allocation.erase("expected_sha256");
    allocation["source"] = tag(lr);
    allocation["selections"] = {
        {"kind", "referenced_rows"},
        {"source", tag(inputs_receipt, "/sections/selections/data")}};
    auto ar = call("allocation_economics", allocation);
    store::ResultReader ar_reader(bundle_reference(ar),
                                  {std::nullopt, 33554432});
    check(ar_reader.select("/sections/economics/data").describe()["children"] ==
          d::dec(n));
    check(
        ar_reader
            .select("/sections/economics/data/0/conditional/value/net_markout")
            .read_value() == J({{"numerator", "9"}, {"denominator", "1"}}));
    if (!large) {
      auto q = allocation;
      q["selections"] = selections;
      auto old = legacy("allocation_economics", q), modern = body_of(ar);
      for (auto name :
           {"economics", "studies", "execution", "replay", "book_frames"})
        check(old["sections"][name] == modern["sections"][name]);
    }
    std::filesystem::remove_all(
        bundle_reference(inputs_receipt)
            .manifest_path.substr(
                0, bundle_reference(inputs_receipt).manifest_path.rfind('/')));
    auto standalone = allocation;
    standalone["selections"] = selections;
    auto rerun = call("allocation_economics", standalone);
    check(rerun["summary"] == ar["summary"]);
    std::filesystem::remove_all(bundle_reference(lr).manifest_path.substr(
        0, bundle_reference(lr).manifest_path.rfind('/')));
    store::ResultReader portable(bundle_reference(ar), {std::nullopt, 0});
    check(portable.verify_closure()["verification_extent"] ==
          "full_logical_closure");
    std::cout << "execution chain rows " << n << " passed" << std::endl;
  }
  if (large) {
    constexpr std::uint64_t n = 200001;
    auto [book, census, ids] = fixture(n, true);
    book["census_result"] = tag(census);
    book["signal_ids"] = J::array({ids[0]});
    auto prefix = call("book", book);
    store::ResultReader before(bundle_reference(prefix),
                               {std::nullopt, 33554432});
    check(before.select("/sections/book_checkpoint/data/orders")
              .describe()["children"] == d::dec(n));
    book["signal_ids"] = J::array({ids[1]});
    book["initial_state"] = {{"mode", "checkpoint"}, {"result", tag(prefix)}};
    auto continuation = call("book", book);
    store::ResultReader after(bundle_reference(continuation),
                              {std::nullopt, 33554432});
    check(
        s::logical::Value(after.select("/sections/book_checkpoint/data/orders"))
            .sha256() ==
        s::logical::Value(
            before.select("/sections/book_checkpoint/data/orders"))
            .sha256());
    check(after.select("/sections/book_frames/data/0/data/order_count")
              .read_value() == d::dec(n));
    check(after.select("/sections/book_frames/data/0/data/bids/0/size")
              .read_value() == "100001");
    check(after.select("/sections/book_frames/data/0/data/asks/0/size")
              .read_value() == "100000");
    std::filesystem::remove_all(bundle_reference(prefix).manifest_path.substr(
        0, bundle_reference(prefix).manifest_path.rfind('/')));
    store::ResultReader portable(bundle_reference(continuation),
                                 {std::nullopt, 0});
    check(portable.verify_closure()["verification_extent"] ==
          "full_logical_closure");
    std::cout << "checkpoint orders " << n << " hydration passed" << std::endl;
  }
  std::cout << "execution checks " << checks << " large " << large << std::endl;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
