#include "request.hpp"
#include "store_reader.hpp"
#include <map>
namespace symphony::sqav::administration {
using namespace symphony::sqv_admin;
namespace storage = sqpv::inspection;
namespace {
struct Budget {
  std::uint64_t ceiling, prior;
};
Budget metadata(const storage::Snapshot &s) {
  const auto &d = s.metadata.description();
  auto revision = std::string_view(d.dataset_revision);
  require(revision.starts_with("budget:"));
  revision.remove_prefix(7);
  auto separator = revision.find(":prior:");
  require(separator != std::string_view::npos);
  Budget b{decimal(revision.substr(0, separator)),
           decimal(revision.substr(separator + 7))};
  require(b.ceiling > 0 && b.prior <= b.ceiling);
  sqmv::Description exact{
      "databento-historical-attempts",
      d.dataset_revision,
      "sqav-attempt-v1",
      "sqav-attempt-v1",
      "private:attempt-ledger",
      "sqav-databento-dbn-cpp",
      {{sqmv::EvidenceRole::schema, "sqav", "sqav-attempt-v1"},
       {sqmv::EvidenceRole::layout, "sqav", "sqav-attempt-v1"},
       {sqmv::EvidenceRole::access, "sqav", "private:attempt-ledger"}}};
  sqmv::Manifest expected;
  accepted(sqmv::Manifest::create(exact, {65536, 4096, 8}, expected));
  require(expected.reference() == s.metadata.reference() &&
          s.partition == "attempts" && s.first_sequence == 1 &&
          s.max_frame_bytes == 65536 && s.max_batches <= 8192 &&
          s.max_batches % 2 == 0 &&
          s.store_generation == s.producer_generation);
  return b;
}
bool token(std::string_view s) {
  return !s.empty() && s.size() <= 128 &&
         std::all_of(s.begin(), s.end(),
                     [](unsigned char c) { return c >= 33 && c <= 126; });
}
struct Attempt {
  std::string id, request;
  std::uint64_t cost = 0;
  unsigned ordinal = 0, maximum = 0, outcome = 0;
};
const char *outcome(unsigned n) {
  constexpr std::array<const char *, 5> names{
      "reserved", "completed", "rejected", "cancelled", "indeterminate"};
  require(n < names.size());
  return names[n];
}
} // namespace
Json administer(std::string_view, const Json &p, std::int64_t end) {
  fields(p, {"protocol", "store", "offset", "limit"});
  auto offset = u64(p, "offset"), limit = u64(p, "limit");
  require(offset <= 4096 && limit > 0 && limit <= 128);
  std::vector<Attempt> attempts;
  std::map<std::string, std::size_t> by_id;
  std::map<std::string, unsigned> ordinals;
  std::uint64_t reserved = 0;
  auto snapshot = storage::inspect(
      p.at("store"), end, [&](const auto &s, const auto &batch, auto payload) {
        auto budget = metadata(s);
        const auto &d = batch.descriptor();
        require(d.record_count == 1 &&
                d.source_binding == "databento-historical-attempts" &&
                d.source_position.empty());
        std::string_view b(reinterpret_cast<const char *>(payload.data()),
                           payload.size());
        require(storage::take(b, 4) == std::string_view("SQA\1", 4));
        auto status = static_cast<unsigned char>(storage::take(b, 1)[0]);
        require(status <= 4);
        auto id = storage::field(b, 128);
        require(token(id));
        if (status == 0) {
          require(!by_id.contains(id) && attempts.size() < s.max_batches / 2);
          auto request = storage::field(b, 128);
          auto parameters = storage::field(b, 32768);
          auto quote = storage::field(b, 128);
          auto cost = storage::take64(b), quoted = storage::take64(b),
               expires = storage::take64(b), admitted = storage::take64(b);
          require(b.size() == 2);
          auto ordinal = static_cast<unsigned char>(b[0]),
               maximum = static_cast<unsigned char>(b[1]);
          require(token(request) && token(quote) && !parameters.empty() &&
                  quoted > 0 && quoted <= admitted && admitted < expires &&
                  expires - quoted <= 86400000 && ordinal > 0 &&
                  ordinal <= maximum && maximum <= 8);
          require(
              request ==
              "sqdh1-sha256-" +
                  engine::sha256_hex(
                      "https://hist.databento.com/v0/timeseries.get_range?" +
                      parameters));
          require(cost <= budget.ceiling - budget.prior - reserved &&
                  ordinal == ordinals[request] + 1);
          ++ordinals[request];
          reserved += cost;
          by_id.emplace(id, attempts.size());
          attempts.push_back({id, request, cost, ordinal, maximum, 0});
        } else {
          require(b.empty() && by_id.contains(id));
          auto &prior = attempts[by_id.at(id)];
          require(prior.outcome == 0);
          prior.outcome = status;
        }
      });
  auto budget = metadata(snapshot);
  if (snapshot.recovery_required)
    refuse("sqv.store.recovery_required",
           "Attempt ledger requires explicit recovery before accounting", 3);
  Json page = Json::array();
  std::uint64_t unresolved = 0;
  for (std::size_t i = 0; i < attempts.size(); ++i) {
    const auto &a = attempts[i];
    if (a.outcome == 0 || a.outcome == 4)
      ++unresolved;
    if (i >= offset && i - offset < limit)
      page.push_back(
          {{"attempt_id", hex(a.id)},
           {"request_reference", a.request},
           {"reserved_ceiling_nano_usd", std::to_string(a.cost)},
           {"ordinal", std::to_string(a.ordinal)},
           {"maximum", std::to_string(a.maximum)},
           {"persisted_outcome", outcome(a.outcome)},
           {"restart_outcome", outcome(a.outcome == 0 ? 4 : a.outcome)}});
  }
  const auto returned = static_cast<std::uint64_t>(page.size());
  return {{"store", storage::summary(snapshot)},
          {"attempts", std::to_string(attempts.size())},
          {"unresolved", std::to_string(unresolved)},
          {"ceiling_nano_usd", std::to_string(budget.ceiling)},
          {"prior_charge_nano_usd", std::to_string(budget.prior)},
          {"charged_ceiling_nano_usd", std::to_string(budget.prior + reserved)},
          {"remaining_nano_usd",
           std::to_string(budget.ceiling - budget.prior - reserved)},
          {"offset", std::to_string(offset)},
          {"entries", page},
          {"next_offset", offset + returned < attempts.size()
                              ? Json(std::to_string(offset + returned))
                              : Json(nullptr)},
          {"accounting_scope", "conservative_reserved_ceilings"},
          {"execution_capability", "not_issued"},
          {"actual_provider_cost", "not_observed"}};
}
} // namespace symphony::sqav::administration
