#include "combinatorial.hpp"
#include "detail.hpp"
#include <algorithm>
#include <limits>

namespace symphony::sbv::detail {
namespace {
using U = std::uint64_t;
struct Interval {
  U start, end;
};
U add(U a, U b) {
  need(b <= std::numeric_limits<U>::max() - a, "split timestamp overflow");
  return a + b;
}
U sub(U a, U b) {
  need(b <= a, "split timestamp underflow");
  return a - b;
}
std::vector<Interval> merge(std::vector<Interval> v) {
  std::sort(v.begin(), v.end(), [](auto a, auto b) {
    return a.start < b.start || (a.start == b.start && a.end < b.end);
  });
  std::vector<Interval> out;
  for (auto x : v) {
    if (out.empty() || x.start > out.back().end)
      out.push_back(x);
    else
      out.back().end = std::max(out.back().end, x.end);
  }
  return out;
}
bool overlaps(const std::vector<Interval> &v, Interval x) {
  const auto it =
      std::lower_bound(v.begin(), v.end(), x.start,
                       [](auto a, U start) { return a.end < start; });
  return it != v.end() && it->start <= x.end;
}
Json intervals(const std::vector<Interval> &v) {
  Json out = Json::array();
  for (auto x : v)
    out.push_back({{"start_ns", dec(x.start)}, {"end_ns", dec(x.end)}});
  return out;
}
struct GroupedPlan {
  Json groups = Json::array(), folds = Json::array(), selection;
};
GroupedPlan grouped_plan(const Json &plan, std::int64_t end) {
  keys(plan, {"kind", "groups", "test_group_count", "fold_offset", "fold_count",
              "fit_cutoff_ns"});
  const auto &groups = plan.at("groups");
  need(groups.is_array() && groups.size() >= 2,
       "at least two time groups required");
  const auto k = u64(plan.at("test_group_count"));
  need(k > 0 && k < groups.size(),
       "test_group_count must leave training groups");
  std::set<std::string> ids;
  U previous_end = 0;
  for (const auto &g : groups) {
    deadline(end);
    keys(g, {"id", "start_ns", "end_ns"});
    const auto id = str(g.at("id"));
    const auto a = u64(g.at("start_ns")), b = u64(g.at("end_ns"));
    need(!id.empty() && id.size() <= 256 && ids.insert(id).second,
         "bounded unique group id required");
    need(
        a < b && a >= previous_end,
        "time groups must be nonempty, ordered and disjoint; gaps are allowed");
    previous_end = b;
  }
  const auto cutoff = u64(plan.at("fit_cutoff_ns"));
  const auto total = combinatorial::choose(groups.size(), k, end);
  auto rank = combinatorial::Natural(plan.at("fold_offset"));
  need(rank.compare(total) <= 0, "fold offset exceeds combination count");
  auto remaining = total;
  remaining.subtract(rank);
  // A single materialized page must be host-addressable. The complete space
  // and page offset do not have that restriction.
  auto count = plan.at("fold_count").is_null() ? u64(Json(remaining.text()))
                                               : u64(plan.at("fold_count"));
  need(combinatorial::Natural(count).compare(remaining) <= 0 &&
           count <= std::numeric_limits<std::size_t>::max(),
       "fold page exceeds remaining combinations or host address space");
  GroupedPlan out;
  out.groups = groups;
  std::vector<std::size_t> picked;
  if (count)
    picked = combinatorial::unrank(groups.size(), k, rank, end);
  for (U i = 0; i < count; ++i) {
    deadline(end);
    Json train = Json::array(), test = Json::array();
    std::size_t at = 0;
    for (std::size_t j = 0; j < groups.size(); ++j) {
      if (at < picked.size() && picked[at] == j) {
        test.push_back(dec(j));
        ++at;
      } else
        train.push_back(dec(j));
    }
    out.folds.push_back({{"id", "combination-" + rank.text()},
                         {"combination_rank", rank.text()},
                         {"train_groups", train},
                         {"test_groups", test},
                         {"fit_cutoff_ns", dec(cutoff)}});
    rank.increment();
    if (i + 1 < count)
      combinatorial::next(picked, groups.size());
  }
  out.selection = {
      {"total_combinations", total.text()},
      {"fold_offset", plan.at("fold_offset")},
      {"materialized_folds", dec(count)},
      {"next_fold_offset",
       rank.compare(total) < 0 ? Json(rank.text()) : Json(nullptr)},
      {"complete_space",
       plan.at("fold_offset") == "0" && rank.compare(total) == 0},
      {"test_appearances_per_group_in_complete_space",
       combinatorial::choose(groups.size() - 1, k - 1, end).text()},
      {"ordering", "lexicographic input group indices"}};
  return out;
}
Json fold_plan(const Json &plan) {
  const auto kind = str(plan.at("kind"));
  if (kind == "explicit") {
    keys(plan, {"kind", "folds"});
    need(plan.at("folds").is_array() && plan.at("folds").size() <= 128,
         "at most 128 explicit folds");
    return plan.at("folds");
  }
  need(kind == "rolling" || kind == "expanding", "unknown split schedule");
  keys(plan,
       {"kind", "train_start_ns", "first_train_end_ns", "first_fit_cutoff_ns",
        "gap_ns", "test_duration_ns", "step_ns", "fold_count"});
  const auto start = u64(plan.at("train_start_ns")),
             end = u64(plan.at("first_train_end_ns")),
             fit = u64(plan.at("first_fit_cutoff_ns")),
             gap = u64(plan.at("gap_ns")),
             duration = u64(plan.at("test_duration_ns")),
             step = u64(plan.at("step_ns")), count = u64(plan.at("fold_count"));
  need(start < end && duration > 0 && step > 0 && count > 0 && count <= 128,
       "invalid split schedule bounds");
  Json out = Json::array();
  U shift = 0;
  for (U i = 0; i < count; ++i) {
    auto train_end = add(end, shift), test_start = add(train_end, gap);
    out.push_back(
        {{"id", "fold-" + dec(i)},
         {"train_start_ns", dec(kind == "rolling" ? add(start, shift) : start)},
         {"train_end_ns", dec(train_end)},
         {"test_start_ns", dec(test_start)},
         {"test_end_ns", dec(add(test_start, duration))},
         {"fit_cutoff_ns", dec(add(fit, shift))}});
    if (i + 1 < count)
      shift = add(shift, step);
  }
  return out;
}
} // namespace

Json split(const Json &p, std::int64_t end) {
  keys(p, {"protocol",          "path",
           "expected_sha256",   "output_path",
           "pointer",           "id_pointer",
           "start_pointer",     "end_pointer",
           "available_pointer", "clock_domain",
           "input_description", "plan",
           "chronology",        "purge",
           "purge_before_ns",   "purge_after_ns",
           "embargo_ns",        "availability",
           "retain_rows",       "extensions"});
  need(p.at("extensions").is_object() && p.at("retain_rows").is_boolean(),
       "extensions and retention selection required");
  const auto chronology = str(p.at("chronology")), purge = str(p.at("purge")),
             availability = str(p.at("availability"));
  need(chronology == "past_only" || chronology == "unrestricted",
       "unknown chronology");
  need(purge == "label_overlap" || purge == "none", "unknown purging policy");
  need(availability == "by_fit_cutoff" || availability == "ignore",
       "unknown availability policy");
  const auto before = u64(p.at("purge_before_ns")),
             after = u64(p.at("purge_after_ns")),
             embargo = u64(p.at("embargo_ns"));
  need(purge != "none" || (before == 0 && after == 0),
       "purge padding requires label_overlap");
  for (auto name : {"clock_domain", "input_description"})
    need(!str(p.at(name)).empty() && str(p.at(name)).size() <= 4096,
         "bounded clock and description required");
  for (auto name : {"pointer", "id_pointer", "start_pointer", "end_pointer"})
    need(str(p.at(name)).size() <= 4096, "split pointer bound");
  const bool has_available = !p.at("available_pointer").is_null();
  need(availability != "by_fit_cutoff" || has_available,
       "availability evidence pointer required");
  if (has_available)
    need(str(p.at("available_pointer")).size() <= 4096,
         "availability pointer bound");
  auto source = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(source);
  need(!str(p.at("expected_sha256")).empty() &&
           source.at("content_sha256") == p.at("expected_sha256"),
       "split source digest mismatch");
  const Json::json_pointer ptr(str(p.at("pointer"))),
      idp(str(p.at("id_pointer"))), sp(str(p.at("start_pointer"))),
      ep(str(p.at("end_pointer"))),
      ap(has_available ? str(p.at("available_pointer")) : "");
  need(source.contains(ptr) && source.at(ptr).is_array() &&
           source.at(ptr).size() <= 65536,
       "bounded split source array required");
  const auto &rows = source.at(ptr);
  const bool grouped = str(p.at("plan").at("kind")) == "combinatorial";
  auto gp = grouped ? grouped_plan(p.at("plan"), end) : GroupedPlan{};
  const auto plans = grouped ? gp.folds : fold_plan(p.at("plan"));
  need(grouped || rows.size() * plans.size() <= 131072,
       "legacy split row-fold workload bound 131072");
  need(plans.empty() || rows.size() <= std::numeric_limits<std::size_t>::max() /
                                           plans.size(),
       "split row-fold size exceeds host representation");
  std::vector<Interval> group_windows;
  for (const auto &g : gp.groups)
    group_windows.push_back({u64(g.at("start_ns")), u64(g.at("end_ns"))});
  auto group_at = [&](U t) {
    auto it = std::upper_bound(group_windows.begin(), group_windows.end(), t,
                               [](U time, auto w) { return time < w.start; });
    if (it == group_windows.begin() || t >= (it - 1)->end)
      return group_windows.size();
    return static_cast<std::size_t>(it - group_windows.begin() - 1);
  };
  std::vector<U> test_uses(group_windows.size()),
      train_uses(group_windows.size());
  struct Observation {
    Interval label;
    U available;
  };
  std::vector<Observation> observations;
  std::vector<std::size_t> row_groups;
  Json table = Json::array();
  const bool retain = p.at("retain_rows").get<bool>();
  std::vector<std::size_t> row_bytes;
  std::size_t retained_bytes = 0;
  std::set<std::string> ids;
  for (std::size_t i = 0; i < rows.size(); ++i) {
    deadline(end);
    const auto &row = rows[i];
    need(row.contains(idp) && row.contains(sp) && row.contains(ep) &&
             (!has_available || row.contains(ap)),
         "split observation fields missing");
    auto id = str(row.at(idp));
    need(!id.empty() && id.size() <= 256 && ids.insert(id).second,
         "bounded unique observation id required");
    auto a = u64(row.at(sp)), b = u64(row.at(ep)),
         available = has_available ? u64(row.at(ap)) : 0;
    need(a <= b, "reversed label interval");
    observations.push_back({{a, b}, available});
    row_groups.push_back(group_at(a));
    row_bytes.push_back(retain ? row.dump().size() : 0);
    table.push_back({{"source_index", dec(i)},
                     {"id", id},
                     {"start_ns", dec(a)},
                     {"end_ns", dec(b)},
                     {"available_ns",
                      has_available ? Json(dec(available)) : Json(nullptr)}});
  }
  auto retain_row = [&](Json &target, std::size_t i) {
    if (!retain)
      return;
    need(row_bytes[i] <= (64U << 20) - retained_bytes,
         "retained split row bytes exceed 64 MiB; select index-only output "
         "explicitly");
    retained_bytes += row_bytes[i];
    target.push_back(rows[i]);
  };
  Json folds = Json::array();
  std::set<std::string> fold_ids;
  U ready = 0;
  for (const auto &f : plans) {
    deadline(end);
    const auto id = str(f.at("id"));
    need(!id.empty() && (grouped || id.size() <= 256) &&
             fold_ids.insert(id).second,
         "unique fold id required");
    const auto cutoff = u64(f.at("fit_cutoff_ns"));
    U train_start = 0, train_end = 0, test_start = 0, test_end = 0;
    std::vector<bool> held_out(group_windows.size());
    if (grouped) {
      for (const auto &g : f.at("test_groups")) {
        held_out[u64(g)] = true;
        ++test_uses[u64(g)];
      }
      test_start = group_windows[u64(f.at("test_groups")[0])].start;
      for (const auto &g : f.at("train_groups")) {
        train_end = std::max(train_end, group_windows[u64(g)].end);
        ++train_uses[u64(g)];
      }
    } else {
      keys(f, {"id", "train_start_ns", "train_end_ns", "test_start_ns",
               "test_end_ns", "fit_cutoff_ns"});
      train_start = u64(f.at("train_start_ns"));
      train_end = u64(f.at("train_end_ns"));
      test_start = u64(f.at("test_start_ns"));
      test_end = u64(f.at("test_end_ns"));
      need(train_start < train_end && test_start < test_end,
           "nonempty half-open selection windows required");
    }
    need(chronology != "past_only" ||
             (train_end <= test_start && cutoff <= test_start),
         "past_only requires all train windows and fit cutoff no later than "
         "first test start");
    auto in_test = [&](std::size_t i) {
      auto g = row_groups[i];
      return grouped ? g < held_out.size() && held_out[g]
                     : observations[i].label.start >= test_start &&
                           observations[i].label.start < test_end;
    };
    std::vector<U> group_label_ends(group_windows.size());
    std::vector<bool> populated(group_windows.size());
    std::vector<Interval> labels, padded;
    U last_label_end = 0;
    Json train = Json::array(), test = Json::array(), excluded = Json::array(),
         train_rows = Json::array(), test_rows = Json::array();
    for (std::size_t i = 0; i < observations.size(); ++i) {
      deadline(end);
      auto x = observations[i].label;
      if (!in_test(i))
        continue;
      test.push_back(dec(i));
      retain_row(test_rows, i);
      labels.push_back(x);
      // Overflow/underflow rejects rather than silently clipping the user's
      // interval.
      if (purge == "label_overlap")
        padded.push_back({sub(x.start, before), add(x.end, after)});
      last_label_end = std::max(last_label_end, x.end);
      if (grouped) {
        const auto g = row_groups[i];
        populated[g] = true;
        group_label_ends[g] = std::max(group_label_ends[g], x.end);
      }
    }
    labels = merge(std::move(labels));
    padded = merge(std::move(padded));
    const auto embargo_end =
        grouped || test.empty() ? 0 : add(last_label_end, embargo);
    Json embargo_records = Json::array();
    std::vector<Interval> embargo_union;
    if (grouped && embargo) {
      for (std::size_t g = 0; g < group_windows.size(); ++g) {
        if (!populated[g])
          continue;
        const auto a = group_label_ends[g], b = add(a, embargo);
        embargo_records.push_back({{"group_index", dec(g)},
                                   {"start_exclusive_ns", dec(a)},
                                   {"end_inclusive_ns", dec(b)}});
        // Integer nanoseconds let (a,b] use the shared closed-interval lookup.
        embargo_union.push_back({add(a, 1), b});
      }
      embargo_union = merge(std::move(embargo_union));
    }
    U candidates = 0, overlap_count = 0, late_count = 0, unknown_count = 0,
      embargo_count = 0, retained_overlaps = 0, retained_late = 0;
    for (std::size_t i = 0; i < observations.size(); ++i) {
      deadline(end);
      const auto x = observations[i];
      if (grouped ? row_groups[i] >= held_out.size() || held_out[row_groups[i]]
                  : x.label.start < train_start || x.label.start >= train_end)
        continue;
      ++candidates;
      const bool overlap = overlaps(labels, x.label),
                 late = has_available && x.available > cutoff,
                 embargoed = grouped ? overlaps(embargo_union,
                                                {x.label.start, x.label.start})
                                     : !test.empty() &&
                                           x.label.start > last_label_end &&
                                           x.label.start <= embargo_end;
      overlap_count += overlap;
      late_count += late;
      unknown_count += !has_available;
      embargo_count += embargoed;
      Json reasons = Json::array();
      if (in_test(i))
        reasons.push_back("test_member");
      if (purge == "label_overlap" && overlaps(padded, x.label))
        reasons.push_back("label_overlap_or_padding");
      if (embargoed)
        reasons.push_back("embargo");
      if (availability == "by_fit_cutoff" && late)
        reasons.push_back("unavailable_at_fit_cutoff");
      if (!reasons.empty())
        excluded.push_back({{"source_index", dec(i)}, {"reasons", reasons}});
      else {
        train.push_back(dec(i));
        retain_row(train_rows, i);
        retained_overlaps += overlap;
        retained_late += late;
      }
    }
    const bool usable = !train.empty() && !test.empty();
    ready += usable;
    folds.push_back(
        {{"id", id},
         {"windows", f},
         {"status", usable ? "available" : "unavailable"},
         {"reason", usable ? "" : "empty retained train or test selection"},
         {"train_indices", train},
         {"test_indices", test},
         {"excluded", excluded},
         {"train_rows", retain ? train_rows : Json(nullptr)},
         {"test_rows", retain ? test_rows : Json(nullptr)},
         {"test_label_union", intervals(labels)},
         {"purge_union", intervals(padded)},
         {"embargo_interval",
          grouped || test.empty() || embargo == 0
              ? Json(nullptr)
              : Json{{"start_exclusive_ns", dec(last_label_end)},
                     {"end_inclusive_ns", dec(embargo_end)}}},
         {"diagnostics",
          {{"train_candidates", dec(candidates)},
           {"overlap_candidates", dec(overlap_count)},
           {"late_candidates", dec(late_count)},
           {"unknown_availability_candidates", dec(unknown_count)},
           {"embargo_candidates", dec(embargo_count)},
           {"retained_overlaps", dec(retained_overlaps)},
           {"retained_late", dec(retained_late)}}}});
    if (grouped)
      folds.back()["embargo_intervals"] = embargo_records;
  }
  auto result = base(
      "user-selected temporal sample partitions; no model fitting executed");
  auto &s = result["sections"];
  s["summary"] = section({{"source_rows", dec(rows.size())},
                          {"folds", dec(folds.size())},
                          {"available_folds", dec(ready)},
                          {"clock_domain", p.at("clock_domain")}});
  if (grouped) {
    Json groups = gp.groups;
    for (std::size_t g = 0; g < groups.size(); ++g) {
      groups[g]["group_index"] = dec(g);
      groups[g]["source_indices"] = Json::array();
      groups[g]["selected_test_folds"] = dec(test_uses[g]);
      groups[g]["selected_train_candidate_folds"] = dec(train_uses[g]);
    }
    for (std::size_t i = 0; i < row_groups.size(); ++i)
      if (row_groups[i] < groups.size())
        groups[row_groups[i]]["source_indices"].push_back(dec(i));
    s["groups"] = section(groups);
    s["combination_selection"] = section(gp.selection);
  }
  s["observations"] = section(table);
  s["folds"] = section(folds);
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] =
      section({{"engine_version", version},
               {"source_result_sha256", source.at("content_sha256")},
               {"source_pointer", p.at("pointer")},
               {"profile", grouped ? "combinatorial_intervals_v1"
                                   : "temporal_intervals_v1"},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  s["resources"] =
      section({{"backend", "cpu"},
               {"actual_workers", "1"},
               {"row_fold_pairs", dec(rows.size() * plans.size())},
               {"retained_row_bytes", dec(retained_bytes)},
               {"interval_lookup", "merged union and binary search"}});
  s["diagnostics"] = section(Json::array(
      {"Window membership uses label start in [start,end); label overlap uses "
       "closed [start,end] intervals, including equal-time endpoints.",
       "Purging uses the actual selected test labels, with explicit "
       "before/after padding; embargo uses (maximum selected test label end, "
       "end+duration].",
       "Test membership wins over train membership in this partition profile. "
       "Rows outside both windows are retained in the source index table.",
       "Availability is a caller-declared evidence field, not verified "
       "feature/label lineage. Ignoring it or omitting purging is allowed and "
       "diagnosed.",
       "Input order is preserved in every index and retained-row selection; "
       "intervals alone are sorted internally. No weights are changed.",
       "No model was fit or scored. These partitions do not prove "
       "independence, complete causal features, holdout secrecy, or absence of "
       "prior selection. Reusing tests across folds remains visible.",
       "All intervals share the caller-declared nanosecond clock. Missing "
       "clocks, censored label ends and unrepresented feature lookbacks are "
       "not inferred."}));
  if (grouped) {
    s["diagnostics"]["data"][1] =
        "Purging uses the union of selected test labels with chosen padding. "
        "Embargo is applied separately after the maximum label end of each "
        "populated test group. Empty groups create no embargo.";
    s["diagnostics"]["data"][2] =
        "Test groups are selected k-of-N; training candidates come only from "
        "complementary groups. Group membership uses label start, not label "
        "end. Gaps and rows outside groups are not assigned.";
    s["diagnostics"]["data"].push_back(
        "Combinations and group-use counts describe the selected page. Folds "
        "share observations; no independent performance paths, fitted models, "
        "nested CV or untouched holdout are implied.");
  }
  s["source_context"] =
      section({{"choices", source.at("sections").at("choices")},
               {"provenance", source.at("sections").at("provenance")}});
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "split", end);
}
} // namespace symphony::sbv::detail
