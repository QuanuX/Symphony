#include "rational.hpp"
#include <map>
namespace symphony::sbv::detail {
Json compose_joint(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "paths", "output_path", "extensions"});
  need(p.at("extensions").is_object(), "extensions object required");
  const auto &input = p.at("paths");
  need(input.is_array() && !input.empty() && input.size() <= 4096,
       "explicit joint paths 1..4096 required");
  struct Atom {
    rational::R wealth, mass;
  };
  std::map<std::string, Atom> atoms;
  std::set<std::string> ids;
  Json paths = Json::array();
  rational::R total;
  std::size_t steps = 0;
  for (const auto &path : input) {
    deadline(end);
    keys(path, {"path_id", "returns", "probability"});
    const auto id = str(path.at("path_id"));
    need(!id.empty() && id.size() <= 256 && ids.insert(id).second,
         "unique bounded path identity required");
    const auto &returns = path.at("returns");
    need(returns.is_array() && !returns.empty() && returns.size() <= 16,
         "joint path steps 1..16");
    if (steps == 0)
      steps = returns.size();
    need(returns.size() == steps,
         "joint paths require the same declared number of steps");
    auto mass = rational::read_ratio(path.at("probability"));
    need(mass.n >= 0, "joint path probability cannot be negative");
    total = rational::plus(total, mass);
    rational::R wealth{1, 1};
    Json trajectory = Json::array({rational::wire(wealth)});
    for (const auto &v : returns) {
      const auto r = rational::read_ratio(v);
      need(r.n >= -r.d,
           "multiplicative wealth domain requires returns at least -1");
      wealth = rational::times(wealth, rational::plus({1, 1}, r));
      trajectory.push_back(rational::wire(wealth));
    }
    paths.push_back({{"path_id", id},
                     {"returns", returns},
                     {"mass", rational::wire(mass)},
                     {"wealth", trajectory}});
    const auto key = dec(wealth.n) + "/" + dec(wealth.d);
    auto [it, fresh] = atoms.emplace(key, Atom{wealth, {0, 1}});
    (void)fresh;
    it->second.mass = rational::plus(it->second.mass, mass);
  }
  need(total.n == total.d,
       "joint probability must sum exactly to one; no implicit normalization");
  Json terminal = Json::array();
  for (const auto &[key, a] : atoms) {
    (void)key;
    terminal.push_back({{"wealth", rational::wire(a.wealth)},
                        {"mass", rational::wire(a.mass)}});
  }
  auto result =
      base("native exact composition of explicitly supplied joint paths");
  auto &s = result["sections"];
  s["summary"] = section({{"path_count", dec(paths.size())},
                          {"steps", dec(steps)},
                          {"terminal_atom_count", dec(terminal.size())},
                          {"total_mass", rational::wire(total)}});
  s["distributions"] = section(
      {{"dependence", "explicit_joint_paths"},
       {"paths", paths},
       {"terminal_atoms", terminal},
       {"economics", "sequential multiplicative wealth from one"},
       {"measure", "user-supplied exact path probability"},
       {"numeric_contract", "exact reduced int64 rationals; overflow rejects"},
       {"calibration", "not verified"}});
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] =
      section({{"engine_version", version},
               {"input_kind", "explicit user joint scenario"},
               {"market_feasibility",
                "not verified; only supplied paths are enumerated"}});
  s["resources"] = section({{"backend", "cpu"}, {"workers", "1"}});
  s["diagnostics"] = section(
      Json::array({"Marginal support never authorizes an unlisted path. No "
                   "independence assumption or extremum fabrication.",
                   "Admitted paths are user scenarios, not proof of "
                   "shared-liquidity or capital feasibility."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "compose-joint", end);
}
} // namespace symphony::sbv::detail
