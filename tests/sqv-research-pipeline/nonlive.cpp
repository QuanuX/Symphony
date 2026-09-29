#include "public_fixture.hpp"
#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <symphony/sqav/databento/attempts.hpp>
#include <symphony/sqdv/checkpoint.hpp>
#include <unistd.h>
using namespace symphony;
namespace db = sqav::databento;
void check(bool b, const char *why) {
  if (!b) {
    std::fprintf(stderr, "non-live pipeline: %s\n", why);
    std::abort();
  }
}
struct Root {
  std::string path;
  Root() {
    char name[] = "/private/tmp/sqv-nonlive-XXXXXX";
    auto p = ::mkdtemp(name);
    check(p, "root");
    path = p;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
int main() {
  Root attempts, data, checkpoints;
  db::AttemptBudget budget{{}, 5'000'000'000, 15'646'000, 4, 1U << 20};
  budget.generation[0] = 2;
  db::AttemptLedger ledger;
  check(db::AttemptLedger::create(attempts.path, budget, ledger) ==
            db::AttemptStatus::ok,
        "ledger");
  auto bytes = public_fixture::fixture();
  db::Limits dl{65536, 4096, 100};
  db::FileView file;
  check(db::FileView::inspect(bytes, dl, file) == db::Status::ok, "fixture");
  const auto &meta = file.metadata();
  db::HistoricalPlan plan;
  check(db::HistoricalPlan::create({std::string(meta.dataset),
                                    {"ESH1"},
                                    meta.start,
                                    meta.end,
                                    meta.limit},
                                   {dl, 86'400'000'000'000ULL, 8, 3, 60},
                                   plan) == db::Status::ok,
        "plan");
  db::AttemptTicket ticket;
  check(ledger.reserve(plan, "fixture-attempt",
                       {"fixture:cost-ceiling", 1000, 100, 1000}, 200,
                       ticket) == db::AttemptStatus::ok,
        "reserve");
  check(ticket.claim(plan, 201) == db::AttemptStatus::ok, "claim once");
  db::HistoricalResponse response;
  check(db::HistoricalResponse::begin(plan, ticket.ordinal(), 200, {},
                                      response) == db::Status::ok,
        "response");
  check(response.append(bytes) == db::Status::ok, "append");
  db::HistoricalReport report;
  check(response.finish(db::TransportEnd::complete, report) == db::Status::ok,
        "finish response");
  check(ledger.finish(ticket, db::AttemptOutcome::completed) ==
            db::AttemptStatus::ok,
        "finish attempt");
  sqav::Capture capture;
  const sqav::Limits cl{65536, 16384, 4096};
  check(response.capture({"fixture-attempt",
                          "offline-fixture",
                          "public-v0.68.0",
                          "private:fixture",
                          {sqav::TimeRole::acquisition, "2026-09-28",
                           "iso-date", "UTC-calendar", "day", "offline"}},
                         cl, capture) == db::Status::ok,
        "capture");
  sqmv::Manifest manifest;
  check(capture.metadata(
            false, {sqmv::EvidenceRole::access, "fixture", "private:fixture"},
            {65536, 4096, 128}, manifest) == sqav::Status::ok,
        "metadata");
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 8}, flow) ==
            sqfv::Status::ok,
        "flow");
  sqav::Position position{"nonlive", {}, 1};
  position.producer_generation[0] = 1;
  sqfv::Batch batch;
  check(capture.prepare(flow, manifest, position, batch) == sqav::Status::ok,
        "prepare");
  sqpv::Options options{position.partition,
                        position.producer_generation,
                        {},
                        1,
                        {73728, 1U << 20, 8}};
  options.store_generation[0] = 3;
  sqdv::Config config{"nonlive-view",
                      "fixture-consumer",
                      "capture",
                      position.partition,
                      position.producer_generation,
                      1,
                      sqdv::Profile::asynchronous_retention};
  sqdv::CheckpointOptions checkpoint_options{{}, 8, 1U << 20};
  checkpoint_options.generation[0] = 4;
  sqdv::Checkpoint baseline, saved;
  {
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create_async(data.path, manifest, options, flow,
                                             {2, 147456},
                                             source) == sqdv::Status::ok,
          "async source");
    sqdv::Session session;
    check(sqdv::Session::create(flow, manifest, config, {65536, 2}, &source,
                                nullptr, session) == sqdv::Status::ok,
          "session");
    check(session.checkpoint(baseline) == sqdv::Status::ok, "baseline");
    sqdv::CheckpointJournal journal;
    check(sqdv::CheckpointJournal::create(checkpoints.path, session,
                                          checkpoint_options,
                                          journal) == sqdv::Status::ok,
          "journal");
    sqdv::QueuedBatch queued;
    check(source.enqueue(batch, queued) == sqdv::Status::ok, "enqueue");
    check(session.offer_preview(queued) == sqdv::Status::ok, "preview");
    sqdv::Delivery delivery;
    check(session.take(delivery) == sqdv::Status::ok, "take");
    check(session.acknowledge_processed(delivery) == sqdv::Status::ok,
          "acknowledge");
    check(source.finish_retention() == sqdv::Status::ok, "drain");
    check(journal.save_retained(session, saved) == sqdv::Status::ok &&
              saved.next_sequence == 2,
          "persist covered checkpoint");
  }
  ledger.reset();
  check(db::AttemptLedger::open(attempts.path, budget, ledger) ==
            db::AttemptStatus::ok,
        "reopen budget");
  db::AttemptSnapshot cost;
  check(ledger.snapshot(cost) == db::AttemptStatus::ok &&
            cost.charged_ceiling_nano_usd == 15'647'000 && cost.attempts == 1,
        "recovered charge");
  db::AttemptTicket duplicate;
  check(ledger.reserve(plan, "fixture-attempt",
                       {"fixture:cost-ceiling", 1000, 100, 1000}, 202,
                       duplicate) == db::AttemptStatus::conflict,
        "no repeated execution");
  {
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::open_async(data.path, manifest, options, flow,
                                           {2, 147456},
                                           source) == sqdv::Status::ok,
          "reopen data");
    sqdv::CheckpointJournal journal;
    check(sqdv::CheckpointJournal::open(checkpoints.path, baseline,
                                        checkpoint_options,
                                        journal) == sqdv::Status::ok,
          "reopen journal");
    check(journal.load(saved) == sqdv::Status::ok, "load durable progress");
    sqdv::Session resumed;
    check(sqdv::Session::create(flow, manifest, config, {65536, 2}, &source,
                                &saved, resumed) == sqdv::Status::ok,
          "resume");
    sqdv::SessionStats stats;
    check(resumed.stats(stats) == sqdv::Status::ok &&
              stats.next_offer_sequence == 2,
          "no repeated delivery");
    // Explicit historical replay uses the original baseline; it cannot rewind
    // the durable journal.
    sqdv::Session replay;
    check(sqdv::Session::create(flow, manifest, config, {65536, 2}, &source,
                                &baseline, replay) == sqdv::Status::ok,
          "explicit replay");
    check(journal.save_retained(replay, saved) == sqdv::Status::stale,
          "no rollback");
    check(replay.offer_next(flow) == sqdv::Status::ok, "retained read");
    sqdv::Delivery delivery;
    check(replay.take(delivery) == sqdv::Status::ok, "replay take");
    sqav::Capture restored;
    check(sqav::Capture::from_delivery(delivery.payload(),
                                       delivery.descriptor(), manifest, cl,
                                       restored) == sqav::Status::ok,
          "restore capture");
    check(restored.reference() == capture.reference() &&
              restored.description().coverage == sqav::Coverage::partial &&
              std::ranges::equal(restored.original(), bytes),
          "exact bytes and coverage");
  }
  std::puts("non-live pipeline: durable attempt, bounded response, coverage, "
            "retention, checkpoint and reopened replay passed; no network");
}
