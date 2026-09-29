#include "common.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <symphony/sqav/databento/attempts.hpp>
#include <symphony/sqdv/checkpoint.hpp>
#include <symphony/sqpv/local_store.hpp>
#include <sys/stat.h>
using namespace symphony;
using sqv_admin::Json;
void check(bool b) {
  if (!b)
    throw std::runtime_error(
        "native administration fixture construction failed");
}
std::string root(const std::string &base, const std::string &leaf) {
  auto path = base + "/" + leaf;
  check(::mkdir(path.c_str(), 0700) == 0);
  return path;
}
Json store_selection(const std::string &path,
                     const sqfv::Generation &generation) {
  std::ifstream f(path + "/metadata", std::ios::binary);
  std::string raw((std::istreambuf_iterator<char>(f)), {});
  check(raw.size() > 16);
  std::uint64_t n = 0;
  for (std::size_t i = 8; i < 16; ++i)
    n = (n << 8) | static_cast<unsigned char>(raw[i]);
  check(n >= 32 && n < 65537 && n <= raw.size() - 16);
  auto reference =
      "sqmv1-sha256-" + sqv_admin::hex(std::string_view(raw).substr(
                            16 + static_cast<std::size_t>(n) - 32, 32));
  return {{"root", path},
          {"expected_metadata_reference", reference},
          {"expected_store_generation", sqv_admin::hex(generation)},
          {"max_read_bytes", "1048576"},
          {"max_batches", "32"}};
}
void request(const std::string &base, const std::string &name,
             const std::string &op, const std::string &owner, Json selection) {
  Json p{{"protocol", "symphony." + owner + "." + op + "-input.v1"},
         {"store", std::move(selection)}};
  if (owner == "sqav") {
    p["offset"] = "0";
    p["limit"] = "128";
  }
  std::ofstream f(base + "/" + name + ".json");
  f << p.dump(2) << '\n';
  check(f.good());
}
sqmv::Manifest retained_metadata(const std::string &path) {
  std::ifstream f(path + "/metadata", std::ios::binary);
  std::string raw((std::istreambuf_iterator<char>(f)), {});
  check(raw.size() > 16);
  std::uint64_t size = 0;
  for (std::size_t i = 8; i < 16; ++i)
    size = (size << 8) | static_cast<unsigned char>(raw[i]);
  check(size >= 32 && size <= 65536 && size <= raw.size() - 16);
  auto bytes = std::span<const std::uint8_t>(
      reinterpret_cast<const std::uint8_t *>(raw.data() + 16),
      static_cast<std::size_t>(size));
  auto reference = "sqmv1-sha256-" + sqv_admin::hex(bytes.last(32));
  sqmv::Manifest m;
  check(sqmv::Manifest::resolve(bytes, reference, {65536, 4096, 128}, m) ==
        sqmv::Status::ok);
  return m;
}
void integer(std::string &b, std::uint64_t n) {
  for (int i = 7; i >= 0; --i)
    b.push_back(static_cast<char>(n >> (8 * i)));
}
void field(std::string &b, std::string_view text) {
  integer(b, text.size());
  b += text;
}
void typed_fixture(const std::string &base, const std::string &name,
                   const std::string &source, const sqpv::Options &options,
                   std::string_view owner, std::string_view payload) {
  auto path = root(base, name);
  auto m = retained_metadata(source);
  sqpv::Store store;
  check(sqpv::Store::create(path, m, options, store) == sqpv::Status::ok);
  if (!payload.empty()) {
    sqfv::Context flow;
    check(sqfv::Context::create({65536, 73728, 65536, 1048576, 1}, flow) ==
          sqfv::Status::ok);
    sqfv::Descriptor descriptor;
    check(m.binding(descriptor.binding) == sqmv::Status::ok);
    descriptor.partition = options.partition;
    descriptor.producer_generation = options.producer_generation;
    descriptor.batch_sequence = 1;
    descriptor.record_count = 1;
    descriptor.source_binding = m.description().dataset_id;
    sqfv::Batch b;
    check(
        flow.prepare_copy(descriptor,
                          sqfv::ByteView(reinterpret_cast<const std::uint8_t *>(
                                             payload.data()),
                                         payload.size()),
                          b) == sqfv::Status::ok);
    sqpv::Receipt receipt;
    check(store.append(flow, b, receipt) == sqpv::Status::ok);
  }
  store.reset();
  request(base, name,
          owner == "sqdv" ? "checkpoint-inspect" : "attempts-inspect",
          std::string(owner), store_selection(path, options.store_generation));
}
int main(int argc, char **argv) try {
  check(argc == 2);
  std::string base = argv[1];
  check(::mkdir(base.c_str(), 0700) == 0);
  sqmv::Manifest m;
  sqmv::Description d{"fixture",
                      "revision",
                      "schema",
                      "layout",
                      "private:test",
                      "fixture",
                      {{sqmv::EvidenceRole::schema, "fixture", "schema"},
                       {sqmv::EvidenceRole::layout, "fixture", "layout"},
                       {sqmv::EvidenceRole::access, "fixture", "scope"}}};
  check(sqmv::Manifest::create(d, {65536, 4096, 128}, m) == sqmv::Status::ok);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 16U << 20, 16}, flow) ==
        sqfv::Status::ok);
  sqpv::Options options{"partition", {}, {}, 7, {73728, 1U << 20, 16}};
  options.producer_generation[0] = 1;
  options.store_generation[0] = 2;
  auto batch = [&](std::uint64_t seq) {
    sqfv::Descriptor descriptor;
    check(m.binding(descriptor.binding) == sqmv::Status::ok);
    descriptor.partition = options.partition;
    descriptor.producer_generation = options.producer_generation;
    descriptor.batch_sequence = seq;
    descriptor.record_count = 1;
    descriptor.source_binding = "fixture";
    std::array<std::uint8_t, 3> bytes{0, 255, 1};
    sqfv::Batch b;
    check(flow.prepare_copy(descriptor, bytes, b) == sqfv::Status::ok);
    return b;
  };
  for (const auto *name : {"store", "empty", "exhausted"}) {
    auto path = root(base, name);
    auto selected = options;
    if (std::string_view(name) == "exhausted")
      selected.first_sequence = UINT64_MAX;
    sqpv::Store s;
    check(sqpv::Store::create(path, m, selected, s) == sqpv::Status::ok);
    if (std::string_view(name) != "empty") {
      auto b = batch(selected.first_sequence);
      sqpv::Receipt receipt;
      check(s.append(flow, b, receipt) == sqpv::Status::ok);
    }
    s.reset();
    request(base, name, "store-inspect", "sqpv",
            store_selection(path, selected.store_generation));
  }
  auto checkpoint = root(base, "checkpoint");
  sqdv::Config config{"fixture-view",
                      "recipient",
                      "interface",
                      "partition",
                      options.producer_generation,
                      7,
                      sqdv::Profile::disposable};
  sqdv::Session session;
  check(sqdv::Session::create(flow, m, config, {65536, 4}, nullptr, nullptr,
                              session) == sqdv::Status::ok);
  sqdv::CheckpointOptions jo{{}, 8, 1U << 20};
  jo.generation[0] = 9;
  sqdv::CheckpointJournal journal;
  check(sqdv::CheckpointJournal::create(checkpoint, session, jo, journal) ==
        sqdv::Status::ok);
  auto b = batch(7);
  check(session.offer_live(b) == sqdv::Status::ok);
  sqdv::Delivery delivery;
  check(session.take(delivery) == sqdv::Status::ok);
  check(session.acknowledge_processed(delivery) == sqdv::Status::ok);
  sqdv::Checkpoint saved;
  check(journal.save(session, saved) == sqdv::Status::ok);
  journal.reset();
  request(base, "checkpoint", "checkpoint-inspect", "sqdv",
          store_selection(checkpoint, jo.generation));
  namespace db = sqav::databento;
  auto attempts = root(base, "attempts");
  db::AttemptBudget budget{{}, 5000000000, 15646000, 8, 1U << 20};
  budget.generation[0] = 3;
  db::AttemptLedger ledger;
  check(db::AttemptLedger::create(attempts, budget, ledger) ==
        db::AttemptStatus::ok);
  db::HistoricalPlan plan;
  check(db::HistoricalPlan::create({"GLBX.MDP3", {"ESH1"}, 1, 100, 10},
                                   {{65536, 4096, 100}, 1000, 8, 3, 60},
                                   plan) == db::Status::ok);
  db::AttemptQuote quote{"fixture:quote", 1000000000, 1000, 5000};
  db::AttemptTicket ticket;
  check(ledger.reserve(plan, "attempt-1", quote, 2000, ticket) ==
        db::AttemptStatus::ok);
  check(ticket.claim(plan, 2000) == db::AttemptStatus::ok);
  check(ledger.finish(ticket, db::AttemptOutcome::completed) ==
        db::AttemptStatus::ok);
  check(ledger.reserve(plan, "attempt-2", quote, 2000, ticket) ==
        db::AttemptStatus::ok);
  ledger.reset();
  request(base, "attempts", "attempts-inspect", "sqav",
          store_selection(attempts, budget.generation));
  sqpv::Options checkpoint_store{
      "checkpoints", jo.generation, jo.generation, 1, {16384, 1048576, 8}};
  typed_fixture(base, "checkpoint-empty", checkpoint, checkpoint_store, "sqdv",
                {});
  std::string bad_checkpoint("SQC\1", 4);
  integer(bad_checkpoint, 8);
  bad_checkpoint.push_back(0);
  typed_fixture(base, "checkpoint-invalid-baseline", checkpoint,
                checkpoint_store, "sqdv", bad_checkpoint);
  sqpv::Options attempt_store{"attempts",
                              budget.generation,
                              budget.generation,
                              1,
                              {65536, 1048576, 16}};
  typed_fixture(base, "attempts-empty", attempts, attempt_store, "sqav", {});
  std::string orphan("SQA\1\1", 5);
  field(orphan, "orphan");
  typed_fixture(base, "attempts-orphan-terminal", attempts, attempt_store,
                "sqav", orphan);
  for (const auto *name :
       {"attempts-overspend", "attempts-invalid-ordinal",
        "attempts-invalid-request", "attempts-invalid-time"}) {
    std::string bytes("SQA\1", 4);
    bytes.push_back(0);
    field(bytes, "attempt-1");
    field(bytes, std::string_view(name) == "attempts-invalid-request"
                     ? "sqdh1-sha256-wrong"
                     : plan.reference());
    field(bytes, plan.parameters());
    field(bytes, "fixture:quote");
    integer(bytes, std::string_view(name) == "attempts-overspend"
                       ? 6000000000ULL
                       : 1000000000ULL);
    integer(bytes, 1000);
    integer(bytes, 5000);
    integer(bytes,
            std::string_view(name) == "attempts-invalid-time" ? 5000 : 2000);
    bytes.push_back(std::string_view(name) == "attempts-invalid-ordinal" ? 2
                                                                         : 1);
    bytes.push_back(3);
    typed_fixture(base, name, attempts, attempt_store, "sqav", bytes);
  }
  std::cout << "Native writer fixtures created\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
