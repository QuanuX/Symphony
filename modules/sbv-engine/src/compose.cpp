#include "rational.hpp"
#include <map>
#include <numeric>
namespace symphony::sbv::detail {
using namespace rational;
Json compose(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "returns", "weights", "steps", "dependence",
           "output_path", "extensions"});
  const auto &returns = p.at("returns");
  const auto &weights = p.at("weights");
  need(returns.is_array() && weights.is_array() &&
           returns.size() == weights.size() && !returns.empty() &&
           returns.size() <= 64,
       "explicit matched support and measure required");
  need(p.at("extensions").is_object(), "extensions object required");
  const auto steps = u64(p.at("steps"));
  need(steps >= 1 && steps <= 16, "steps 1..16");
  const auto dependence = str(p.at("dependence"));
  need(dependence == "independent" || dependence == "same_draw",
       "explicit dependence model required");
  std::vector<R> growth, measure;
  R total;
  for (std::size_t i = 0; i < returns.size(); ++i) {
    auto r = read_ratio(returns[i]), w = read_ratio(weights[i]);
    need(r.n >= -r.d && w.n >= 0,
         "wealth cannot be negative and weights must be nonnegative");
    growth.push_back(plus({1, 1}, r));
    measure.push_back(w);
    total = plus(total, w);
  }
  need(total.n == total.d,
       "weights must sum exactly to one; normalization is never implicit");
  std::size_t count = returns.size();
  if (dependence == "independent")
    for (std::size_t i = 1; i < steps; ++i) {
      need(count <= 65536 / returns.size(),
           "path enumeration bound; choose fewer steps/support points");
      count *= returns.size();
    }
  Json paths = Json::array();
  struct Atom {
    R wealth, mass;
  };
  std::map<std::string, Atom> atoms;
  for (std::size_t i = 0; i < count; ++i) {
    deadline(end);
    std::size_t code = i;
    R wealth{1, 1}, mass{1, 1};
    Json trajectory = Json::array({wire(wealth)}), draws = Json::array();
    for (std::size_t j = 0; j < steps; ++j) {
      const auto draw = dependence == "same_draw" ? i : code % returns.size();
      code /= returns.size();
      wealth = times(wealth, growth[draw]);
      if (dependence == "independent" || j == 0)
        mass = times(mass, measure[draw]);
      draws.push_back(dec(draw));
      trajectory.push_back(wire(wealth));
    }
    paths.push_back({{"path_id", dec(i)},
                     {"draws", draws},
                     {"wealth", trajectory},
                     {"mass", wire(mass)}});
    auto key = dec(wealth.n) + "/" + dec(wealth.d);
    auto [it, fresh] = atoms.emplace(key, Atom{wealth, {0, 1}});
    (void)fresh;
    it->second.mass = plus(it->second.mass, mass);
  }
  Json terminal = Json::array();
  R terminal_mass;
  for (const auto &[key, a] : atoms) {
    (void)key;
    terminal.push_back({{"wealth", wire(a.wealth)}, {"mass", wire(a.mass)}});
    terminal_mass = plus(terminal_mass, a.mass);
  }
  need(terminal_mass.n == terminal_mass.d, "terminal measure invariant");
  auto r = base("native exact rational scenario composition; user-supplied "
                "support and measure");
  auto &s = r["sections"];
  s["summary"] = section({{"path_count", dec(paths.size())},
                          {"terminal_atom_count", dec(terminal.size())},
                          {"steps", dec(steps)},
                          {"total_mass", wire(terminal_mass)}});
  s["distributions"] = section(
      {{"economics", "sequential multiplicative wealth; initial wealth one"},
       {"support", returns},
       {"measure", weights},
       {"dependence", dependence},
       {"paths", paths},
       {"terminal_atoms", terminal},
       {"numeric_contract", "reduced exact signed int64 rationals; overflow "
                            "rejects rather than rounds"},
       {"calibration", "user assumptions; not calibrated"}});
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] = section({{"engine_version", version},
                             {"input_kind", "explicit user scenario model"},
                             {"market_data", "none"}});
  s["resources"] = section({{"backend", "cpu"}, {"workers", "1"}});
  s["diagnostics"] = section(Json::array(
      {"Path values and probabilities are separate. Mixed paths are one part "
       "of the full distribution.",
       "No implicit reweighting, independence or economic convention."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(r), p, "compose", end);
}
} // namespace symphony::sbv::detail
