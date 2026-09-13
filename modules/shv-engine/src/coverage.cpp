#include "shv.hpp"
#include <algorithm>
namespace symphony::knowledge::shv {
namespace {
void validate(const Json &s, std::size_t depth, std::size_t &count,
              const std::string &asof) {
  if (++count > 128 || depth > 16)
    invalid("selector work bound exceeded");
  auto op = text(s, "op", 16);
  if (op == "all") {
    fields(s, {"op"});
    return;
  }
  if (op == "date") {
    fields(s, {"op", "basis", "from", "through"});
    if (text(s, "basis", 32) != "model_introduction")
      invalid("unsupported date basis; no manufacture inference");
    auto a = text(s, "from", 10), b = text(s, "through", 10);
    if (!date_valid(a) || !date_valid(b) || a > b || b > asof)
      invalid("invalid selector date interval");
    return;
  }
  if (op == "ids" || op == "class" || op == "manufacturer") {
    fields(s, {"op", "values"});
    array(s.at("values"), 128);
    std::set<std::string> seen;
    for (const auto &v : s.at("values")) {
      Json x = {{"value", v}};
      auto key =
          op == "manufacturer" ? text(x, "value", 256) : ident(x, "value");
      if (!seen.insert(key).second)
        invalid("duplicate selector value");
    }
    return;
  }
  if (op == "and" || op == "or") {
    fields(s, {"op", "args"});
    array(s.at("args"), 32);
    if (s.at("args").empty())
      invalid("empty boolean selector");
    for (const auto &x : s.at("args"))
      validate(x, depth + 1, count, asof);
    return;
  }
  if (op == "not") {
    fields(s, {"op", "arg"});
    validate(s.at("arg"), depth + 1, count, asof);
    return;
  }
  invalid("unsupported selector operator");
}
// -1 is unresolved, 0 excluded, 1 included. No missing date is invented.
int match(const Json &s, const Json &subject) {
  auto op = s.at("op").get<std::string>();
  if (op == "all")
    return 1;
  if (op == "date") {
    const auto &d = subject.at("introduced");
    if (d.is_null())
      return -1;
    auto a = d.at("from").get<std::string>(),
         b = d.at("through").get<std::string>();
    auto from = s.at("from").get<std::string>(),
         through = s.at("through").get<std::string>();
    if (b < from || a > through)
      return 0;
    if (a >= from && b <= through)
      return 1;
    return -1;
  }
  if (op == "ids" || op == "class" || op == "manufacturer") {
    auto key = op == "ids"
                   ? "id"
                   : (op == "class" ? "hardware_class" : "manufacturer");
    for (const auto &v : s.at("values"))
      if (v == subject.at(key))
        return 1;
    return 0;
  }
  if (op == "not") {
    int v = match(s.at("arg"), subject);
    return v < 0 ? -1 : 1 - v;
  }
  bool unknown = false;
  for (const auto &child : s.at("args")) {
    int v = match(child, subject);
    if (op == "and" && v == 0)
      return 0;
    if (op == "or" && v == 1)
      return 1;
    unknown |= v < 0;
  }
  return unknown ? -1 : (op == "and" ? 1 : 0);
}
} // namespace
Json coverage_default(const Json &p) {
  fields(p, {"as_of"});
  auto asof = text(p, "as_of", 10);
  if (!date_valid(asof) || asof < "2018-01-01")
    invalid("default as_of precedes start or is invalid; use an explicit "
            "caller profile");
  return seal(Json{{"protocol", "symphony.shv.coverage-profile.v1"},
                   {"as_of", asof},
                   {"selector", Json{{"op", "date"},
                                     {"basis", "model_introduction"},
                                     {"from", "2018-01-01"},
                                     {"through", asof}}}});
}
Json coverage_plan(const Json &p) {
  fields(p, {"profile", "subjects"});
  const auto &profile = p.at("profile");
  fields(profile, {"protocol", "as_of", "selector", "digest"});
  check_seal(profile);
  if (text(profile, "protocol") != "symphony.shv.coverage-profile.v1")
    invalid("unsupported profile");
  auto asof = text(profile, "as_of", 10);
  if (!date_valid(asof))
    invalid("invalid as_of");
  std::size_t count = 0;
  validate(profile.at("selector"), 0, count, asof);
  array(p.at("subjects"), 128);
  std::set<std::string> ids;
  Json decisions = Json::array();
  Json counts = {{"included", 0}, {"excluded", 0}, {"unresolved", 0}};
  for (const auto &s : p.at("subjects")) {
    summary(s);
    if (!ids.insert(ident(s, "id")).second)
      invalid("duplicate subject");
    int v = match(profile.at("selector"), s);
    std::string status = v < 0 ? "unresolved" : (v ? "included" : "excluded");
    counts[status] = counts[status].get<int>() + 1;
    decisions.push_back(Json{{"subject_id", s.at("id")}, {"status", status}});
  }
  std::sort(decisions.begin(), decisions.end(),
            [](const auto &a, const auto &b) {
              return a.at("subject_id") < b.at("subject_id");
            });
  return seal(Json{{"protocol", "symphony.shv.coverage-result.v1"},
                   {"profile", profile},
                   {"subjects", p.at("subjects")},
                   {"decisions", decisions},
                   {"counts", counts}});
}
} // namespace symphony::knowledge::shv
