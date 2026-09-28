#include <algorithm>
#include <mutex>
#include <new>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/sqav/databento/attempts.hpp>
#include <unistd.h>
namespace symphony::sqav::databento {
namespace {
using AS = AttemptStatus;
bool token(std::string_view s) noexcept {
  return !s.empty() && s.size() <= 128 &&
         std::all_of(s.begin(), s.end(),
                     [](unsigned char c) { return c >= 33 && c <= 126; });
}
bool terminal(AttemptOutcome o) noexcept {
  return o >= AttemptOutcome::completed && o <= AttemptOutcome::indeterminate;
}
AS mapped(sqpv::Status s) noexcept {
  switch (s) {
#define MAP(x)                                                                 \
  case sqpv::Status::x:                                                        \
    return AS::x
    MAP(ok);
    MAP(duplicate);
    MAP(limit);
    MAP(busy);
    MAP(conflict);
    MAP(stale);
    MAP(missing);
    MAP(corrupt);
    MAP(unsafe_path);
    MAP(io_error);
    MAP(outcome_uncertain);
    MAP(closed);
    MAP(no_memory);
#undef MAP
  case sqpv::Status::invalid_argument:
    return AS::invalid_argument;
  case sqpv::Status::binding_mismatch:
    return AS::conflict;
  default:
    return AS::internal_error;
  }
}
AS flow(sqfv::Status s) noexcept {
  return s == sqfv::Status::ok          ? AS::ok
         : s == sqfv::Status::no_memory ? AS::no_memory
         : s == sqfv::Status::limit     ? AS::limit
                                        : AS::internal_error;
}
void integer(std::vector<std::uint8_t> &b, std::uint64_t n) {
  for (unsigned i = 0; i < 8; ++i)
    b.push_back(static_cast<std::uint8_t>(n >> (56 - 8 * i)));
}
void field(std::vector<std::uint8_t> &b, std::string_view s) {
  integer(b, s.size());
  b.insert(b.end(), s.begin(), s.end());
}
struct Cursor {
  ByteView b;
  std::size_t pos = 0;
  bool integer(std::uint64_t &n) noexcept {
    if (b.size() - pos < 8)
      return false;
    n = 0;
    for (unsigned i = 0; i < 8; ++i)
      n = (n << 8) | b[pos++];
    return true;
  }
  bool field(std::string &s, std::size_t max) {
    std::uint64_t n;
    if (!integer(n) || n > max || n > b.size() - pos)
      return false;
    s.assign(reinterpret_cast<const char *>(b.data() + pos),
             static_cast<std::size_t>(n));
    pos += n;
    return true;
  }
};
struct Entry {
  std::string id, request, parameters, quote;
  std::uint64_t cost = 0, quoted = 0, expires = 0, admitted = 0;
  std::uint8_t ordinal = 0, maximum = 0;
  AttemptOutcome outcome = AttemptOutcome::reserved;
  bool claimed = false;
};
std::vector<std::uint8_t> encode(const Entry &e, bool finish) {
  std::vector<std::uint8_t> b{
      'S', 'Q', 'A', 1,
      static_cast<std::uint8_t>(finish ? e.outcome : AttemptOutcome::reserved)};
  field(b, e.id);
  if (!finish) {
    field(b, e.request);
    field(b, e.parameters);
    field(b, e.quote);
    integer(b, e.cost);
    integer(b, e.quoted);
    integer(b, e.expires);
    integer(b, e.admitted);
    b.push_back(e.ordinal);
    b.push_back(e.maximum);
  }
  return b;
}
bool decode(ByteView b, Entry &e) {
  if (b.size() < 5 || b[0] != 'S' || b[1] != 'Q' || b[2] != 'A' || b[3] != 1 ||
      b[4] > 4)
    return false;
  e.outcome = static_cast<AttemptOutcome>(b[4]);
  Cursor c{b, 5};
  if (!c.field(e.id, 128) || !token(e.id))
    return false;
  if (e.outcome == AttemptOutcome::reserved) {
    if (!c.field(e.request, 128) || !c.field(e.parameters, 32768) ||
        !c.field(e.quote, 128) || !c.integer(e.cost) || !c.integer(e.quoted) ||
        !c.integer(e.expires) || !c.integer(e.admitted) ||
        b.size() - c.pos != 2)
      return false;
    e.ordinal = b[c.pos++];
    e.maximum = b[c.pos++];
    if (!token(e.request) || !token(e.quote) || e.parameters.empty() ||
        !e.quoted || e.quoted > e.admitted || e.admitted >= e.expires ||
        e.expires - e.quoted > 86400000 || !e.ordinal ||
        e.ordinal > e.maximum || e.maximum > 8)
      return false;
  }
  return c.pos == b.size();
}
} // namespace
namespace detail {
struct AttemptLedgerState {
  pid_t pid = ::getpid();
  mutable std::mutex mutex;
  sqpv::Store store;
  sqfv::Context context;
  sqmv::Manifest manifest;
  sqfv::Descriptor descriptor;
  AttemptBudget budget;
  std::vector<Entry> entries;
  std::uint64_t charged = 0, records = 0;
  bool uncertain = false;
  AS append(const Entry &entry, bool finish) {
    auto bytes = encode(entry, finish);
    descriptor.batch_sequence = records + 1;
    sqfv::Batch batch;
    auto s = flow(context.prepare_copy(descriptor, bytes, batch));
    if (s != AS::ok)
      return s;
    sqpv::Receipt receipt;
    s = mapped(store.append(context, batch, receipt));
    if (s == AS::outcome_uncertain)
      uncertain = true;
    if (s == AS::ok)
      ++records;
    return s;
  }
};
struct AttemptTicketState {
  std::weak_ptr<AttemptLedgerState> owner;
  std::size_t index = 0;
  std::uint8_t ordinal = 0;
};
} // namespace detail
namespace {
AS initialize(bool create, const std::string &root, const AttemptBudget &b,
              std::shared_ptr<detail::AttemptLedgerState> &out) {
  if (!b.ceiling_nano_usd || b.prior_charge_nano_usd > b.ceiling_nano_usd ||
      !b.max_attempts || b.max_attempts > 4096 || !b.max_store_bytes ||
      !std::any_of(b.generation.begin(), b.generation.end(),
                   [](auto x) { return x != 0; }))
    return AS::invalid_argument;
  auto s = std::make_shared<detail::AttemptLedgerState>();
  s->budget = b;
  s->charged = b.prior_charge_nano_usd;
  s->entries.reserve(b.max_attempts);
  auto status =
      flow(sqfv::Context::create({49152, 65536, 4096, 262144, 1}, s->context));
  if (status != AS::ok)
    return status;
  sqmv::Description d{
      "databento-historical-attempts",
      "budget:" + std::to_string(b.ceiling_nano_usd) +
          ":prior:" + std::to_string(b.prior_charge_nano_usd),
      "sqav-attempt-v1",
      "sqav-attempt-v1",
      "private:attempt-ledger",
      "sqav-databento-dbn-cpp",
      {{sqmv::EvidenceRole::schema, "sqav", "sqav-attempt-v1"},
       {sqmv::EvidenceRole::layout, "sqav", "sqav-attempt-v1"},
       {sqmv::EvidenceRole::access, "sqav", "private:attempt-ledger"}}};
  auto ms = sqmv::Manifest::create(d, {65536, 4096, 8}, s->manifest);
  if (ms != sqmv::Status::ok)
    return ms == sqmv::Status::no_memory ? AS::no_memory : AS::invalid_argument;
  ms = s->manifest.binding(s->descriptor.binding);
  if (ms != sqmv::Status::ok)
    return AS::no_memory;
  s->descriptor.partition = "attempts";
  s->descriptor.source_binding = "databento-historical-attempts";
  s->descriptor.producer_generation = b.generation;
  s->descriptor.record_count = 1;
  sqpv::Options options{"attempts",
                        b.generation,
                        b.generation,
                        1,
                        {65536, b.max_store_bytes, b.max_attempts * 2}};
  status =
      mapped(create ? sqpv::Store::create(root, s->manifest, options, s->store)
                    : sqpv::Store::open(root, s->manifest, options, s->store));
  if (status != AS::ok)
    return status;
  sqpv::Snapshot snapshot;
  status = mapped(s->store.snapshot(snapshot));
  if (status != AS::ok)
    return status;
  for (std::uint64_t i = 1; i <= snapshot.committed_batches; ++i) {
    sqfv::Batch batch;
    sqpv::Receipt receipt;
    status = mapped(s->store.read(i, s->context, batch, receipt));
    if (status != AS::ok)
      return status;
    const auto &dsc = batch.descriptor();
    if (dsc.record_count != 1 ||
        dsc.source_binding != s->descriptor.source_binding ||
        !dsc.source_position.empty())
      return AS::corrupt;
    sqfv::Lease lease;
    status = flow(batch.acquire(s->descriptor.binding.access_scope, lease));
    if (status != AS::ok)
      return status;
    Entry entry;
    if (!decode(lease.payload(), entry))
      return AS::corrupt;
    auto previous =
        std::find_if(s->entries.begin(), s->entries.end(),
                     [&](const auto &e) { return e.id == entry.id; });
    if (entry.outcome == AttemptOutcome::reserved) {
      if (previous != s->entries.end() || s->entries.size() >= b.max_attempts ||
          entry.cost > b.ceiling_nano_usd - s->charged)
        return AS::corrupt;
      unsigned count = 0;
      for (const auto &e : s->entries)
        if (e.request == entry.request)
          ++count;
      if (entry.ordinal != count + 1 ||
          entry.request !=
              "sqdh1-sha256-" + knowledge::engine::sha256_hex(
                                    std::string(historical_endpoint) + "?" +
                                    entry.parameters))
        return AS::corrupt;
      s->charged += entry.cost;
      s->entries.push_back(std::move(entry));
    } else {
      if (previous == s->entries.end() ||
          previous->outcome != AttemptOutcome::reserved)
        return AS::corrupt;
      previous->outcome = entry.outcome;
    }
  }
  s->records = snapshot.committed_batches;
  // Pending records have no reissued execution capability after restart.
  for (auto &e : s->entries)
    if (e.outcome == AttemptOutcome::reserved)
      e.outcome = AttemptOutcome::indeterminate;
  out = std::move(s);
  return AS::ok;
}
} // namespace
AttemptTicket::AttemptTicket() noexcept = default;
AttemptTicket::~AttemptTicket() noexcept = default;
AttemptTicket::AttemptTicket(AttemptTicket &&) noexcept = default;
AttemptTicket &AttemptTicket::operator=(AttemptTicket &&) noexcept = default;
std::uint8_t AttemptTicket::ordinal() const noexcept {
  return state_ ? state_->ordinal : 0;
}
AS AttemptTicket::claim(const HistoricalPlan &plan,
                        std::uint64_t now) noexcept {
  if (!state_ || !plan)
    return AS::invalid_argument;
  auto owner = state_->owner.lock();
  if (!owner)
    return AS::closed;
  if (owner->pid != ::getpid())
    return AS::stale;
  try {
    std::lock_guard lock(owner->mutex);
    if (owner->uncertain)
      return AS::closed;
    auto &e = owner->entries[state_->index];
    if (e.request != plan.reference() || e.parameters != plan.parameters())
      return AS::conflict;
    if (e.claimed || e.outcome != AttemptOutcome::reserved)
      return AS::duplicate;
    if (now < e.admitted || now >= e.expires)
      return AS::stale;
    e.claimed = true;
    return AS::ok;
  } catch (...) {
    return AS::internal_error;
  }
}
AttemptLedger::AttemptLedger() noexcept = default;
AttemptLedger::~AttemptLedger() noexcept = default;
AttemptLedger::AttemptLedger(AttemptLedger &&) noexcept = default;
AttemptLedger &AttemptLedger::operator=(AttemptLedger &&) noexcept = default;
void AttemptLedger::reset() noexcept { state_.reset(); }
AS AttemptLedger::create(const std::string &root, const AttemptBudget &budget,
                         AttemptLedger &out) noexcept {
  try {
    AttemptLedger ready;
    auto s = initialize(true, root, budget, ready.state_);
    if (s == AS::ok)
      out = std::move(ready);
    return s;
  } catch (const std::bad_alloc &) {
    return AS::no_memory;
  } catch (...) {
    return AS::internal_error;
  }
}
AS AttemptLedger::open(const std::string &root, const AttemptBudget &budget,
                       AttemptLedger &out) noexcept {
  try {
    AttemptLedger ready;
    auto s = initialize(false, root, budget, ready.state_);
    if (s == AS::ok)
      out = std::move(ready);
    return s;
  } catch (const std::bad_alloc &) {
    return AS::no_memory;
  } catch (...) {
    return AS::internal_error;
  }
}
AS AttemptLedger::reserve(const HistoricalPlan &plan, std::string_view id,
                          const AttemptQuote &quote, std::uint64_t now,
                          AttemptTicket &out) noexcept {
  if (!state_)
    return AS::closed;
  if (state_->pid != ::getpid())
    return AS::stale;
  if (!plan || !token(id) || !token(quote.evidence_ref) ||
      !quote.quoted_unix_ms || quote.quoted_unix_ms > now ||
      now >= quote.expires_unix_ms ||
      quote.expires_unix_ms - quote.quoted_unix_ms > 86400000)
    return AS::invalid_argument;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (s.uncertain)
      return AS::closed;
    if (quote.ceiling_nano_usd > s.budget.ceiling_nano_usd - s.charged ||
        s.entries.size() >= s.budget.max_attempts)
      return AS::limit;
    unsigned count = 0;
    for (const auto &e : s.entries) {
      if (e.id == id)
        return AS::conflict;
      if (e.request == plan.reference())
        ++count;
    }
    if (count >= plan.limits()->max_attempts)
      return AS::limit;
    Entry e{std::string(id),
            std::string(plan.reference()),
            std::string(plan.parameters()),
            quote.evidence_ref,
            quote.ceiling_nano_usd,
            quote.quoted_unix_ms,
            quote.expires_unix_ms,
            now,
            static_cast<std::uint8_t>(count + 1),
            plan.limits()->max_attempts,
            AttemptOutcome::reserved,
            false};
    AttemptTicket ticket;
    ticket.state_ = std::make_unique<detail::AttemptTicketState>();
    ticket.state_->owner = state_;
    ticket.state_->index = s.entries.size();
    ticket.state_->ordinal = e.ordinal;
    auto status = s.append(e, false);
    if (status != AS::ok)
      return status;
    s.charged += e.cost;
    s.entries.push_back(std::move(e));
    out = std::move(ticket);
    return AS::ok;
  } catch (const std::bad_alloc &) {
    return AS::no_memory;
  } catch (...) {
    return AS::internal_error;
  }
}
AS AttemptLedger::finish(const AttemptTicket &ticket,
                         AttemptOutcome outcome) noexcept {
  if (!state_ || !ticket.state_)
    return AS::closed;
  if (state_->pid != ::getpid())
    return AS::stale;
  if (!terminal(outcome))
    return AS::invalid_argument;
  try {
    auto owner = ticket.state_->owner.lock();
    if (owner.get() != state_.get())
      return AS::conflict;
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (s.uncertain)
      return AS::closed;
    auto &e = s.entries[ticket.state_->index];
    if (e.outcome != AttemptOutcome::reserved)
      return e.outcome == outcome ? AS::duplicate : AS::conflict;
    if (!e.claimed && outcome == AttemptOutcome::completed)
      return AS::invalid_argument;
    auto event = e;
    event.outcome = outcome;
    auto status = s.append(event, true);
    if (status != AS::ok)
      return status;
    e.outcome = outcome;
    return AS::ok;
  } catch (const std::bad_alloc &) {
    return AS::no_memory;
  } catch (...) {
    return AS::internal_error;
  }
}
AS AttemptLedger::snapshot(AttemptSnapshot &out) const noexcept {
  if (!state_)
    return AS::closed;
  if (state_->pid != ::getpid())
    return AS::stale;
  try {
    std::lock_guard lock(state_->mutex);
    if (state_->uncertain)
      return AS::closed;
    AttemptSnapshot result{
        state_->charged, state_->budget.ceiling_nano_usd - state_->charged,
        static_cast<std::uint32_t>(state_->entries.size()), 0};
    for (const auto &e : state_->entries)
      if (e.outcome == AttemptOutcome::reserved ||
          e.outcome == AttemptOutcome::indeterminate)
        ++result.unresolved;
    out = result;
    return AS::ok;
  } catch (...) {
    return AS::internal_error;
  }
}
AS AttemptLedger::lookup(std::string_view id,
                         AttemptOutcome &out) const noexcept {
  if (!state_)
    return AS::closed;
  if (state_->pid != ::getpid())
    return AS::stale;
  try {
    std::lock_guard lock(state_->mutex);
    if (state_->uncertain)
      return AS::closed;
    for (const auto &e : state_->entries)
      if (e.id == id) {
        out = e.outcome;
        return AS::ok;
      }
    return AS::missing;
  } catch (...) {
    return AS::internal_error;
  }
}
} // namespace symphony::sqav::databento
