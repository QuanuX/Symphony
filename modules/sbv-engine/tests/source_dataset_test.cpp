#include "census.hpp"
#include "dataset.hpp"
#include "public_fixture.hpp"
#include "source_owners.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/stat.h>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace sqav = symphony::sqav;
namespace db = sqav::databento;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("check failed at " + std::to_string(at.line()));
}
template <class F> void rejects(F f) {
  bool failed = false;
  try {
    f();
  } catch (const std::exception &) {
    failed = true;
  }
  check(failed);
}
J read(const std::string &path) {
  return J::parse(d::read_file(path, e::no_deadline));
}
const J &data(const J &r, const char *section) {
  return r.at("sections").at(section).at("data");
}
#include "fixture_requests.hpp"
int main(int argc, char **argv) try {
  check(argc == 3);
#ifdef __APPLE__
  char temp[] = "/private/tmp/sbv-retained-dataset-XXXXXX";
#else
  char temp[] = "/tmp/sbv-retained-dataset-XXXXXX";
#endif
  check(::mkdtemp(temp));
  const std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  const auto templates = read(argv[2]).at("templates");
  auto fixture = public_fixture::fixture();
  std::vector<unsigned char> bytes(fixture.begin(), fixture.begin() + 360);
  auto put = [&](std::size_t off, std::uint64_t n, unsigned size) {
    for (unsigned i = 0; i < size; ++i)
      bytes[off + i] = static_cast<unsigned char>(n >> (i * 8));
  };
  for (unsigned i = 0; i < 12; ++i) {
    const auto off = bytes.size();
    bytes.insert(bytes.end(), fixture.begin() + 360, fixture.begin() + 416);
    put(off + 8, 1609160400000000000ULL + i * 100, 8);
    put(off + 40, 1609160400000000000ULL + i * 100, 8);
    put(off + 24, (100 + i % 3) * 1000000000ULL, 8);
    put(off + 32, 1, 4);
    put(off + 52, i, 4);
    bytes[off + 38] = 'T';
    bytes[off + 36] = 128;
    bytes[off + 37] = 0;
  }
  const std::string raw(reinterpret_cast<const char *>(bytes.data()),
                        bytes.size());
  d::create_file(root + "/original.dbn", raw, e::no_deadline);
  check(::mkdir((root + "/store").c_str(), 0700) == 0);
  db::FileView view;
  check(db::FileView::inspect_dataset(bytes, {}, view) == db::Status::ok);
  (void)d::source_retain(request(root, raw, view.metadata()), e::no_deadline);
  const auto source_result = read(root + "/source.json");
  const auto retained = feed(root + "/source.json", source_result);
  J file{{"source_path", root + "/original.dbn"},
         {"source_sha256", e::sha256_hex(raw)},
         {"dataset", "GLBX.MDP3"}};
  J selected{{"retained_source", retained}};
  auto from_file = d::load_dataset(file, e::no_deadline);
  auto from_retained = d::load_dataset(selected, e::no_deadline);
  check(from_file->path == from_retained->path &&
        from_file->sha256 == from_retained->sha256 &&
        from_file->dataset_name == from_retained->dataset_name);
  check(from_file->events == from_retained->events &&
        from_retained->events.size() == 12);
  check(from_retained->load_buffer_bytes ==
        bytes.size() + from_retained->events.capacity() * sizeof(db::Mbo));
  check(from_file->source_delivery.is_null() &&
        !from_file->evidence(false).contains("load_buffer_scope"));
  check(from_retained->source_delivery.at("after_ack_before_release")
            .at("next_processed_sequence") == "8");
  check(from_retained->source_delivery.at("after_release")
            .at("outstanding_bytes") == "0");
  check(from_retained->evidence(false).at("mode") == "retained_source");
  check(from_retained->evidence(true).at("mode") == "resident");
  check(from_retained->evidence(true).at("source_reads_this_job") == "0");
  check(from_retained->evidence(true).at("source_delivery") ==
        from_retained->source_delivery);
  auto no_ack = selected;
  no_ack["retained_source"]["delivery"]["processed_ack"] = "none";
  check(d::load_dataset(no_ack, e::no_deadline)
            ->source_delivery.at("final_checkpoint")
            .at("next_sequence") == "7");
  auto exact = selected;
  exact["memory_budget_bytes"] = d::dec(from_retained->load_buffer_bytes);
  check(d::load_dataset(exact, e::no_deadline)->events ==
        from_retained->events);
  rejects([&] {
    auto p = exact;
    p["memory_budget_bytes"] = d::dec(from_retained->load_buffer_bytes - 1);
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = selected;
    p["dataset_limits"] = {{"max_source_bytes", d::dec(bytes.size() - 1)},
                           {"max_source_events", nullptr},
                           {"max_metadata_bytes", nullptr}};
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = selected;
    p["dataset_limits"] = {{"max_source_bytes", nullptr},
                           {"max_source_events", "11"},
                           {"max_metadata_bytes", nullptr}};
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = selected;
    p["dataset_limits"] = {{"max_source_bytes", nullptr},
                           {"max_source_events", nullptr},
                           {"max_metadata_bytes", "1"}};
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = selected;
    p["retained_source"]["reference"]["expected_sha256"] = std::string(64, '0');
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = file;
    p["retained_source"] = retained;
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] {
    auto p = file;
    p.erase("dataset");
    (void)d::load_dataset(p, e::no_deadline);
  });
  rejects([&] { (void)d::load_dataset(J::object(), e::no_deadline); });
  rejects([&] { (void)d::load_dataset(selected, 1); });
  auto with_source = [&](J p, bool is_retained) {
    for (const auto *k :
         {"source_path", "source_sha256", "dataset", "retained_source"})
      p.erase(k);
    p.update(is_retained ? selected : file);
    return p;
  };
  unsigned serial = 0;
  auto call = [&](const std::string &op, J p,
                  const d::Dataset *resident = nullptr) {
    p["output_path"] =
        root + "/" + op + "-" + std::to_string(++serial) + ".json";
    J receipt;
    if (op == "run")
      receipt = d::run(p, e::no_deadline, resident);
    else if (op == "evaluate")
      receipt = d::evaluate(p, e::no_deadline, resident);
    else if (op == "book")
      receipt = d::book(p, e::no_deadline, resident);
    else if (op == "generate_census")
      receipt = d::generate_census(p, e::no_deadline, resident);
    else
      receipt = s::dispatch(op, p, e::no_deadline);
    auto result = read(p.at("output_path"));
    s::validate_result(result);
    check(receipt.at("content_sha256") == result.at("content_sha256"));
    return std::pair{p.at("output_path").get<std::string>(), result};
  };
  J run{{"protocol", "symphony.sbv.run-input.v1"},
        {"criteria",
         {{"rule", "spaced_trades"},
          {"spacing_ns", "200"},
          {"min_trade_size", "1"},
          {"direction", "any"},
          {"max_signals", "4"}}},
        {"execution",
         {{"model", "none"},
          {"horizon_ns", "300"},
          {"price_offsets_nanos", J::array()},
          {"probability_numerator", "0"},
          {"probability_denominator", "1"}}},
        {"replay",
         {{"before_ns", "100"}, {"after_ns", "300"}, {"retain_events", true}}},
        {"studies", J::array({"signal_summary"})},
        {"workers", "1"},
        {"extensions", J::object()}};
  const auto [fp, fr] = call("run", with_source(run, false));
  const auto [rp, rr] = call("run", with_source(run, true));
  for (const auto *k : {"signals", "census", "execution", "replay", "studies",
                        "summary", "provenance"})
    check(data(fr, k) == data(rr, k));
  check(data(rr, "choices").contains("retained_source") &&
        !data(rr, "choices").contains("source_path"));
  check(d::census_evidence(fr) == d::census_evidence(rr));
  auto evaluation = with_source(templates.at("evaluate"), true);
  evaluation["census"] = {{"path", rp},
                          {"expected_sha256", rr.at("content_sha256")},
                          {"pointer", ""}};
  evaluation["model"]["horizon_ns"] = "300";
  evaluation["model"]["parameters"]["levels_per_side"] = "1";
  const auto [ep, er] = call("evaluate", evaluation);
  check(d::census_evidence(er) == d::census_evidence(rr));
  const auto [efp, efr] = call("evaluate", with_source(evaluation, false));
  for (const auto *k : {"signals", "census", "execution", "replay", "studies",
                        "summary", "provenance"})
    check(data(er, k) == data(efr, k));
  auto book = with_source(templates.at("book"), true);
  book["census_result"] = {{"path", rp},
                           {"expected_sha256", rr.at("content_sha256")}};
  book["signal_ids"] = J::array({data(rr, "signals").at(0).at("signal_id")});
  const auto [bp, br] = call("book", book);
  check(data(br, "provenance").at("source_path") == file.at("source_path"));
  const auto [bfp, bfr] = call("book", with_source(book, false));
  for (const auto *k : {"book_frames", "replay", "summary", "provenance"})
    check(data(br, k) == data(bfr, k));
  auto gen = with_source(templates.at("generate_census"), true);
  gen["provider"]["library"] = {
      {"path", argv[1]},
      {"expected_sha256",
       e::sha256_hex(d::read_file(argv[1], e::no_deadline))}};
  gen["provider"]["id"] = "sbv-test-provider";
  const auto [gp, gr] = call("generate_census", gen);
  check(d::census_evidence(gr).at("kind") == "native_provider");
  const auto [gfp, gfr] = call("generate_census", with_source(gen, false));
  for (const auto *k :
       {"signals", "census", "provider", "summary", "provenance"})
    check(data(gr, k) == data(gfr, k));
  // Derived artifacts retain the actual feed in source_context, never as new
  // delivery work in their own resources. Generic transform chains preserve it.
  auto analysis = templates.at("analyze");
  analysis["path"] = ep;
  analysis["expected_sha256"] = er.at("content_sha256");
  analysis["pointer"] = "/sections/signals/data";
  analysis["value_pointer"] = "/anchor_price_nanos";
  analysis["value_type"] = "integer";
  const auto [ap, ar] = call("analyze", analysis);
  check(data(ar, "source_context").at("dataset_feed") ==
        data(er, "resources").at("dataset_feed"));
  auto inherited_analysis = analysis;
  inherited_analysis["path"] = ap;
  inherited_analysis["expected_sha256"] = ar.at("content_sha256");
  inherited_analysis["pointer"] = "/sections/series/data";
  inherited_analysis["value_pointer"] = "/value";
  inherited_analysis["value_type"] = "rational";
  const auto [aap, aar] = call("analyze", inherited_analysis);
  check(data(aar, "source_context").at("dataset_feed") ==
        data(er, "resources").at("dataset_feed"));
  auto resampling = templates.at("resample");
  resampling["path"] = ap;
  resampling["expected_sha256"] = ar.at("content_sha256");
  resampling["sample_size"] = "2";
  resampling["replicates"] = "2";
  const auto [bsp, bsr] = call("resample", resampling);
  check(data(bsr, "source_context").at("dataset_feed") ==
        data(er, "resources").at("dataset_feed"));
  auto splitting = templates.at("split");
  splitting["path"] = ap;
  splitting["expected_sha256"] = ar.at("content_sha256");
  splitting["pointer"] = "/sections/signals/data";
  splitting["id_pointer"] = "/signal_id";
  splitting["start_pointer"] = "/available_ns";
  splitting["end_pointer"] = "/available_ns";
  splitting["plan"] = {{"kind", "expanding"},
                       {"train_start_ns", "1609160399999999999"},
                       {"first_train_end_ns", "1609160400000000300"},
                       {"first_fit_cutoff_ns", "1609160400000000300"},
                       {"gap_ns", "0"},
                       {"test_duration_ns", "400"},
                       {"step_ns", "400"},
                       {"fold_count", "1"}};
  const auto [sp, sr] = call("split", splitting);
  check(data(sr, "source_context").at("dataset_feed") ==
        data(er, "resources").at("dataset_feed"));
  auto liquid = templates.at("liquidity");
  liquid["path"] = bp;
  liquid["expected_sha256"] = br.at("content_sha256");
  liquid["orders"] = J::array();
  const auto [lp, lr] = call("liquidity", liquid);
  check(data(lr, "source_context").at("dataset_feed") ==
        data(br, "resources").at("dataset_feed"));
  // Resident jobs bind the same selection or the original tuple without I/O.
  check(std::filesystem::remove(root + "/original.dbn"));
  from_retained->bind(file);
  from_retained->bind(selected);
  rejects([&] {
    auto p = file;
    p["source_path"] = "/no/such/other.dbn";
    from_retained->bind(p);
  });
  rejects([&] {
    auto p = selected;
    p["retained_source"]["delivery"]["recipient_id"] = "other";
    from_retained->bind(p);
  });
  const auto [rmp, rmr] =
      call("run", with_source(run, false), from_retained.get());
  check(d::census_evidence(rmr) == d::census_evidence(rr));
  check(data(rmr, "resources").at("dataset_feed").at("source_delivery") ==
        from_retained->source_delivery);
  const auto [rep, rer] = call("evaluate", evaluation, from_retained.get());
  check(data(er, "execution") == data(rer, "execution"));
  const auto [rbp, rbr] = call("book", book, from_retained.get());
  check(data(br, "book_frames") == data(rbr, "book_frames"));
  const auto [rgp, rgr] = call("generate_census", gen, from_retained.get());
  check(data(gr, "census") == data(rgr, "census"));
  // Format-valid but receive-clock-invalid input must fail Dataset admission
  // before after_admission is reached. Fresh ephemeral delivery remains
  // replayable.
  const auto badroot = root + "/bad-decode";
  check(::mkdir(badroot.c_str(), 0700) == 0);
  check(::mkdir((badroot + "/store").c_str(), 0700) == 0);
  put(360 + 56 + 40, 1, 8);
  const std::string badraw(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  d::create_file(badroot + "/original.dbn", badraw, e::no_deadline);
  (void)d::source_retain(request(badroot, badraw, view.metadata()),
                         e::no_deadline);
  const auto badfeed =
      feed(badroot + "/source.json", read(badroot + "/source.json"));
  rejects([&] {
    (void)d::load_dataset(J{{"retained_source", badfeed}}, e::no_deadline);
  });
  {
    d::RetainedOriginal after_failure(badfeed, e::no_deadline);
    check(after_failure.original_bytes().size() == badraw.size());
  }
  // Result consumption has no dependency on retained ancestor or current store.
  check(std::filesystem::remove(root + "/source.json"));
  std::filesystem::rename(root + "/store", root + "/store-unavailable");
  from_retained->bind(file);
  from_retained->bind(selected);
  check(d::census_evidence(rr) == data(rr, "census"));
  check(d::census_evidence(er) == data(er, "census"));
  check(d::census_evidence(rmr) == data(rmr, "census"));
  rejects([&] {
    auto r = rr;
    r["sections"]["choices"]["data"]["retained_source"]["reference"]["path"] =
        "/different";
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"]["source_delivery"]
     ["source"]["original"]["source_sha256"] = std::string(64, '0');
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"]["source_delivery"]
     ["after_release"]["outstanding_bytes"] = "1";
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["provenance"]["data"]["source_path"] = "/different";
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"].erase("source_delivery");
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"]["source_bytes"] = "1";
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"]
     ["source_reads_this_job"] = "0";
    (void)d::census_evidence(r);
  });
  rejects([&] {
    auto r = rr;
    r["sections"]["resources"]["data"]["dataset_feed"]["load_buffer_scope"] =
        "all_process_memory";
    (void)d::census_evidence(r);
  });
  // Economics and composition re-admit retained delivery after all original
  // source/store/descriptor ancestors have disappeared. Compose then needs only
  // the economic artifact, not its evaluated-result or census ancestors.
  auto econ = templates.at("economics");
  econ["path"] = ep;
  econ["expected_sha256"] = er.at("content_sha256");
  econ["selections"][0]["signal_id"] =
      data(er, "signals").at(2).at("signal_id");
  const auto [ecp, ecr] = call("economics", econ);
  check(data(ecr, "economics").at(0).at("status") == "available");
  check(data(ecr, "source_context").at("dataset_feed") ==
        data(er, "resources").at("dataset_feed"));
  check(std::filesystem::remove(ep));
  check(std::filesystem::remove(rp));
  auto composition = templates.at("compose_economics");
  composition["components"][0]["source"] = {
      {"path", ecp},
      {"expected_sha256", ecr.at("content_sha256")},
      {"pointer", ""}};
  composition["components"][0]["signal_id"] =
      econ["selections"][0]["signal_id"];
  const auto [cp, cr] = call("compose_economics", composition);
  check(cr.at("sections").at("distributions").at("status") == "available");
  // A prior file context without a feed remains supported.
  auto file_econ = econ;
  file_econ["path"] = efp;
  file_econ["expected_sha256"] = efr.at("content_sha256");
  const auto [fecp, fecr] = call("economics", file_econ);
  auto legacy = fecr;
  legacy["sections"]["source_context"]["data"].erase("dataset_feed");
  legacy = s::seal_result(std::move(legacy));
  const auto legacyp = root + "/legacy-economics.json";
  d::create_file(legacyp, legacy.dump(), e::no_deadline);
  auto legacy_composition = composition;
  legacy_composition["components"][0]["source"] = {
      {"path", legacyp},
      {"expected_sha256", legacy.at("content_sha256")},
      {"pointer", ""}};
  const auto [lcp, lcr] = call("compose_economics", legacy_composition);
  check(data(lcr, "distributions") == data(cr, "distributions"));
  auto malformed_economics = [&](auto mutate) {
    auto bad = ecr;
    mutate(bad["sections"]["source_context"]["data"]);
    bad = s::seal_result(std::move(bad));
    const auto path =
        root + "/bad-economic-context-" + std::to_string(++serial) + ".json";
    d::create_file(path, bad.dump(), e::no_deadline);
    auto rejected = composition;
    rejected["components"][0]["source"] = {
        {"path", path},
        {"expected_sha256", bad.at("content_sha256")},
        {"pointer", ""}};
    rejects([&] { (void)call("compose_economics", rejected); });
  };
  malformed_economics([](J &context) { context.erase("dataset_feed"); });
  malformed_economics([](J &context) {
    context["dataset_feed"]["source_delivery"]["after_release"]
           ["outstanding_bytes"] = "1";
  });
  malformed_economics([](J &context) {
    context["choices"]["data"]["retained_source"]["reference"]["path"] =
        "/different";
  });
  std::cout << "retained dataset mirror: " << checks << " checks passed\n";
  return 0;
} catch (const std::exception &err) {
  std::cerr << err.what() << "\n";
  return 1;
}
