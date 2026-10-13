#include "../src/consumer_source.hpp"
#include "../src/dataset.hpp"
#include "../src/measurement.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <sys/resource.h>
#include <thread>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace m = d::measurement;
using J = s::Json;
using Clock = std::chrono::steady_clock;
unsigned checks = 0;
void check(bool ok,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("performance assertion " +
                             std::to_string(loc.line()));
}
auto now() { return Clock::now(); }
std::uint64_t ns(Clock::time_point start) {
  return static_cast<std::uint64_t>(
      std::chrono::duration_cast<std::chrono::nanoseconds>(now() - start)
          .count());
}
J read(const std::string &p) {
  return J::parse(d::read_file(p, e::no_deadline));
}
J phases(const m::Recorder &r) {
  J out = J::object();
  const char *names[] = {"source_read_identity", "decode_validate",
                         "compute_scratch", "final_publication"};
  for (std::size_t i = 0; i < 4; ++i)
    out[names[i]] = {{"exclusive_wall_ns", d::dec(r.phases[i].wall_ns)},
                     {"calls", d::dec(r.phases[i].calls)}};
  return out;
}
std::uint64_t total(const m::Recorder &r) {
  std::uint64_t t = 0;
  for (auto p : r.phases)
    t += p.wall_ns;
  return t;
}
J scientific(const J &r) {
  J selected = J::object();
  for (const auto *key : {"summary", "signals", "execution", "distributions",
                          "studies", "replay", "census"})
    selected[key] = r.at("sections").at(key);
  return selected;
}
void unit() {
  check(!m::recorder && !m::current);
  m::Recorder record;
  {
    m::Binding binding(record);
    auto start = now();
    {
      m::Scope outer(m::Phase::compute);
      {
        m::Scope child(m::Phase::decode);
        child.finish();
        child.finish();
      }
      {
        m::Scope child(m::Phase::publication);
      }
    }
    check(total(record) <= ns(start));
    check(record.phases[0].calls == 0 && record.phases[1].calls == 1 &&
          record.phases[2].calls == 1 && record.phases[3].calls == 1);
    std::thread other([] {
      check(!m::recorder && !m::current);
      m::Scope inactive(m::Phase::read);
    });
    other.join();
    check(record.phases[0].calls == 0);
    try {
      m::Scope child(m::Phase::read);
      throw std::runtime_error("injected");
    } catch (const std::exception &) {
    }
    check(!m::current && record.phases[0].calls == 1);
  }
  check(!m::recorder && !m::current);
  std::cout << "measurement scope checks " << checks << '\n';
}
int main(int argc, char **argv) try {
  if (argc == 1) {
    unit();
    return 0;
  }
  check(argc == 2);
  auto config = read(argv[1]);
  const auto root = d::str(config.at("output_directory"));
  std::filesystem::create_directories(root);
  const auto warmups = d::u64(config.at("warmups")),
             repeats = d::u64(config.at("repeats"));
  check(repeats > 0 && warmups <= UINT64_MAX - repeats);
  J rows = J::array(), loads = J::array();
  std::map<std::string, std::string> expected;
  for (const auto &input : config.at("inputs")) {
    const auto id = d::str(input.at("id"));
    auto baseline = read(input.at("request_path"));
    baseline.erase("output");
    baseline.erase("output_path");
    for (const auto &feed : config.at("feeds")) {
      check(feed == "file" || feed == "ram");
      std::unique_ptr<d::Dataset> loaded;
      if (feed == "ram") {
        m::Recorder load;
        auto begin = now();
        {
          m::Binding binding(load);
          loaded = d::load_dataset(baseline, e::no_deadline);
        }
        const auto wall = ns(begin);
        check(total(load) <= wall);
        check(load.phases[0].calls == 1 && load.phases[1].calls == 1);
        loads.push_back(
            {{"input", id},
             {"feed", "ram"},
             {"wall_ns", d::dec(wall)},
             {"phases", phases(load)},
             {"source_reads", "1"},
             {"decodes", "1"},
             {"decoded_bytes", d::dec(loaded->events.size() *
                                      sizeof(symphony::sqav::databento::Mbo))},
             {"load_buffer_bytes", d::dec(loaded->load_buffer_bytes)}});
      }
      for (const auto &storage : config.at("outputs"))
        for (const auto &workers : config.at("workers"))
          for (std::uint64_t iteration = 0; iteration < warmups + repeats;
               ++iteration) {
            check(storage == "legacy" || storage == "partitioned");
            auto p = baseline;
            p["workers"] = workers;
            const auto label = id + "-" + d::str(feed) + "-" + d::str(storage) +
                               "-w" + d::str(workers) + "-r" +
                               d::dec(iteration);
            const auto path = root + "/" + label;
            if (storage == "legacy")
              p["output_path"] = path + ".json";
            else
              p["output"] = {
                  {"kind", "partitioned"},
                  {"bundle_path", path},
                  {"workspace_path", path + "-work"},
                  {"write_options", config.at("write_options")},
                  {"scratch_read_options", config.at("read_options")}};
            d::create_file(path + "-request.json", p.dump(2), e::no_deadline);
            m::Recorder record;
            auto begin = now();
            J receipt;
            {
              m::Binding binding(record);
              receipt = d::run(p, e::no_deadline, loaded.get());
            }
            const auto produce = ns(begin);
            check(total(record) <= produce);
            const auto reads = feed == "ram" ? 0U : 1U;
            check(record.phases[0].calls == reads &&
                  record.phases[1].calls == reads &&
                  record.phases[2].calls == 1 && record.phases[3].calls == 1);
            check(receipt["status"] == "completed" ||
                  receipt["status"] == "partial");
            d::create_file(path + "-receipt.json", receipt.dump(2),
                           e::no_deadline);
            begin = now();
            J result;
            if (storage == "legacy")
              result = read(p["output_path"]);
            else {
              J tag{{"kind", "bundle"},
                    {"reference", receipt["storage"]["reference"]},
                    {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
                    {"read_options", config.at("read_options")}};
              result = d::consumer::source(J{{"source", tag}}, e::no_deadline)
                           .value.materialize();
            }
            s::validate_result(result);
            const auto export_wall = ns(begin);
            auto digest = e::sha256_hex(scientific(result).dump());
            auto [it, inserted] = expected.emplace(id, digest);
            check(inserted || it->second == digest);
            auto resources = result["sections"]["resources"]["data"];
            check(resources["dataset_feed"]["decodes_this_job"] ==
                  (feed == "ram" ? "0" : "1"));
            check(resources["actual_workers"] == workers);
            struct rusage usage{};
            check(getrusage(RUSAGE_SELF, &usage) == 0);
            rows.push_back(
                {{"label", label},
                 {"input", id},
                 {"feed", feed},
                 {"storage", storage},
                 {"workers", workers},
                 {"iteration", d::dec(iteration)},
                 {"warmup", iteration < warmups},
                 {"producer_wall_ns", d::dec(produce)},
                 {"phases", phases(record)},
                 {"unattributed_producer_ns", d::dec(produce - total(record))},
                 {"verified_materialization_ns", d::dec(export_wall)},
                 {"complete_work_ns", d::dec(produce + export_wall)},
                 {"scientific_sha256", digest},
                 {"result_sha256", result["content_sha256"]},
                 {"receipt_path", path + "-receipt.json"},
                 {"resources", resources},
                 {"process_peak_rss_platform_units", d::dec(usage.ru_maxrss)}});
          }
    }
  }
  auto report = d::base("R5.4 private benchmark evidence");
  report["sections"]["summary"] =
      d::section({{"rows", d::dec(rows.size())},
                  {"input_groups", d::dec(expected.size())},
                  {"scientific_equivalence", true}});
  report["sections"]["measurements"] = d::section(rows);
  report["sections"]["loads"] = d::section(loads);
  report["sections"]["choices"] = d::section(config);
  report["sections"]["diagnostics"] = d::section(J::array(
      {"Per-thread exclusive wall scopes; parallel compute is critical-path "
       "wall time, not sum of worker CPU time.",
       "Read includes identity validation and retained-owner admission. Decode "
       "includes DBN validation, allocation and admission completion.",
       "Compute includes scratch I/O and run envelope work; final publication "
       "includes validation, hashing and durable result output.",
       "Materialization verifies the entire logical hash; installed qxctl "
       "export is measured separately.",
       "RAM load is separately reported and must be included when amortizing "
       "complete workloads.",
       "Process peak RSS is cumulative, not per-case allocation. Apple rusage "
       "uses bytes; other hosts may use KiB.",
       "No cache eviction, cold-cache, NUMA, device, cluster or comparative "
       "speed claim."}));
  d::persist(report, {{"output_path", config.at("report_path")}}, "run",
             e::no_deadline);
  std::cout << "Verified " << rows.size() << " benchmark executions in "
            << expected.size() << " scientific groups\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
