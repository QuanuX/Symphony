#include "wide_rational.hpp"
#include <algorithm>
#include <atomic>
#include <mutex>
#include <thread>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
// SplitMix64 permutation constants match Sebastiano Vigna's public-domain
// reference: https://prng.di.unimi.it/splitmix64.c . State is per replica.
struct Generator {
  std::uint64_t state;
  std::uint64_t next() {
    auto z = (state += 0x9e3779b97f4a7c15ULL);
    z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9ULL;
    z = (z ^ (z >> 27)) * 0x94d049bb133111ebULL;
    return z ^ (z >> 31);
  }
  std::uint64_t below(std::uint64_t n) {
    const auto threshold = (std::uint64_t{0} - n) % n;
    for (;;) {
      auto r = next();
      if (r >= threshold)
        return r % n;
    }
  }
};
} // namespace
Json resample(const Json &p, std::int64_t end) {
  keys(p,
       {"protocol",  "path",          "expected_sha256",   "output_path",
        "pointer",   "value_pointer", "value_type",        "sampling_measure",
        "scheme",    "sample_size",   "block_size",        "replicates",
        "seed",      "workers",       "retain_indices",    "studies",
        "quantiles", "unit",          "input_description", "extensions"});
  need(p.at("extensions").is_object() && p.at("retain_indices").is_boolean(),
       "extensions and index policy required");
  need(p.at("sampling_measure") == "uniform_rows",
       "only explicit uniform row sampling is currently implemented");
  const auto scheme = str(p.at("scheme"));
  need(scheme == "iid" || scheme == "moving_block" ||
           scheme == "circular_block",
       "unknown bootstrap scheme");
  need(p.at("value_type") == "integer" || p.at("value_type") == "rational",
       "exact source type required");
  need(!str(p.at("unit")).empty() && str(p.at("unit")).size() <= 128 &&
           !str(p.at("input_description")).empty() &&
           str(p.at("input_description")).size() <= 4096,
       "unit and input description required");
  const auto count = u64(p.at("replicates")), size = u64(p.at("sample_size")),
             block = u64(p.at("block_size")), seed = u64(p.at("seed")),
             workers = u64(p.at("workers"));
  need(count >= 1 && count <= 1024 && size >= 1 && size <= 65536 &&
           count * size <= 1048576 && block >= 1 && workers >= 1 &&
           workers <= 64,
       "bootstrap workload bounds");
  const bool retain = p.at("retain_indices").get<bool>();
  need(!retain || count * size <= 65536, "retained index bound 65536");
  need(scheme != "iid" || block == 1, "iid sampling requires block_size one");
  need(p.at("studies").is_array() && p.at("studies").size() <= 2,
       "bootstrap study bound");
  std::set<std::string> studies;
  for (const auto &id : p.at("studies")) {
    auto s = str(id);
    need((s == "bootstrap_mean_distribution" ||
          s == "bootstrap_mean_quantiles") &&
             studies.insert(s).second,
         "unknown/duplicate bootstrap study");
  }
  need(p.at("quantiles").is_array() && p.at("quantiles").size() <= 32 &&
           (studies.contains("bootstrap_mean_quantiles")
                ? !p.at("quantiles").empty()
                : p.at("quantiles").empty()),
       "selected bootstrap quantiles required");
  std::vector<w::R> quantiles;
  for (const auto &q : p.at("quantiles")) {
    auto r = w::parameter(q);
    need(r.n >= 0 && r.n <= r.d, "bootstrap quantile outside [0,1]");
    quantiles.push_back(r);
  }
  auto source = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(source);
  need(!str(p.at("expected_sha256")).empty() &&
           source.at("content_sha256") == p.at("expected_sha256"),
       "bootstrap source digest mismatch");
  const auto pointer = str(p.at("pointer")),
             value_pointer = str(p.at("value_pointer"));
  need(pointer.size() <= 4096 && value_pointer.size() <= 4096,
       "bootstrap pointer bounds");
  Json::json_pointer ptr(pointer), vp(value_pointer);
  need(source.contains(ptr) && source.at(ptr).is_array() &&
           !source.at(ptr).empty() && source.at(ptr).size() <= 65536,
       "nonempty bounded source array required");
  const auto &rows = source.at(ptr);
  need(block <= rows.size(), "block cannot exceed source length");
  std::vector<w::R> values;
  for (const auto &row : rows) {
    deadline(end);
    need(row.contains(vp), "bootstrap value absent; select/exclude "
                           "observations before resampling");
    values.push_back(p.at("value_type") == "integer"
                         ? w::R{w::integer(row.at(vp)), 1}
                         : w::artifact(row.at(vp)));
  }
  struct Draw {
    w::R mean;
    Json indices = Json::array();
  };
  std::vector<Draw> draws(count);
  std::atomic<std::size_t> next{0};
  std::atomic<bool> stop{false};
  std::mutex errors;
  std::exception_ptr failure;
  const auto actual = std::min<std::uint64_t>(workers, count);
  auto work = [&] {
    try {
      while (!stop) {
        auto replicate = next.fetch_add(1);
        if (replicate >= count)
          break;
        Generator mix{seed + static_cast<std::uint64_t>(replicate) *
                                 0xd1342543de82ef95ULL};
        Generator rng{mix.next()};
        auto &draw = draws[replicate];
        w::R sum{};
        std::uint64_t taken = 0;
        while (taken < size) {
          deadline(end);
          const auto start =
              rng.below(scheme == "moving_block" ? values.size() - block + 1
                                                 : values.size());
          for (std::uint64_t offset = 0; offset < block && taken < size;
               ++offset, ++taken) {
            const auto idx = (start + offset) % values.size();
            sum = w::plus(sum, values[idx]);
            if (retain)
              draw.indices.push_back(dec(idx));
          }
        }
        draw.mean = w::divide(sum, {static_cast<w::I>(size), 1});
      }
    } catch (...) {
      std::lock_guard guard(errors);
      if (!failure)
        failure = std::current_exception();
      stop = true;
    }
  };
  std::vector<std::jthread> pool;
  try {
    for (std::uint64_t i = 0; i < actual; ++i)
      pool.emplace_back(work);
  } catch (...) {
    stop = true;
    throw;
  }
  pool.clear();
  if (failure)
    std::rethrow_exception(failure);
  deadline(end);
  Json rep = Json::array();
  std::vector<w::R> ordered;
  for (std::size_t i = 0; i < draws.size(); ++i) {
    rep.push_back(
        {{"replicate", dec(i)},
         {"mean", w::wire(draws[i].mean)},
         {"source_indices", retain ? draws[i].indices : Json(nullptr)}});
    ordered.push_back(draws[i].mean);
  }
  Json results = Json::array();
  if (studies.contains("bootstrap_mean_distribution")) {
    w::R mean{}, variance{};
    for (const auto &d : draws)
      mean = w::plus(mean, w::times(d.mean, {1, static_cast<w::I>(count)}));
    for (const auto &d : draws) {
      auto delta = w::plus(d.mean, {-mean.n, mean.d});
      variance = w::plus(variance, w::times(w::times(delta, delta),
                                            {1, static_cast<w::I>(count)}));
    }
    results.push_back({{"study_id", "bootstrap_mean_distribution"},
                       {"version", "1"},
                       {"status", "available"},
                       {"reason", ""},
                       {"data",
                        {{"mean_of_means", w::wire(mean)},
                         {"population_variance_of_means", w::wire(variance)},
                         {"replicate_count", dec(count)}}}});
  }
  if (studies.contains("bootstrap_mean_quantiles")) {
    std::sort(ordered.begin(), ordered.end(),
              [](auto a, auto b) { return w::compare(a, b) < 0; });
    Json qs = Json::array();
    for (auto q : quantiles) {
      std::size_t idx = 0;
      while (idx + 1 < count &&
             w::compare({static_cast<w::I>(idx + 1), static_cast<w::I>(count)},
                        q) < 0)
        ++idx;
      qs.push_back({{"q", w::wire(q)}, {"value", w::wire(ordered[idx])}});
    }
    results.push_back(
        {{"study_id", "bootstrap_mean_quantiles"},
         {"version", "1"},
         {"status", "available"},
         {"reason", ""},
         {"data",
          {{"rule",
            "inverse CDF of equally weighted replicate means; q=0 minimum"},
           {"quantiles", qs},
           {"interpretation", "empirical bootstrap percentiles; confidence "
                              "coverage is not established"}}}});
  }
  auto result =
      base("reproducible uniform-row bootstrap of supplied observations");
  auto &s = result["sections"];
  s["summary"] = section({{"source_rows", dec(values.size())},
                          {"replicates", dec(count)},
                          {"sample_size", dec(size)},
                          {"scheme", scheme},
                          {"unit", p.at("unit")}});
  s["resamples"] = section(rep);
  s["studies"] =
      section(results, studies.empty() ? "not_selected" : "available",
              studies.empty() ? "zero studies selected" : "");
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["resources"] = section(
      {{"backend", "cpu"},
       {"requested_workers", dec(workers)},
       {"actual_workers", dec(actual)},
       {"sampled_values", dec(count * size)},
       {"order", "replicate ordinal, independent of worker scheduling"}});
  s["provenance"] = section(
      {{"engine_version", version},
       {"source_result_sha256", source.at("content_sha256")},
       {"pointer", pointer},
       {"generator", "splitmix64-v1 rejection-modulo; replicate seed=next(seed "
                     "+ replicate*0xd1342543de82ef95 modulo 2^64)"},
       {"numeric_profile", "checked_signed_128_rational"},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"}});
  s["diagnostics"] = section(Json::array(
      {"Uniform row resampling is selected explicitly; source "
       "probability/scenario weights are not imported.",
       "IID draws discard serial dependence. Moving blocks preserve "
       "within-block source order; circular blocks wrap at the source end. "
       "Dependence across blocks and market validity are not inferred.",
       "Block draws concatenate until selected sample size and truncate the "
       "last block when needed.",
       "Bootstrap percentiles are empirical resampling summaries, not "
       "automatically confidence intervals or predictive performance.",
       "Training/validation separation and multiple-testing correction remain "
       "caller-selected external protocols."}));
  s["source_context"] =
      section({{"choices", source.at("sections").at("choices")},
               {"provenance", source.at("sections").at("provenance")}});
  const auto &parent = source.at("sections");
  if (parent.at("resources").at("data").contains("dataset_feed"))
    s["source_context"]["data"]["dataset_feed"] =
        parent.at("resources").at("data").at("dataset_feed");
  else if (parent.contains("source_context") &&
           parent.at("source_context").at("data").contains("dataset_feed"))
    s["source_context"]["data"]["dataset_feed"] =
        parent.at("source_context").at("data").at("dataset_feed");
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "resample", end);
}
} // namespace symphony::sbv::detail
