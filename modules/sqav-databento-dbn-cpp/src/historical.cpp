#include <algorithm>
#include <limits>
#include <new>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/sqav/databento/historical.hpp>
namespace symphony::sqav::databento {
namespace detail {
struct HistoricalPlanState {
  HistoricalSelection selection;
  HistoricalLimits limits;
  std::string parameters, reference;
};
struct HistoricalResponseState {
  HistoricalPlan plan;
  std::vector<std::uint8_t> bytes;
  HistoricalReport report;
  bool overflow = false, finished = false;
};
} // namespace detail
namespace {
constexpr std::uint64_t day_ns = 86'400'000'000'000ULL;
bool valid(const HistoricalLimits &l) noexcept {
  return l.dbn.max_file_bytes >= 128 && l.dbn.max_file_bytes <= (64ULL << 20) &&
         l.dbn.max_metadata_bytes >= 128 &&
         l.dbn.max_metadata_bytes <= (1U << 20) &&
         l.dbn.max_metadata_bytes <= l.dbn.max_file_bytes &&
         l.dbn.max_records > 0 && l.dbn.max_records <= (1ULL << 20) &&
         l.max_window_ns > 0 && l.max_window_ns <= day_ns &&
         l.max_symbols > 0 && l.max_symbols <= 128 && l.max_attempts > 0 &&
         l.max_attempts <= 8 && l.max_retry_after_seconds > 0 &&
         l.max_retry_after_seconds <= 86400;
}
bool alnum(char c) noexcept {
  return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
         (c >= '0' && c <= '9');
}
std::string escape(std::string_view s) {
  constexpr char hex[] = "0123456789ABCDEF";
  std::string result;
  for (unsigned char c : s) {
    if (alnum(static_cast<char>(c)) || c == '-' || c == '_' || c == '.' ||
        c == '~')
      result.push_back(static_cast<char>(c));
    else {
      result.push_back('%');
      result.push_back(hex[c >> 4]);
      result.push_back(hex[c & 15]);
    }
  }
  return result;
}
std::string reference(std::string_view value) {
  const auto hash = knowledge::engine::sha256_hex(ByteView(
      reinterpret_cast<const std::uint8_t *>(value.data()), value.size()));
  if (hash.size() != 64)
    throw std::bad_alloc{};
  return "sqdh1-sha256-" + hash;
}
Status bind(const HistoricalPlan &plan, const FileView &file,
            HistoricalReport &r) noexcept {
  const auto &s = *plan.selection();
  const auto &m = file.metadata();
  if (m.dataset != s.dataset || m.start != s.start || m.end != s.end ||
      m.limit != s.record_limit || m.stype_in != 1 || m.stype_out != 0 ||
      m.ts_out || m.symbols != s.symbols.size() ||
      m.record_count > s.record_limit)
    return Status::binding_mismatch;
  // The local profile has <=128 unique symbols. Reject substituted or repeated
  // response symbols, while admitting harmless provider ordering differences.
  for (const auto &expected : s.symbols) {
    unsigned count = 0;
    for (std::uint32_t i = 0; i < m.symbols; ++i) {
      std::string_view symbol;
      if (file.symbol(i, symbol) != Status::ok)
        return Status::internal_error;
      count += symbol == expected;
    }
    if (count != 1)
      return Status::binding_mismatch;
  }
  for (std::uint64_t i = 0; i < m.record_count; ++i) {
    Mbo record;
    if (file.record(i, record) != Status::ok)
      return Status::internal_error;
    if (record.ts_recv < s.start || record.ts_recv >= s.end ||
        (r.last_ts_recv && record.ts_recv < *r.last_ts_recv))
      return Status::binding_mismatch;
    if (!r.first_ts_recv)
      r.first_ts_recv = record.ts_recv;
    r.last_ts_recv = record.ts_recv;
  }
  r.records = m.record_count;
  r.record_limit_reached = m.record_count == s.record_limit;
  r.unresolved_symbols = m.partial != 0 || m.not_found != 0;
  return Status::ok;
}
bool retryable(std::uint16_t status) noexcept {
  return status == 429 || status == 502 || status == 503 || status == 504;
}
} // namespace
Status HistoricalPlan::create(const HistoricalSelection &s,
                              const HistoricalLimits &l,
                              HistoricalPlan &out) noexcept {
  if (!valid(l) || s.dataset.empty() || s.dataset.size() > 15 ||
      s.start >= s.end ||
      s.end > static_cast<std::uint64_t>(
                  std::numeric_limits<std::int64_t>::max()) ||
      s.record_limit == 0 || s.symbols.empty())
    return Status::invalid_argument;
  if (s.end - s.start > l.max_window_ns || s.record_limit > l.dbn.max_records ||
      s.symbols.size() > l.max_symbols)
    return Status::limit;
  for (char c : s.dataset)
    if (!((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.'))
      return Status::invalid_argument;
  for (const auto &symbol : s.symbols) {
    if (symbol.empty() || symbol == "ALL_SYMBOLS")
      return Status::invalid_argument;
    if (symbol.size() > 70)
      return Status::limit;
    for (unsigned char c : symbol)
      if (c < 32 || c > 126 || c == ',')
        return Status::invalid_argument;
  }
  try {
    auto state = std::make_shared<detail::HistoricalPlanState>();
    state->selection = s;
    state->limits = l;
    auto &symbols = state->selection.symbols;
    std::sort(symbols.begin(), symbols.end());
    if (std::adjacent_find(symbols.begin(), symbols.end()) != symbols.end())
      return Status::invalid_argument;
    std::string joined;
    for (const auto &symbol : symbols) {
      if (!joined.empty())
        joined += ',';
      joined += symbol;
    }
    state->parameters =
        "dataset=" + escape(s.dataset) + "&symbols=" + escape(joined) +
        "&schema=mbo&stype_in=raw_symbol&stype_out=instrument_id&start=" +
        std::to_string(s.start) + "&end=" + std::to_string(s.end) +
        "&limit=" + std::to_string(s.record_limit) +
        "&encoding=dbn&compression=none";
    state->reference = databento::reference(std::string(historical_endpoint) +
                                            "?" + state->parameters);
    HistoricalPlan ready;
    ready.state_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
std::string_view HistoricalPlan::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
std::string_view HistoricalPlan::parameters() const noexcept {
  return state_ ? state_->parameters : std::string_view{};
}
const HistoricalSelection *HistoricalPlan::selection() const noexcept {
  return state_ ? &state_->selection : nullptr;
}
const HistoricalLimits *HistoricalPlan::limits() const noexcept {
  return state_ ? &state_->limits : nullptr;
}
Status HistoricalPlan::split(HistoricalPlan &left,
                             HistoricalPlan &right) const noexcept {
  if (!state_ || &left == &right)
    return Status::invalid_argument;
  const auto &s = state_->selection;
  if (s.end - s.start < 2)
    return Status::limit;
  try {
    auto a = s, b = s;
    a.end = b.start = s.start + (s.end - s.start) / 2;
    HistoricalPlan x, y;
    auto status = create(a, state_->limits, x);
    if (status != Status::ok)
      return status;
    status = create(b, state_->limits, y);
    if (status != Status::ok)
      return status;
    left = std::move(x);
    right = std::move(y);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
HistoricalResponse::HistoricalResponse() noexcept = default;
HistoricalResponse::~HistoricalResponse() = default;
HistoricalResponse::HistoricalResponse(HistoricalResponse &&) noexcept =
    default;
HistoricalResponse &
HistoricalResponse::operator=(HistoricalResponse &&) noexcept = default;
Status HistoricalResponse::begin(const HistoricalPlan &plan,
                                 std::uint8_t attempt,
                                 std::uint16_t http_status,
                                 std::optional<std::uint32_t> retry_after,
                                 HistoricalResponse &out) noexcept {
  if (!plan || attempt == 0 || attempt > plan.limits()->max_attempts ||
      (http_status != 0 && (http_status < 200 || http_status > 599)))
    return Status::invalid_argument;
  try {
    auto state = std::make_unique<detail::HistoricalResponseState>();
    state->plan = plan;
    state->report.request = plan;
    state->report.attempt = attempt;
    state->report.http_status = http_status;
    state->report.retry_after_seconds = retry_after;
    HistoricalResponse ready;
    ready.state_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status HistoricalResponse::append(ByteView bytes) noexcept {
  if (!state_ || state_->finished)
    return Status::invalid_argument;
  auto &s = *state_;
  if (s.overflow)
    return Status::limit;
  const auto cap = s.plan.limits()->dbn.max_file_bytes;
  if (bytes.size() > cap - s.report.accepted_bytes) {
    s.overflow = true;
    return Status::limit;
  }
  if (!bytes.empty() && !s.bytes.empty()) {
    const auto a = reinterpret_cast<std::uintptr_t>(bytes.data());
    const auto b = reinterpret_cast<std::uintptr_t>(s.bytes.data());
    if (a <= b ? b - a < bytes.size() : a - b < s.bytes.size())
      return Status::invalid_argument;
  }
  try {
    if (s.report.http_status == 200 && !bytes.empty()) {
      const auto needed = s.bytes.size() + bytes.size();
      if (needed > s.bytes.capacity())
        s.bytes.reserve(static_cast<std::size_t>(std::min(
            cap, std::max<std::uint64_t>(needed, s.bytes.capacity() * 2ULL))));
      s.bytes.insert(s.bytes.end(), bytes.begin(), bytes.end());
    }
    s.report.accepted_bytes += bytes.size();
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status HistoricalResponse::finish(TransportEnd end,
                                  HistoricalReport &out) noexcept {
  if (!state_ || state_->finished ||
      (end != TransportEnd::complete && end != TransportEnd::interrupted &&
       end != TransportEnd::cancelled))
    return Status::invalid_argument;
  auto &s = *state_;
  auto r = s.report;
  r.transport = end;
  if (s.overflow)
    r.outcome = HistoricalOutcome::byte_limit;
  else if (end == TransportEnd::cancelled) {
    r.outcome = HistoricalOutcome::cancelled;
    r.recovery = Recovery::none;
  } else if (end == TransportEnd::interrupted) {
    r.outcome = HistoricalOutcome::interrupted;
    // A known permanent HTTP error is not retried just because its body broke.
    if (r.http_status == 0 || r.http_status == 200 || retryable(r.http_status))
      r.recovery = Recovery::repeat_window;
  } else if (r.http_status != 200) {
    r.outcome = HistoricalOutcome::http_error;
    if (retryable(r.http_status))
      r.recovery = Recovery::repeat_window;
  } else {
    FileView file;
    r.dbn_status = FileView::inspect(s.bytes, s.plan.limits()->dbn, file);
    if (r.dbn_status != Status::ok)
      r.outcome = HistoricalOutcome::invalid_dbn;
    else {
      r.dbn_status = bind(s.plan, file, r);
      if (r.dbn_status != Status::ok) {
        r.outcome = HistoricalOutcome::binding_mismatch;
        r.first_ts_recv.reset();
        r.last_ts_recv.reset();
      } else {
        const bool partial = r.record_limit_reached || r.unresolved_symbols;
        r.outcome = partial ? HistoricalOutcome::partial_window
                            : HistoricalOutcome::complete_window;
        r.coverage = partial ? Coverage::partial : Coverage::complete;
        r.recovery = !partial               ? Recovery::none
                     : r.unresolved_symbols ? Recovery::review
                     : s.plan.selection()->end - s.plan.selection()->start > 1
                         ? Recovery::split_window
                         : Recovery::review;
      }
    }
  }
  if (r.recovery == Recovery::repeat_window &&
      (r.attempt >= s.plan.limits()->max_attempts ||
       (r.retry_after_seconds &&
        *r.retry_after_seconds > s.plan.limits()->max_retry_after_seconds)))
    r.recovery = Recovery::review;
  s.report = r;
  s.finished = true;
  out = r;
  return Status::ok;
}
ByteView HistoricalResponse::body() const noexcept {
  return state_ ? ByteView(state_->bytes) : ByteView{};
}
Status HistoricalResponse::capture(const HistoricalAttribution &a,
                                   const sqav::Limits &limits,
                                   Capture &out) const noexcept {
  if (!state_ || !state_->finished)
    return Status::invalid_argument;
  const auto &s = *state_;
  const auto &r = s.report;
  if (r.outcome != HistoricalOutcome::complete_window &&
      r.outcome != HistoricalOutcome::partial_window)
    return Status::binding_mismatch;
  if (limits.max_capture_bytes == 0 ||
      limits.max_capture_bytes > (64ULL << 20) ||
      limits.max_metadata_bytes == 0 || limits.max_metadata_bytes > 65536 ||
      limits.max_field_bytes == 0 || limits.max_field_bytes > 4096 ||
      a.acquisition.role != TimeRole::acquisition)
    return Status::invalid_argument;
  for (const auto *value :
       {&a.attempt_id, &a.observer_ref, &a.dataset_revision, &a.access_scope,
        &a.acquisition.value, &a.acquisition.format_ref,
        &a.acquisition.clock_ref, &a.acquisition.precision_ref,
        &a.acquisition.evidence_ref})
    if (value->size() > limits.max_field_bytes)
      return Status::limit;
  try {
    const auto &selection = *s.plan.selection();
    // Persist the actual request and bounded outcome facts with the capture;
    // a digest alone would leave coverage evidence unavailable after replay.
    const std::string evidence =
        "sqdh-response-v1;request=" + std::string(s.plan.reference()) +
        ";http=200;transport=complete;attempt=" + std::to_string(r.attempt) +
        ";bytes=" + std::to_string(r.accepted_bytes) +
        ";records=" + std::to_string(r.records) +
        ";capped=" + (r.record_limit_reached ? "1" : "0") +
        ";unresolved=" + (r.unresolved_symbols ? "1" : "0");
    Description d;
    d.source = {"databento",
                "https://hist.databento.com",
                "v0",
                "timeseries.get_range",
                std::string(adapter_id),
                std::string(adapter_version),
                selection.dataset,
                a.dataset_revision,
                std::string(s.plan.reference()),
                std::string(native_schema),
                std::string(encoding_for_version(s.bytes[3])),
                a.access_scope};
    d.attempt_id = a.attempt_id;
    d.attribution_ref = a.observer_ref;
    d.source_position = evidence;
    d.coverage = r.coverage;
    d.coverage_scope = std::string(s.plan.parameters());
    d.coverage_evidence_ref = databento::reference(evidence);
    d.times.push_back(a.acquisition);
    return capture_file(d, s.bytes, s.plan.limits()->dbn, limits, out);
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::sqav::databento
