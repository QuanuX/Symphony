#include <algorithm>
#include <limits>
#include <set>
#include <symphony/source/json.hpp>
#include <symphony/sqav/fred.hpp>
namespace symphony::sqav::fred {
namespace detail {
struct PlanState {
  Selection selection;
  source::HttpRequest request;
  std::string reference;
};
struct PageState {
  Plan plan;
  std::vector<Observation> observations;
  std::vector<std::string> vintages;
  std::vector<std::uint8_t> bytes;
  std::string reference;
  std::uint32_t count = 0;
  std::optional<std::uint32_t> next;
};
} // namespace detail
namespace {
const Selection empty_selection{};
const source::HttpRequest empty_request{};
std::string str(const nlohmann::json &j, const char *name) {
  return j.at(name).get<std::string>();
}
std::uint32_t number(const nlohmann::json &j, const char *name) {
  const auto &v = j.at(name);
  if (!v.is_number_unsigned() ||
      v.get<std::uint64_t>() > std::numeric_limits<std::uint32_t>::max())
    throw Status::malformed;
  return v.get<std::uint32_t>();
}
} // namespace
Status Plan::create(const Selection &s, Plan &out) noexcept {
  if (static_cast<unsigned>(s.operation) > 1 || !source::token(s.series, 128) ||
      s.series.find(':') != s.series.npos || !source::date(s.realtime_start) ||
      !source::date(s.realtime_end) || s.realtime_start > s.realtime_end ||
      s.limit == 0 ||
      s.limit > (s.operation == Operation::observations ? 100000U : 10000U))
    return Status::invalid_argument;
  if (s.operation == Operation::observations &&
      (!source::date(s.observation_start) || !source::date(s.observation_end) ||
       s.observation_start > s.observation_end))
    return Status::invalid_argument;
  if (s.operation == Operation::vintage_dates &&
      (!s.observation_start.empty() || !s.observation_end.empty()))
    return Status::invalid_argument;
  try {
    auto state = std::make_shared<detail::PlanState>();
    state->selection = s;
    auto &r = state->request;
    r.endpoint =
        "https://api.stlouisfed.org/fred/series/" +
        std::string(s.operation == Operation::observations ? "observations"
                                                           : "vintagedates");
    r.parameters = "file_type=json&series_id=" + source::encode(s.series) +
                   "&realtime_start=" + s.realtime_start +
                   "&realtime_end=" + s.realtime_end +
                   "&sort_order=asc&limit=" + std::to_string(s.limit) +
                   "&offset=" + std::to_string(s.offset);
    if (s.operation == Operation::observations)
      r.parameters += "&observation_start=" + s.observation_start +
                      "&observation_end=" + s.observation_end +
                      "&units=lin&output_type=1";
    r.credential = source::HttpRequest::Credential::fred_query_key;
    state->reference = r.reference();
    out.state_ = std::move(state);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
const Selection &Plan::selection() const noexcept {
  return state_ ? state_->selection : empty_selection;
}
const source::HttpRequest &Plan::request() const noexcept {
  return state_ ? state_->request : empty_request;
}
std::string_view Plan::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Page::admit(const Plan &p, const source::HttpResponse &r,
                   const source::JsonLimits &l, Page &out) noexcept {
  if (!p)
    return Status::invalid_argument;
  if (!r.complete)
    return Status::transport_error;
  if (r.http_status != 200)
    return Status::http_error;
  nlohmann::json j;
  auto status = source::parse_json(r.body, l, j);
  if (status != Status::ok)
    return status;
  try {
    const auto &s = p.selection();
    if (str(j, "realtime_start") != s.realtime_start ||
        str(j, "realtime_end") != s.realtime_end ||
        str(j, "sort_order") != "asc" || number(j, "offset") != s.offset ||
        number(j, "limit") != s.limit)
      return Status::binding_mismatch;
    auto state = std::make_shared<detail::PageState>();
    state->plan = p;
    state->count = number(j, "count");
    if (s.offset > state->count)
      return Status::malformed;
    std::size_t records = 0;
    if (s.operation == Operation::observations) {
      if (str(j, "observation_start") != s.observation_start ||
          str(j, "observation_end") != s.observation_end ||
          str(j, "units") != "lin" || number(j, "output_type") != 1 ||
          str(j, "file_type") != "json" ||
          str(j, "order_by") != "observation_date")
        return Status::binding_mismatch;
      const auto &rows = j.at("observations");
      if (!rows.is_array() || rows.size() > s.limit)
        return Status::malformed;
      std::set<std::string> unique;
      std::string previous;
      for (const auto &row : rows) {
        Observation o{str(row, "date"), str(row, "realtime_start"),
                      str(row, "realtime_end"), str(row, "value"), false};
        if (!source::date(o.date) || !source::date(o.realtime_start) ||
            !source::date(o.realtime_end) || o.date < s.observation_start ||
            o.date > s.observation_end || o.realtime_start > o.realtime_end ||
            o.realtime_end < s.realtime_start ||
            o.realtime_start > s.realtime_end || o.date < previous)
          return Status::malformed;
        if (!unique
                 .insert(o.date + "/" + o.realtime_start + "/" + o.realtime_end)
                 .second)
          return Status::malformed;
        o.missing = o.value == ".";
        if (!o.missing && !source::decimal(o.value))
          return Status::malformed;
        previous = o.date;
        state->observations.push_back(std::move(o));
      }
      records = state->observations.size();
    } else {
      if (str(j, "order_by") != "vintage_date")
        return Status::binding_mismatch;
      const auto &rows = j.at("vintage_dates");
      if (!rows.is_array() || rows.size() > s.limit)
        return Status::malformed;
      std::string previous;
      for (const auto &row : rows) {
        auto v = row.get<std::string>();
        if (!source::date(v) || v < s.realtime_start || v > s.realtime_end ||
            (!previous.empty() && v <= previous))
          return Status::malformed;
        previous = v;
        state->vintages.push_back(std::move(v));
      }
      records = state->vintages.size();
    }
    // A finite page must account exactly for the requested remaining count.
    // This refuses silent skips and empty-page loops before publishing a next
    // offset.
    const auto expected = std::min(s.limit, state->count - s.offset);
    if (records != expected)
      return Status::malformed;
    const auto next = s.offset + static_cast<std::uint32_t>(records);
    if (next < state->count)
      state->next = next;
    state->bytes = r.body;
    state->reference =
        "fred-page-v1-" +
        source::digest(
            std::string(p.reference()) + "\n" +
            std::string(reinterpret_cast<const char *>(r.body.data()),
                        r.body.size()));
    out.state_ = std::move(state);
    return Status::ok;
  } catch (Status x) {
    return x;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
std::span<const Observation> Page::observations() const noexcept {
  return state_ ? std::span<const Observation>(state_->observations)
                : std::span<const Observation>{};
}
std::span<const std::string> Page::vintage_dates() const noexcept {
  return state_ ? std::span<const std::string>(state_->vintages)
                : std::span<const std::string>{};
}
source::Bytes Page::original() const noexcept {
  return state_ ? source::Bytes(state_->bytes) : source::Bytes{};
}
std::string_view Page::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
std::uint32_t Page::total_count() const noexcept {
  return state_ ? state_->count : 0;
}
std::optional<std::uint32_t> Page::next_offset() const noexcept {
  return state_ ? state_->next : std::nullopt;
}
Status Page::capture(std::string_view attempt, std::string_view attribution,
                     std::string_view scope, const TimeEvidence &acq,
                     const sqav::Limits &limits, Capture &out) const noexcept {
  if (!state_ || acq.role != TimeRole::acquisition)
    return Status::invalid_argument;
  try {
    const auto &s = state_->plan.selection();
    const auto operation = s.operation == Operation::observations
                               ? "series.observations"
                               : "series.vintagedates";
    Description d{{"stlouisfed:fred", "fred-api", "observed-2026-09-28",
                   operation, "sqav-fred-cpp", "0.1.0-dev", s.series,
                   s.realtime_start + "/" + s.realtime_end,
                   std::string(state_->plan.reference()),
                   "fred-json-" + std::string(operation), "utf8-json",
                   std::string(scope)},
                  std::string(attempt),
                  std::string(attribution),
                  "offset=" + std::to_string(s.offset),
                  s.offset == 0 && !state_->next ? Coverage::complete
                                                 : Coverage::partial,
                  state_->plan.request().parameters,
                  state_->reference,
                  state_->observations.size() + state_->vintages.size(),
                  {acq}};
    auto status = Capture::create(d, state_->bytes, limits, out);
    return status == sqav::Status::ok          ? Status::ok
           : status == sqav::Status::no_memory ? Status::no_memory
           : status == sqav::Status::limit     ? Status::limit
                                               : Status::invalid_argument;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status collect(const Plan &p, source::CredentialUse &credentials,
               const source::HttpLimits &http, const source::JsonLimits &json,
               std::stop_token stop, Page &out) noexcept {
  if (!p || !source::valid_limits(json) || http.max_body_bytes > json.bytes)
    return Status::invalid_argument;
  source::HttpResponse response;
  auto s = source::fetch(p.request(), &credentials, http, stop, response);
  if (s != Status::ok)
    return s;
  return Page::admit(p, response, json, out);
}
} // namespace symphony::sqav::fred
