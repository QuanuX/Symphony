#include "shv.hpp"
#include <set>
namespace symphony::knowledge::shv {
Json table_value(const Json &rows, const Json &mapping) {
  auto type = text(mapping, "value_type", 16);
  if (type == "table_rows") {
    fields(mapping,
           {"predicate", "section", "columns", "value_type", "qualifier"});
    const auto &columns = mapping.at("columns");
    array(columns, 16);
    if (columns.empty() || rows.size() < 2 || rows.size() > 33)
      invalid("matrix dimensions outside bounds");
    std::set<std::string> names, keys;
    for (const auto &column : columns) {
      if (!names.insert(text(Json{{"column", column}}, "column", 256)).second)
        invalid("duplicate matrix column");
    }
    Json result = Json::array();
    for (std::size_t i = 0; i < rows.size(); ++i) {
      const auto &row = rows[i];
      if (row.size() != columns.size())
        invalid("matrix width changed");
      Json values = Json::array();
      for (std::size_t j = 0; j < row.size(); ++j) {
        if (row[j].at("tag") != (i == 0 ? "th" : "td"))
          invalid("matrix cell kind changed");
        if (i == 0 && row[j].at("value") != columns[j])
          invalid("matrix header changed");
        values.push_back(row[j].at("value"));
      }
      if (i != 0) {
        auto key = values[0].get<std::string>();
        if (key.empty() || !keys.insert(key).second)
          invalid("empty or duplicate matrix row identity");
        result.push_back(values);
      }
    }
    return Json{{"columns", columns}, {"rows", result}};
  }
  fields(mapping, {"predicate", "section", "label", "next_label", "value_type",
                   "qualifier"});
  const auto label = text(mapping, "label", 256);
  const bool last = mapping.at("next_label").is_null();
  const auto next = last ? std::string{} : text(mapping, "next_label", 256);
  std::set<std::string> labels;
  Json result;
  bool found = false;
  for (std::size_t i = 0; i < rows.size(); ++i) {
    const auto &row = rows[i];
    if (row.size() != 2 || row[0].at("tag") != "th" || row[1].at("tag") != "td")
      invalid("scalar table requires exact TH/TD rows");
    const auto key = text(row[0], "value", 256);
    if (!labels.insert(key).second)
      invalid("duplicate scalar table label");
    if (key == label) {
      if (last ? i + 1 != rows.size()
               : (i + 1 == rows.size() || rows[i + 1][0].at("value") != next))
        invalid("changed adjacent table field");
      result = source_value(row[1].at("value").get<std::string>(), type);
      found = true;
    }
  }
  if (!found)
    invalid("table field unavailable");
  return result;
}
} // namespace symphony::knowledge::shv
