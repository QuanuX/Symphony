#include "request.hpp"
#include "store_reader.hpp"
namespace symphony::sqdv::administration {
using namespace symphony::sqv_admin;
namespace storage = sqpv::inspection;
namespace {
struct Position {
  std::uint64_t next = 0;
  bool exhausted = false;
  bool operator==(const Position &) const = default;
};
Position metadata(const storage::Snapshot &s) {
  const auto &d = s.metadata.description();
  auto revision = std::string_view(d.dataset_revision);
  require(!d.dataset_id.empty() && d.dataset_id.size() <= 128 &&
          revision.starts_with("baseline:"));
  revision.remove_prefix(9);
  auto separator = revision.find(':');
  require(separator != std::string_view::npos);
  Position p{decimal(revision.substr(0, separator)), false};
  auto suffix = revision.substr(separator + 1);
  require(suffix == "open" || suffix == "exhausted");
  p.exhausted = suffix == "exhausted";
  require(!p.exhausted || p.next == UINT64_MAX);
  sqmv::Description exact{
      d.dataset_id,
      d.dataset_revision,
      "sqdv-checkpoint-v1",
      "sqdv-checkpoint-v1",
      "private-checkpoint:" + d.dataset_id,
      "sqdv-delivery-cpp",
      {{sqmv::EvidenceRole::schema, "sqdv", "sqdv-checkpoint-v1"},
       {sqmv::EvidenceRole::layout, "sqdv", "sqdv-checkpoint-v1"},
       {sqmv::EvidenceRole::access, "sqdv", d.dataset_id},
       {sqmv::EvidenceRole::source, "sqdv", d.dataset_id}}};
  sqmv::Manifest expected;
  accepted(sqmv::Manifest::create(exact, {65536, 4096, 8}, expected));
  require(expected.reference() == s.metadata.reference() &&
          s.partition == "checkpoints" && s.first_sequence == 1 &&
          s.max_frame_bytes == 16384 &&
          s.store_generation == s.producer_generation);
  return p;
}
Json position(const Position &p) {
  return {{"next_sequence", std::to_string(p.next)},
          {"sequence_exhausted", p.exhausted}};
}
} // namespace
Json administer(std::string_view, const Json &p, std::int64_t end) {
  fields(p, {"protocol", "store"});
  Position latest;
  std::uint64_t count = 0;
  auto snapshot = storage::inspect(
      p.at("store"), end, [&](const auto &s, const auto &batch, auto payload) {
        auto baseline = metadata(s);
        const auto &d = batch.descriptor();
        require(d.record_count == 1 &&
                d.source_binding == s.metadata.description().dataset_id &&
                d.source_position.empty());
        std::string_view b(reinterpret_cast<const char *>(payload.data()),
                           payload.size());
        require(storage::take(b, 4) == std::string_view("SQC\1", 4));
        Position next{storage::take64(b), false};
        require(b.size() == 1 && (b[0] == 0 || b[0] == 1));
        next.exhausted = b[0] != 0;
        require(!next.exhausted || next.next == UINT64_MAX);
        if (count == 0)
          require(next == baseline);
        else
          require(!latest.exhausted &&
                  (next.next > latest.next ||
                   (latest.next == UINT64_MAX && next.next == UINT64_MAX &&
                    next.exhausted)));
        latest = next;
        ++count;
      });
  auto baseline = metadata(snapshot);
  if (snapshot.recovery_required)
    refuse("sqv.store.recovery_required",
           "Checkpoint journal requires explicit recovery before projection",
           3);
  return {{"store", storage::summary(snapshot)},
          {"view_reference", hex(snapshot.metadata.description().dataset_id)},
          {"baseline", position(baseline)},
          {"latest_persisted", count ? position(latest) : Json(nullptr)},
          {"checkpoint_state", count ? "persisted" : "baseline_not_persisted"},
          {"acknowledgment_scope", "local_processing_only"},
          {"remote_commit", "not_established"}};
}
} // namespace symphony::sqdv::administration
