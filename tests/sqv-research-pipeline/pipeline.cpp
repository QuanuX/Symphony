#include <symphony/sqav/capture.hpp>
#include <symphony/sqtv/integer_conversion.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <algorithm>
#include <array>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <thread>
#include <unistd.h>
using namespace symphony;
namespace {
void check(bool value, const char* label) {
  if (!value) { std::fprintf(stderr, "SQV pipeline: %s\n", label); std::abort(); }
}
struct Root {
  std::string path;
  Root() { char name[] = "/private/tmp/sqv20-pipeline-XXXXXX";
    auto p = ::mkdtemp(name); check(p, "private root"); path = p; }
  ~Root() { std::filesystem::remove_all(path); }
};
sqfv::Context context() {
  sqfv::Context out;
  check(sqfv::Context::create({1<<20, 2<<20, 8192, 32<<20, 16}, out) == sqfv::Status::ok, "context");
  return out;
}
sqpv::Options options() {
  sqpv::Options o; o.partition = "partition:sample"; o.producer_generation[0] = 1;
  o.store_generation[0] = 2; o.first_sequence = 7; o.limits = {2<<20, 32<<20, 128}; return o;
}
sqdv::Config config(const sqpv::Options& o, sqdv::Profile profile) {
  return {"view:research", "consumer:offline", "interface:exact-bytes-v1", o.partition,
    o.producer_generation, o.first_sequence, profile};
}
sqav::Capture capture() {
  sqav::Description d;
  d.source = {"provider:synthetic", "interface:bytes", "version:1", "operation:sample", "adapter:fixture",
    "version:1", "dataset:integers", "revision:1", "selection:sample", std::string(sqtv::integer_schema),
    "sqtv-int-v1-i16-le", "scope:private"};
  d.attempt_id = "attempt:1"; d.attribution_ref = "producer:fixture";
  d.source_position = "source:42"; d.coverage = sqav::Coverage::partial;
  d.coverage_scope = "selection:sample"; d.coverage_evidence_ref = "evidence:partial";
  d.source_record_count = 3;
  d.times = {{sqav::TimeRole::acquisition, "2026-09-28T00:00:00Z", "iso8601", "clock:fixture", "second", "evidence:clock"}};
  std::array<std::uint8_t,6> raw{1,0,255,255,0,128};
  sqav::Capture out;
  check(sqav::Capture::create(d, raw, {1<<20,65536,4096}, out) == sqav::Status::ok, "capture");
  return out;
}
sqmv::Manifest metadata(const sqav::Capture& c, bool raw) {
  sqmv::Manifest out;
  check(c.metadata(raw, {sqmv::EvidenceRole::access, "principal:fixture", "grant:fixture"},
      {65536,4096,128}, out) == sqav::Status::ok, "derived metadata");
  return out;
}
void pipeline() {
  Root raw_root, derived_root;
  auto c = capture(); auto envelope = metadata(c, false); auto original = metadata(c, true);
  auto ctx = context(); const auto o = options();
  sqav::Position position{o.partition, o.producer_generation, 7};
  sqfv::Batch preserved, input;
  check(c.prepare(ctx, envelope, position, preserved) == sqav::Status::ok, "prepare preserved envelope");
  check(c.prepare_original(ctx, original, position, input) == sqav::Status::ok, "prepare original representation");
  sqtv::Result converted;
  check(sqtv::Result::convert(ctx, input, original, {32,sqtv::Signedness::signed_integer,sqtv::ByteOrder::big},
      {o.partition,o.producer_generation,7}, {100,1<<20,1<<20}, converted) == sqtv::Status::ok, "convert captured integers");
  const auto lineage = converted.metadata().description().evidence;
  check(std::any_of(lineage.begin(),lineage.end(),[&](const auto& e){return e.evidence_ref == c.reference();}), "capture lineage survives conversion");
  check(std::any_of(lineage.begin(),lineage.end(),[](const auto& e){return e.role == sqmv::EvidenceRole::coverage && e.evidence_ref == "evidence:partial";}), "coverage survives");
  sqdv::RetainedSource raw_store, derived_store;
  check(sqdv::RetainedSource::create_async(raw_root.path,envelope,o,ctx,{4,1<<20},raw_store)==sqdv::Status::ok,"raw asynchronous store");
  check(sqdv::RetainedSource::create_async(derived_root.path,converted.metadata(),o,ctx,{4,1<<20},derived_store)==sqdv::Status::ok,"converted asynchronous store");
  sqdv::Session preview, slow;
  auto cfg = config(o,sqdv::Profile::asynchronous_retention);
  check(sqdv::Session::create(ctx,converted.metadata(),cfg,{1024,2},&derived_store,nullptr,preview)==sqdv::Status::ok,"preview");
  cfg.recipient_id = "consumer:slow";
  check(sqdv::Session::create(ctx,converted.metadata(),cfg,{12,1},&derived_store,nullptr,slow)==sqdv::Status::ok,"independent preview");
  sqdv::QueuedBatch raw_ticket, ticket;
  check(raw_store.enqueue(preserved,raw_ticket)==sqdv::Status::ok,"queue original");
  check(derived_store.enqueue(converted.batch(),ticket)==sqdv::Status::ok,"queue conversion");
  check(preview.offer_preview(raw_ticket)==sqdv::Status::binding_mismatch,"cross-store ticket refuses");
  check(preview.offer_preview(ticket)==sqdv::Status::ok,"preview admission proof");
  check(slow.offer_preview(ticket)==sqdv::Status::ok,"slow branch");
  sqdv::Delivery delivered;
  check(preview.take(delivered)==sqdv::Status::ok,"preview take");
  const std::array<std::uint8_t,12> expected{0,0,0,1,255,255,255,255,255,255,128,0};
  check(std::ranges::equal(delivered.payload(),expected),"exact transformed bytes");
  check(delivered.retention_receipt()==nullptr,"preview never invents durability");
  check(preview.acknowledge_processed(delivered)==sqdv::Status::ok,"processing acknowledgment");
  check(raw_store.finish_retention()==sqdv::Status::ok && derived_store.finish_retention()==sqdv::Status::ok,"drain both writers");
  sqpv::AsyncSnapshot status;
  check(derived_store.retention_status(status)==sqdv::Status::ok && status.confirmed.next_sequence==8 && status.pending_batches==0 && !status.accepting,"confirmed watermark");
  sqdv::Checkpoint checkpoint;
  check(preview.checkpoint(checkpoint)==sqdv::Status::ok && checkpoint.next_sequence==8,"processing checkpoint");
  sqmv::Manifest output_metadata;
  check(converted.metadata().retain(output_metadata)==sqmv::Status::ok,"retain metadata");
  // Queue pins and contexts must survive producer handle release.
  preview.reset(); slow.reset(); ticket.reset(); raw_ticket.reset(); delivered.reset();
  derived_store.reset(); raw_store.reset(); converted = {}; input = {}; preserved = {}; ctx = {};
  auto replay_context = context();
  check(sqdv::RetainedSource::open_async(derived_root.path,output_metadata,o,replay_context,{4,1<<20},derived_store)==sqdv::Status::ok,"reopen confirmed converted store");
  cfg = config(o,sqdv::Profile::asynchronous_retention);
  check(sqdv::Session::create(replay_context,output_metadata,cfg,{1024,2},&derived_store,nullptr,preview)==sqdv::Status::ok,"restart view");
  check(preview.offer_next(replay_context)==sqdv::Status::ok && preview.take(delivered)==sqdv::Status::ok,"retained catchup");
  check(delivered.origin()==sqdv::Origin::retained && delivered.retention_receipt() && std::ranges::equal(delivered.payload(),expected),"real retained receipt and bytes");
  preview.reset(); delivered.reset();
  check(sqdv::Session::create(replay_context,output_metadata,cfg,{1024,2},&derived_store,&checkpoint,preview)==sqdv::Status::ok,"resume exact preview checkpoint");
  check(preview.offer_next(replay_context)==sqdv::Status::missing,"no fabricated next batch");
  sqdv::RetainedSource recovered_raw;
  check(sqdv::RetainedSource::open(raw_root.path,envelope,o,recovered_raw)==sqdv::Status::ok,"reopen raw evidence");
  sqdv::Session raw_session;
  cfg = config(o,sqdv::Profile::retained_before_delivery);
  check(sqdv::Session::create(replay_context,envelope,cfg,{1<<20,2},&recovered_raw,nullptr,raw_session)==sqdv::Status::ok,"raw replay session");
  check(raw_session.offer_next(replay_context)==sqdv::Status::ok && raw_session.take(delivered)==sqdv::Status::ok,"raw replay");
  sqav::Capture reconstructed;
  check(sqav::Capture::from_delivery(delivered.payload(),delivered.descriptor(),envelope,{1<<20,65536,4096},reconstructed)==sqav::Status::ok && reconstructed.reference()==c.reference(),"exact original capture survives full pipeline");
}
void queue_failures() {
  auto c = capture(); auto m = metadata(c,false); auto ctx = context(); auto o = options();
  Root root; sqpv::AsyncStore store;
  o.limits.max_batches = 1;
  check(sqpv::AsyncStore::create(root.path,m,o,ctx,{4,1<<20},store)==sqpv::Status::ok,"bounded store");
  sqfv::Batch a,b,wrong;
  check(c.prepare(ctx,m,{o.partition,o.producer_generation,7},a)==sqav::Status::ok,"batch7");
  check(c.prepare(ctx,m,{o.partition,o.producer_generation,8},b)==sqav::Status::ok,"batch8");
  check(c.prepare(ctx,m,{"wrong",o.producer_generation,7},wrong)==sqav::Status::ok,"wrong batch");
  check(store.submit(wrong)==sqpv::Status::binding_mismatch,"wrong identity");
  check(store.submit(b)==sqpv::Status::gap,"gap");
  check(store.submit(a)==sqpv::Status::ok,"admit7");
  check(store.submit(b)==sqpv::Status::ok,"admit8 before storage bound reached");
  ctx = {}; a = {}; b = {}; wrong = {};
  check(store.finish()==sqpv::Status::limit,"storage exhaustion is explicit failure");
  sqpv::AsyncSnapshot status;
  check(store.snapshot(status)==sqpv::Status::ok && status.failure==sqpv::Status::limit && status.confirmed.next_sequence==8 && status.pending_batches==1 && !status.accepting,"lag and failure remain visible");
  store.reset();
  auto reopened_context = context();
  check(sqpv::AsyncStore::open(root.path,m,o,reopened_context,{4,1<<20},store)==sqpv::Status::ok,"recovery uses committed prefix");
  check(store.snapshot(status)==sqpv::Status::ok && status.next_admission_sequence==8 && !status.pending_batches,"volatile admission not restored");
  check(store.finish()==sqpv::Status::ok,"empty drain");
  Root tiny; sqpv::AsyncStore limited;
  check(sqpv::AsyncStore::create(tiny.path,m,options(),reopened_context,{1,1},limited)==sqpv::Status::ok,"small queue byte bound");
  sqfv::Batch batch; check(c.prepare(reopened_context,m,{o.partition,o.producer_generation,7},batch)==sqav::Status::ok,"oversize batch");
  check(limited.submit(batch)==sqpv::Status::limit,"oversize refuses before queue");
}
void metadata_refusals() {
  auto c = capture(); auto m = metadata(c,true); auto ctx = context(); auto o = options();
  sqfv::Batch output;
  auto altered = m.description(); altered.dataset_revision = "wrong";
  sqmv::Manifest wrong;
  check(sqmv::Manifest::create(altered,{65536,4096,128},wrong)==sqmv::Status::ok,"wrong metadata fixture");
  check(c.prepare_original(ctx,wrong,{o.partition,o.producer_generation,7},output)==sqav::Status::binding_mismatch && !output,"source binding refuses");
  altered = m.description(); std::erase_if(altered.evidence,[](const auto& e){return e.role==sqmv::EvidenceRole::lineage;});
  check(sqmv::Manifest::create(altered,{65536,4096,128},wrong)==sqmv::Status::ok,"missing lineage fixture");
  check(c.prepare_original(ctx,wrong,{o.partition,o.producer_generation,7},output)==sqav::Status::binding_mismatch,"missing capture lineage refuses");
  auto prior = std::string(m.reference());
  check(c.metadata(true,{sqmv::EvidenceRole::schema,"p","e"},{65536,4096,128},m)==sqav::Status::invalid_argument && m.reference()==prior,"metadata output preserved");
  sqfv::Context retained; check(ctx.retain(retained)==sqfv::Status::ok,"retain context");
  sqfv::ContextStats before,after; check(ctx.stats(before)==sqfv::Status::ok,"initial accounting");
  retained = {}; check(ctx.stats(after)==sqfv::Status::ok && after.allocation_bytes<before.allocation_bytes,"context handle charge released");
}
}
int main() { pipeline(); queue_failures(); metadata_refusals(); std::puts("SQV pipeline: capture/metadata/conversion/preview/retention/recovery and refusal scenarios passed"); }
