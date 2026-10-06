#include "wide_rational.hpp"
#include <map>
#include <optional>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
struct Valuation {
  std::int64_t mark, unit;
  std::uint64_t time;
  w::R multiplier, activation, filled_order, per_unit, basis;
};
Valuation valuation(const Json &v) {
  keys(v, {"id", "version", "mark_price_nanos", "mark_available_ns",
           "mark_evidence", "price_unit_nanos", "value_per_price_unit",
           "activation_cost", "filled_order_cost", "per_filled_unit_cost",
           "return_basis", "pnl_unit"});
  need(v.at("id") == "filled_quantity_markout" && v.at("version") == "1",
       "unsupported allocation valuation");
  Valuation x{i64(v.at("mark_price_nanos")),
              i64(v.at("price_unit_nanos")),
              u64(v.at("mark_available_ns")),
              w::parameter(v.at("value_per_price_unit")),
              w::parameter(v.at("activation_cost")),
              w::parameter(v.at("filled_order_cost")),
              w::parameter(v.at("per_filled_unit_cost")),
              w::parameter(v.at("return_basis"))};
  need(x.mark != INT64_MAX && x.unit > 0 && x.time != UINT64_MAX &&
           x.basis.n != 0,
       "valuation price, time, unit or basis invalid");
  need(!str(v.at("pnl_unit")).empty() && str(v.at("pnl_unit")).size() <= 128 &&
           !str(v.at("mark_evidence")).empty() &&
           str(v.at("mark_evidence")).size() <= 4096,
       "bounded unit and mark evidence required");
  return x;
}
struct Allocation {
  std::uint64_t q, requested, time;
  w::I notional;
  bool buy;
  std::optional<w::R> activation;
};
Allocation allocation(const Json &row, const std::string &mode) {
  need(row.at("protocol") == "symphony.sbv.liquidity-outcome.v1" &&
           row.at("model_id") == "displayed_depth_sweep" &&
           row.at("model_version") == "1",
       "typed liquidity outcome required");
  const auto &o = row.at("order");
  need(row.at("order_id") == o.at("order_id") &&
           row.at("signal_id") == o.at("signal_id") &&
           row.at("frame_source_ordinal") == o.at("frame_source_ordinal"),
       "allocation/order identity mismatch");
  need(o.at("side") == "buy" || o.at("side") == "sell",
       "invalid allocation side");
  Allocation a{0, u64(o.at("quantity")), 0,
               0, o.at("side") == "buy", std::nullopt};
  need(a.requested > 0 && a.requested <= 1000000000,
       "allocation quantity bound");
  const auto &p = o.at("activation_probability");
  if (p.at("status") == "supplied") {
    keys(p, {"status", "value"});
    a.activation = w::parameter(p.at("value"));
    need(a.activation->n >= 0 && a.activation->n <= a.activation->d,
         "invalid activation probability");
  } else {
    keys(p, {"status", "reason"});
    need(p.at("status") == "unavailable" && !str(p.at("reason")).empty(),
         "activation declaration required");
  }
  need(row.at("conditioning") == (mode == "shared_snapshot"
                                      ? "all_orders_active_in_input_order"
                                      : "selected_order_active"),
       "source conditioning mismatch");
  if (mode == "shared_snapshot")
    need(!a.activation || w::equal(*a.activation, {1, 1}),
         "shared stochastic branching unavailable");
  need(row.at("status") == "available" || row.at("status") == "unavailable",
       "allocation status invalid");
  if (row.at("status") == "unavailable") {
    need(row.at("data").is_null() && !str(row.at("reason")).empty(),
         "invalid unavailable allocation");
    return a;
  }
  const auto &d = row.at("data");
  a.q = u64(d.at("filled_quantity"));
  a.time = u64(d.at("frame_available_ns"));
  need(a.q <= a.requested && u64(d.at("requested_quantity")) == a.requested &&
           u64(d.at("unfilled_quantity")) == a.requested - a.q &&
           a.time != UINT64_MAX,
       "inconsistent allocation quantities/time");
  const auto &fills = d.at("fills");
  need(fills.is_array() && fills.size() <= 64, "allocation fill bound");
  std::uint64_t total = 0;
  for (const auto &f : fills) {
    auto q = u64(f.at("quantity"));
    auto price = i64(f.at("price_nanos"));
    need(price != INT64_MAX && q > 0 && q <= a.q - total,
         "invalid allocated fill");
    total += q;
    a.notional = w::add(a.notional, w::mul(price, q));
  }
  need(total == a.q &&
           w::integer(d.at("notional_price_nanos_times_quantity")) ==
               a.notional,
       "fill/notional mismatch");
  need(o.at("time_in_force") == "IOC" || o.at("time_in_force") == "FOK",
       "unsupported allocation time in force");
  need(o.at("time_in_force") != "FOK" || a.q == 0 || a.q == a.requested,
       "partial FOK allocation invalid");
  const auto disposition = a.q == a.requested               ? "full_fill"
                           : a.q                            ? "partial_fill"
                           : o.at("time_in_force") == "FOK" ? "fok_canceled"
                                                            : "no_fill";
  need(d.at("disposition") == disposition, "allocation disposition mismatch");
  const auto &vwap = d.at("vwap_price_nanos");
  if (a.q) {
    need(vwap.at("status") == "available", "filled allocation lacks VWAP");
    const auto &r = vwap.at("value");
    need(w::equal(
             {w::integer(r.at("numerator")), w::integer(r.at("denominator"))},
             {a.notional, static_cast<w::I>(a.q)}),
         "allocation VWAP mismatch");
  } else
    need(vwap.at("status") == "unavailable" && vwap.at("value").is_null(),
         "zero fill has VWAP");
  return a;
}
} // namespace
Json allocation_economics(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "path", "expected_sha256", "output_path", "selections",
           "studies", "on_unavailable", "extensions"});
  need(p.at("extensions").is_object(), "extensions object required");
  need(p.at("on_unavailable") == "unavailable" ||
           p.at("on_unavailable") == "reject",
       "unavailable policy invalid");
  std::set<std::string> studies;
  need(p.at("studies").is_array() && p.at("studies").size() <= 2,
       "study bound");
  for (const auto &v : p.at("studies")) {
    auto id = str(v);
    need((id == "allocation_costs" || id == "activation_moments") &&
             studies.insert(id).second,
         "unknown/duplicate allocation study");
  }
  auto source = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(source);
  need(!str(p.at("expected_sha256")).empty() &&
           source.at("content_sha256") == p.at("expected_sha256"),
       "source result identity mismatch");
  const auto &s = source.at("sections"),
             &provenance = s.at("provenance").at("data"),
             &choices = s.at("choices").at("data");
  need(provenance.at("model_id") == "displayed_depth_sweep" &&
           provenance.at("model_version") == "1" &&
           choices.at("protocol") == "symphony.sbv.liquidity-input.v1",
       "admitted liquidity result required");
  auto mode = str(choices.at("liquidity_mode"));
  need(mode == "independent" || mode == "shared_snapshot",
       "source liquidity mode invalid");
  const auto &rows = s.at("execution").at("data"),
             &orders = choices.at("orders");
  need(rows.is_array() && rows.size() <= 4096 && orders.is_array() &&
           rows.size() == orders.size(),
       "source allocation bound/choice mismatch");
  std::map<std::string, std::pair<Json, Allocation>> allocations;
  for (std::size_t i = 0; i < rows.size(); ++i) {
    deadline(end);
    const auto &row = rows[i];
    need(row.at("order") == orders[i], "source order choice mismatch");
    auto ordinal = u64(row.at("frame_source_ordinal"));
    need(mode != "shared_snapshot" || i == 0 ||
             ordinal == u64(rows[0].at("frame_source_ordinal")),
         "shared source requires one snapshot");
    auto a = allocation(row, mode);
    need(allocations.emplace(str(row.at("order_id")), std::pair{row, a}).second,
         "duplicate source order");
  }
  need(p.at("selections").is_array() && p.at("selections").size() <= 4096,
       "selection bound");
  Json output = Json::array(), costs = Json::array(), moments = Json::array();
  std::set<std::string> selected;
  std::uint64_t available = 0, partial = 0;
  for (const auto &choice : p.at("selections")) {
    deadline(end);
    keys(choice, {"order_id", "valuation", "mode", "inactive_pnl"});
    auto id = str(choice.at("order_id"));
    need(allocations.contains(id) && selected.insert(id).second,
         "unknown/duplicate selected order");
    auto v = valuation(choice.at("valuation"));
    bool mixture = choice.at("mode") == "activation_mixture";
    need(mixture || choice.at("mode") == "conditional_only",
         "allocation economics mode invalid");
    w::R inactive{};
    if (mixture)
      inactive = w::parameter(choice.at("inactive_pnl"));
    else
      need(choice.at("inactive_pnl").is_null(),
           "inactive P&L belongs to selected mixture");
    const auto &[row, a] = allocations.at(id);
    Json out{{"protocol", "symphony.sbv.allocation-economic-outcome.v1"},
             {"order_id", id},
             {"signal_id", row.at("signal_id")},
             {"conditioning", row.at("conditioning")},
             {"valuation", choice.at("valuation")},
             {"status", "available"},
             {"reason", ""},
             {"conditional", missing("source allocation unavailable")},
             {"mixture",
              {{"status", "not_selected"},
               {"reason", "conditional-only selection"},
               {"value", nullptr}}}};
    Json cost = missing("source allocation unavailable"),
         moment = missing("activation mixture not selected");
    if (row.at("status") == "unavailable") {
      out["status"] = "unavailable";
      out["reason"] = row.at("reason");
      if (mixture) {
        out["mixture"] = missing(str(row.at("reason")));
        moment = out["mixture"];
      }
    } else {
      need(v.time >= a.time, "valuation mark precedes execution frame");
      auto gross =
          w::times(w::reduce({w::mul(a.buy ? 1 : -1,
                                     w::add(w::mul(v.mark, a.q), -a.notional)),
                              v.unit}),
                   v.multiplier);
      auto unit_cost = w::times(v.per_unit, {static_cast<w::I>(a.q), 1});
      auto fill_cost = a.q ? v.filled_order : w::R{};
      auto total_cost = w::plus(v.activation, w::plus(fill_cost, unit_cost));
      auto net = w::plus(gross, {-total_cost.n, total_cost.d});
      auto ret = w::divide(net, v.basis);
      Json breakdown{{"activation", w::wire(v.activation)},
                     {"filled_order", w::wire(fill_cost)},
                     {"filled_units", w::wire(unit_cost)},
                     {"total", w::wire(total_cost)}};
      out["conditional"] = {{"status", "available"},
                            {"value",
                             {{"filled_quantity", dec(a.q)},
                              {"unfilled_quantity", dec(a.requested - a.q)},
                              {"gross_markout", w::wire(gross)},
                              {"costs", breakdown},
                              {"net_markout", w::wire(net)},
                              {"return", w::wire(ret)}}}};
      cost = {{"status", "available"}, {"value", breakdown}};
      if (mixture) {
        if (!a.activation) {
          out["status"] = "partial";
          out["reason"] = "activation probability unavailable";
          out["mixture"] = missing("activation probability unavailable");
          moment = out["mixture"];
        } else {
          auto prob = *a.activation, other = w::plus({1, 1}, {-prob.n, prob.d}),
               inactive_return = w::divide(inactive, v.basis);
          out["mixture"] = {
              {"status", "scenario_assumption"},
              {"value", Json::array({{{"branch", "active"},
                                      {"weight", w::wire(prob)},
                                      {"net_markout", w::wire(net)},
                                      {"return", w::wire(ret)}},
                                     {{"branch", "inactive"},
                                      {"weight", w::wire(other)},
                                      {"net_markout", w::wire(inactive)},
                                      {"return", w::wire(inactive_return)}}})}};
          if (studies.contains("activation_moments")) {
            auto mean = w::plus(w::times(prob, net), w::times(other, inactive));
            auto delta = w::plus(net, {-inactive.n, inactive.d});
            // Endpoint mass has exactly zero variance without squaring an
            // unused tail.
            auto variance =
                (prob.n == 0 || other.n == 0)
                    ? w::R{}
                    : w::times(w::times(prob, other), w::times(delta, delta));
            moment = {{"status", "scenario_assumption"},
                      {"value",
                       {{"expected_net_markout", w::wire(mean)},
                        {"net_markout_variance", w::wire(variance)},
                        {"expected_return", w::wire(w::divide(mean, v.basis))},
                        {"return_variance",
                         w::wire(w::divide(w::divide(variance, v.basis),
                                           v.basis))}}}};
          }
        }
      }
    }
    if (out.at("status") == "available")
      ++available;
    else {
      need(p.at("on_unavailable") != "reject", str(out.at("reason")).c_str());
      if (out.at("status") == "partial")
        ++partial;
    }
    if (studies.contains("allocation_costs"))
      costs.push_back({{"order_id", id},
                       {"pnl_unit", choice.at("valuation").at("pnl_unit")},
                       {"result", cost}});
    if (studies.contains("activation_moments"))
      moments.push_back({{"order_id", id},
                         {"pnl_unit", choice.at("valuation").at("pnl_unit")},
                         {"result", moment}});
    output.push_back(std::move(out));
  }
  auto result = base("quantity-aware allocation economics");
  result["status"] = "partial";
  auto &sections = result["sections"];
  sections["economics"] =
      section(output, available == output.size() ? "available" : "partial",
              available == output.size()
                  ? ""
                  : "one or more allocations or mixtures unavailable");
  sections["summary"] =
      section({{"selected_orders", dec(output.size())},
               {"available", dec(available)},
               {"partial", dec(partial)},
               {"unavailable", dec(output.size() - available - partial)},
               {"aggregation",
                "per order; no portfolio or joint distribution inferred"}});
  Json computed = Json::array();
  if (studies.contains("allocation_costs"))
    computed.push_back({{"study_id", "allocation_costs"},
                        {"version", "1"},
                        {"results", costs}});
  if (studies.contains("activation_moments"))
    computed.push_back({{"study_id", "activation_moments"},
                        {"version", "1"},
                        {"results", moments}});
  sections["studies"] =
      section(computed, studies.empty() ? "not_selected" : "available",
              studies.empty() ? "zero studies selected" : "");
  for (const auto *name :
       {"signals", "execution", "replay", "book_frames", "book_checkpoint"})
    sections[name] = s.at(name);
  sections["source_context"] =
      section({{"choices", s.at("choices")},
               {"provenance", s.at("provenance")},
               {"summary", s.at("summary")},
               {"parent_context", s.at("source_context")}});
  auto c = p;
  c.erase("output_path");
  sections["choices"] = section(c);
  sections["resources"] = section(
      {{"backend", "cpu"},
       {"actual_workers", output.empty() ? "0" : "1"},
       {"ordering", "stable selection order; immutable source allocations"}});
  sections["provenance"] =
      section({{"engine_version", version},
               {"source_result_sha256", source.at("content_sha256")},
               {"source_census_sha256", provenance.at("source_census_sha256")},
               {"transform_id", "filled_quantity_markout"},
               {"transform_version", "1"},
               {"numeric_profile", "checked signed 128-bit reduced rational; "
                                   "overflow rejects before persistence"},
               {"mark_evidence", "caller supplied; not independently verified"},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  sections["diagnostics"] = section(Json::array(
      {"Only filled units contribute to gross markout; activation costs also "
       "apply to active zero-fill orders.",
       "Negative costs are explicit rebates; mark, multiplier, P&L unit and "
       "nonzero signed return basis are caller conventions.",
       "Markout is a hypothetical valuation, not a liquidation fill, realized "
       "account P&L or calibrated forecast.",
       "Activation branches retain source conditioning, including shared "
       "all-active dependence; no cross-order portfolio distribution is "
       "inferred.",
       "Source hashes verify content identity, not publisher authenticity. "
       "Replay and complete execution allocations remain portable."}));
  sections["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "allocation-economics", end);
}
} // namespace symphony::sbv::detail
