#include <algorithm>
#include <array>
#include <charconv>
#include <limits>
#include <set>
#include <symphony/source/json.hpp>
#include <symphony/sqav/databento/reference.hpp>
#include <zstd.h>
namespace symphony::sqav::databento::reference {
namespace detail {
struct PlanState {
  Selection selection;
  source::HttpRequest request;
  std::string reference;
};
struct PageState {
  Plan plan;
  std::vector<std::uint8_t> compressed, decoded;
  std::uint32_t records = 0;
  std::string reference;
};
struct QuoteState {
  std::string request, reference;
  std::uint64_t nano = 0, quoted = 0, expires = 0;
};
} // namespace detail
namespace {
const source::HttpRequest empty_request{};
const Selection empty_selection{};
std::string operation(Operation o) {
  switch (o) {
  case Operation::corporate_actions:
    return "corporate_actions.get_range";
  case Operation::adjustment_factors:
    return "adjustment_factors.get_range";
  case Operation::security_master_range:
    return "security_master.get_range";
  case Operation::security_master_last:
    return "security_master.get_last";
  }
  return {};
}
bool valid(const Limits &l) {
  return source::valid_limits(l.record_json) && l.compressed_bytes > 0 &&
         l.compressed_bytes <= (64U << 20) && l.decompressed_bytes > 0 &&
         l.decompressed_bytes <= (64U << 20) && l.records > 0 &&
         l.records <= 65536 && l.window_log >= 10 && l.window_log <= 23;
}
Status decompress(source::Bytes bytes, const Limits &l,
                  std::vector<std::uint8_t> &out) {
  if (bytes.empty())
    return Status::malformed;
  if (bytes.size() > l.compressed_bytes)
    return Status::limit;
  if (ZSTD_versionNumber() != 10507)
    return Status::unsupported;
  auto ctx = std::unique_ptr<ZSTD_DCtx, decltype(&ZSTD_freeDCtx)>(
      ZSTD_createDCtx(), ZSTD_freeDCtx);
  if (!ctx)
    return Status::no_memory;
  if (ZSTD_isError(
          ZSTD_DCtx_setParameter(ctx.get(), ZSTD_d_windowLogMax, l.window_log)))
    return Status::internal_error;
  ZSTD_inBuffer in{bytes.data(), bytes.size(), 0};
  std::array<std::uint8_t, 65536> scratch{};
  std::size_t remaining = 1;
  while (in.pos < in.size || remaining != 0) {
    ZSTD_outBuffer dest{scratch.data(), scratch.size(), 0};
    auto before = in.pos;
    remaining = ZSTD_decompressStream(ctx.get(), &dest, &in);
    if (ZSTD_isError(remaining))
      return Status::malformed;
    if (dest.pos > l.decompressed_bytes - out.size())
      return Status::limit;
    out.insert(out.end(), scratch.begin(),
               scratch.begin() + static_cast<std::ptrdiff_t>(dest.pos));
    if (in.pos == in.size && remaining != 0 && dest.pos == 0)
      return Status::malformed;
    if (before == in.pos && dest.pos == 0)
      return Status::malformed;
  }
  return Status::ok;
}
bool identifier(const nlohmann::json &j, const char *field) {
  const auto &v = j.at(field);
  return (v.is_string() && !v.get_ref<const std::string &>().empty()) ||
         v.is_number_unsigned();
}
Status dollars(std::string_view text, std::uint64_t &out) {
  while (!text.empty() && (text.front() == ' ' || text.front() == '\n' ||
                           text.front() == '\r' || text.front() == '\t'))
    text.remove_prefix(1);
  while (!text.empty() && (text.back() == ' ' || text.back() == '\n' ||
                           text.back() == '\r' || text.back() == '\t'))
    text.remove_suffix(1);
  if (text.empty() || text.size() > 128 || text.front() == '-' ||
      text.front() == '+')
    return Status::malformed;
  std::string digits;
  std::size_t p = 0;
  int fractional = 0;
  if (text[p] == '0') {
    digits += '0';
    ++p;
    if (p < text.size() && text[p] >= '0' && text[p] <= '9')
      return Status::malformed;
  } else {
    while (p < text.size() && text[p] >= '0' && text[p] <= '9')
      digits += text[p++];
  }
  if (digits.empty())
    return Status::malformed;
  if (p < text.size() && text[p] == '.') {
    ++p;
    auto b = p;
    while (p < text.size() && text[p] >= '0' && text[p] <= '9') {
      digits += text[p++];
      ++fractional;
    }
    if (p == b)
      return Status::malformed;
  }
  int exponent = 0;
  if (p < text.size() && (text[p] == 'e' || text[p] == 'E')) {
    ++p;
    bool negative = false;
    if (p < text.size() && (text[p] == '+' || text[p] == '-')) {
      negative = text[p] == '-';
      ++p;
    }
    auto b = p;
    while (p < text.size() && text[p] >= '0' && text[p] <= '9') {
      if (exponent > 1000)
        return Status::limit;
      exponent = exponent * 10 + (text[p++] - '0');
    }
    if (p == b)
      return Status::malformed;
    if (negative)
      exponent = -exponent;
  }
  if (p != text.size())
    return Status::malformed;
  auto first = digits.find_first_not_of('0');
  if (first == digits.npos) {
    out = 0;
    return Status::ok;
  }
  digits.erase(0, first);
  const int scale = 9 + exponent - fractional;
  std::uint64_t value = 0;
  bool round = false;
  if (scale >= 0) {
    if (digits.size() + std::size_t(scale) > 20)
      return Status::limit;
    digits.append(std::size_t(scale), '0');
  } else {
    auto cut = std::size_t(-scale);
    if (cut >= digits.size()) {
      out = 1;
      return Status::ok;
    }
    auto keep = digits.size() - cut;
    round = digits.find_first_not_of('0', keep) != digits.npos;
    digits.resize(keep);
  }
  for (char c : digits) {
    unsigned d = unsigned(c - '0');
    if (value > (UINT64_MAX - d) / 10)
      return Status::limit;
    value = value * 10 + d;
  }
  if (round) {
    if (value == UINT64_MAX)
      return Status::limit;
    ++value;
  }
  out = value;
  return Status::ok;
}
} // namespace
Status Plan::create(const Selection &s, Plan &out) noexcept {
  if (static_cast<unsigned>(s.operation) > 3 || s.symbols.empty() ||
      s.symbols.size() > 128)
    return Status::invalid_argument;
  if (s.operation == Operation::security_master_last) {
    if (!s.start.empty() || !s.end.empty())
      return Status::invalid_argument;
  } else if (!source::date(s.start) || !source::date(s.end) || s.start >= s.end)
    return Status::invalid_argument;
  try {
    std::set<std::string> seen;
    std::string symbols;
    for (const auto &symbol : s.symbols) {
      if (!source::token(symbol, 128) || symbol == "ALL_SYMBOLS" ||
          !seen.insert(symbol).second)
        return Status::invalid_argument;
      if (!symbols.empty())
        symbols += ',';
      symbols += symbol;
    }
    auto p = std::make_shared<detail::PlanState>();
    p->selection = s;
    p->request.endpoint =
        "https://hist.databento.com/v0/" + operation(s.operation);
    p->request.post = true;
    p->request.credential = source::HttpRequest::Credential::basic_username;
    p->request.parameters =
        "symbols=" + source::encode(symbols) +
        "&stype_in=raw_symbol&allocate_isins=false&compression=zstd";
    if (s.operation != Operation::security_master_last)
      p->request.parameters += "&start=" + s.start + "&end=" + s.end;
    if (s.operation == Operation::corporate_actions)
      p->request.parameters += "&index=event_date";
    if (s.operation == Operation::security_master_range)
      p->request.parameters += "&index=ts_effective";
    p->reference = p->request.reference();
    out.state_ = std::move(p);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
const source::HttpRequest &Plan::request() const noexcept {
  return state_ ? state_->request : empty_request;
}
const Selection &Plan::selection() const noexcept {
  return state_ ? state_->selection : empty_selection;
}
std::string_view Plan::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Page::admit(const Plan &p, const source::HttpResponse &r,
                   const Limits &l, Page &out) noexcept {
  if (!p || !valid(l))
    return Status::invalid_argument;
  if (!r.complete)
    return Status::transport_error;
  if (r.http_status != 200)
    return Status::http_error;
  try {
    auto state = std::make_shared<detail::PageState>();
    auto s = decompress(r.body, l, state->decoded);
    if (s != Status::ok)
      return s;
    std::string_view text(reinterpret_cast<const char *>(state->decoded.data()),
                          state->decoded.size());
    while (!text.empty()) {
      auto end = text.find('\n');
      auto line = text.substr(0, end);
      if (line.ends_with('\r'))
        line.remove_suffix(1);
      if (line.empty())
        return Status::malformed;
      if (state->records == l.records)
        return Status::limit;
      nlohmann::json j;
      auto status = source::parse_json(
          {reinterpret_cast<const std::uint8_t *>(line.data()), line.size()},
          l.record_json, j);
      if (status != Status::ok)
        return status;
      if (!j.is_object() || !identifier(j, "security_id"))
        return Status::malformed;
      auto op = p.selection().operation;
      if (op == Operation::corporate_actions ||
          op == Operation::adjustment_factors) {
        if (!identifier(j, "event_id"))
          return Status::malformed;
      } else if (!identifier(j, "listing_id"))
        return Status::malformed;
      // Preserve all PIT records, nested fields and unknown additions exactly.
      ++state->records;
      if (end == text.npos)
        break;
      text.remove_prefix(end + 1);
    }
    state->plan = p;
    state->compressed = r.body;
    state->reference =
        "databento-reference-v1-" +
        source::digest(
            std::string(p.reference()) + "\n" +
            std::string(reinterpret_cast<const char *>(r.body.data()),
                        r.body.size()));
    out.state_ = std::move(state);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
source::Bytes Page::compressed_original() const noexcept {
  return state_ ? source::Bytes(state_->compressed) : source::Bytes{};
}
source::Bytes Page::jsonl() const noexcept {
  return state_ ? source::Bytes(state_->decoded) : source::Bytes{};
}
std::uint32_t Page::records() const noexcept {
  return state_ ? state_->records : 0;
}
std::string_view Page::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Page::capture(std::string_view attempt, std::string_view attribution,
                     std::string_view scope, const TimeEvidence &acq,
                     const sqav::Limits &limits, Capture &out) const noexcept {
  if (!state_ || acq.role != TimeRole::acquisition)
    return Status::invalid_argument;
  try {
    const auto &plan = state_->plan;
    auto op = operation(plan.selection().operation);
    Description d{{"databento", "reference-http-v0", "observed-2026-09-28", op,
                   "sqav-databento-reference-cpp", "0.1.0-dev", op,
                   std::string(plan.reference()), std::string(plan.reference()),
                   "databento-reference-jsonl", "utf8-jsonl+zstd-1.5.7",
                   std::string(scope)},
                  std::string(attempt),
                  std::string(attribution),
                  "allocate_isins=false",
                  Coverage::partial,
                  plan.request().parameters,
                  state_->reference,
                  state_->records,
                  {acq}};
    auto s = Capture::create(d, state_->compressed, limits, out);
    return s == sqav::Status::ok          ? Status::ok
           : s == sqav::Status::no_memory ? Status::no_memory
           : s == sqav::Status::limit     ? Status::limit
                                          : Status::invalid_argument;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status collect(const Plan &p, source::CredentialUse &credentials,
               const source::HttpLimits &h, const Limits &l,
               std::stop_token stop, Page &out) noexcept {
  if (!p || !valid(l) || h.max_body_bytes > l.compressed_bytes)
    return Status::invalid_argument;
  source::HttpResponse response;
  auto s = source::fetch(p.request(), &credentials, h, stop, response);
  return s == Status::ok ? Page::admit(p, response, l, out) : s;
}
Status CostQuote::request(const HistoricalPlan &p,
                          source::HttpRequest &out) noexcept {
  if (!p)
    return Status::invalid_argument;
  try {
    const auto &s = *p.selection();
    std::string symbols;
    for (const auto &v : s.symbols) {
      if (!symbols.empty())
        symbols += ',';
      symbols += v;
    }
    source::HttpRequest r{
        "https://hist.databento.com/v0/metadata.get_cost",
        "dataset=" + source::encode(s.dataset) +
            "&symbols=" + source::encode(symbols) +
            "&schema=mbo&stype_in=raw_symbol&start=" + std::to_string(s.start) +
            "&end=" + std::to_string(s.end) +
            "&limit=" + std::to_string(s.record_limit),
        true, source::HttpRequest::Credential::basic_username};
    out = std::move(r);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status CostQuote::admit(const HistoricalPlan &p, const source::HttpResponse &r,
                        std::uint64_t now, std::uint32_t validity,
                        CostQuote &out) noexcept {
  if (!p || now == 0 || validity == 0 || validity > 86400000 ||
      now > UINT64_MAX - validity)
    return Status::invalid_argument;
  if (!r.complete)
    return Status::transport_error;
  if (r.http_status != 200)
    return Status::http_error;
  if (r.body.size() > 128)
    return Status::limit;
  try {
    std::uint64_t nano = 0;
    auto status = dollars(
        {reinterpret_cast<const char *>(r.body.data()), r.body.size()}, nano);
    if (status != Status::ok)
      return status;
    auto s = std::make_shared<detail::QuoteState>();
    s->nano = nano;
    s->quoted = now;
    s->expires = now + validity;
    s->request = p.reference();
    s->reference =
        "db-cost-v1-" +
        source::digest(
            s->request + "\n" +
            std::string(reinterpret_cast<const char *>(r.body.data()),
                        r.body.size()) +
            "\n" + std::to_string(now));
    out.state_ = std::move(s);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status CostQuote::for_attempt(const HistoricalPlan &p, std::uint64_t now,
                              AttemptQuote &out) const noexcept {
  if (!state_ || !p)
    return Status::invalid_argument;
  if (state_->request != p.reference())
    return Status::binding_mismatch;
  if (now < state_->quoted || now >= state_->expires)
    return Status::stale;
  try {
    AttemptQuote q{state_->reference, state_->nano, state_->quoted,
                   state_->expires};
    out = std::move(q);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
std::uint64_t CostQuote::nano_usd() const noexcept {
  return state_ ? state_->nano : 0;
}
Status quote(const HistoricalPlan &p, source::CredentialUse &credentials,
             const source::HttpLimits &h, std::uint64_t now,
             std::uint32_t validity, std::stop_token stop,
             CostQuote &out) noexcept {
  if (now == 0 || validity == 0 || validity > 86400000 ||
      now > UINT64_MAX - validity || h.max_body_bytes > 128)
    return Status::invalid_argument;
  source::HttpRequest r;
  auto s = CostQuote::request(p, r);
  if (s != Status::ok)
    return s;
  source::HttpResponse response;
  s = source::fetch(r, &credentials, h, stop, response);
  return s == Status::ok ? CostQuote::admit(p, response, now, validity, out)
                         : s;
}
} // namespace symphony::sqav::databento::reference
