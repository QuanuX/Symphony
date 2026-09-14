#include "pdf.hpp"
#include "shv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include <algorithm>
#include <charconv>
#include <filesystem>
#include <map>
#include <string_view>
static_assert(std::string_view(symphony::knowledge::shv_pdf::version) ==
                  "0.2.0-dev",
              "Review the exact document reader contract before upgrading the "
              "kernel dependency");
namespace symphony::knowledge::shv {
Json source_value(const std::string &input, const std::string &type) {
  if (input.empty() || input.size() > 4096)
    invalid("field value outside bounds");
  if (type == "string")
    return input;
  if (type == "integer") {
    std::int64_t n = 0;
    auto [p, e] = std::from_chars(input.data(), input.data() + input.size(), n);
    if (e != std::errc{} || p != input.data() + input.size() ||
        std::to_string(n) != input || n < -9007199254740991LL ||
        n > 9007199254740991LL)
      invalid("noncanonical integer source value");
    return n;
  }
  if (type == "date") {
    if (input.size() != 10 || input[2] != '/' || input[5] != '/')
      invalid("unsupported source date format");
    auto iso = input.substr(6, 4) + "-" + input.substr(0, 2) + "-" +
               input.substr(3, 2);
    if (!date_valid(iso))
      invalid("invalid source date");
    return iso;
  }
  if (type == "quarter_20yy") {
    if (input.size() != 5 || input[0] != 'Q' || input[1] < '1' ||
        input[1] > '4' || input[2] != '\'' || input[3] < '0' ||
        input[3] > '9' || input[4] < '0' || input[4] > '9')
      invalid("unsupported 2000-2099 source quarter");
    const std::string starts[] = {"01-01", "04-01", "07-01", "10-01"};
    const std::string ends[] = {"03-31", "06-30", "09-30", "12-31"};
    auto year = "20" + input.substr(3);
    auto q = input[1] - '1';
    return Json{{"precision", "quarter"},
                {"source_text", input},
                {"from", year + "-" + starts[q]},
                {"through", year + "-" + ends[q]}};
  }
  if (type == "tokens") {
    std::set<std::string> seen;
    std::size_t start = 0;
    while (start <= input.size()) {
      auto end = input.find('/', start);
      if (end == input.npos)
        end = input.size();
      auto first = start, last = end;
      while (first < last && input[first] == ' ')
        ++first;
      while (last > first && input[last - 1] == ' ')
        --last;
      auto token = input.substr(first, last - first);
      if (token.empty() || token.size() > 128 || !seen.insert(token).second ||
          seen.size() > 32)
        invalid("invalid finite token set");
      if (end == input.size())
        break;
      start = end + 1;
    }
    Json result = Json::array();
    for (const auto &token : seen)
      result.push_back(token);
    return result;
  }
  invalid("unsupported value_type");
}
namespace {
std::map<std::string, std::pair<std::string, std::string>>
pairs(const std::string &normalized) {
  std::vector<std::string> lines;
  std::size_t start = 0;
  while (start < normalized.size()) {
    auto end = normalized.find('\n', start);
    if (end == normalized.npos)
      invalid("invalid normalized fields");
    lines.push_back(normalized.substr(start, end - start));
    start = end + 1;
  }
  if (lines.size() % 2)
    invalid("unpaired normalized fields");
  std::map<std::string, std::pair<std::string, std::string>> result;
  for (std::size_t i = 0; i < lines.size(); i += 2)
    result[lines[i]] = {lines[i + 1], i + 2 < lines.size() ? lines[i + 2] : ""};
  return result;
}
} // namespace
Json catalogue_build(const engine::Request &r, const Json &p) {
  fields(p, {"source_root", "sources", "subjects"});
  auto root = text(p, "source_root", 4096);
  if (!std::filesystem::path(root).is_absolute())
    invalid("source_root must be absolute");
  array(p.at("sources"), 8);
  array(p.at("subjects"), 32);
  std::map<std::string, std::string> bytes, formats;
  std::set<std::string> paths;
  std::uint64_t total = 0;
  for (const auto &source : p.at("sources")) {
    deadline(r);
    fields(source, {"id", "path", "bytes", "digest", "format"});
    auto id = ident(source, "id"), path = text(source, "path", 4096),
         digest = text(source, "digest", 71),
         format = text(source, "format", 16);
    if (formats.contains(id) || !paths.insert(path).second ||
        !engine::is_safe_relative_path(path))
      invalid("duplicate or unsafe source identity/path");
    if (format != "html" && format != "opaque")
      invalid("unsupported source format");
    if (!source.at("bytes").is_number_unsigned() &&
        !source.at("bytes").is_number_integer())
      invalid("source bytes must be integer");
    if (source.at("bytes") < 0 || source.at("bytes") > 1048576)
      invalid("source byte bound exceeded");
    auto size = source.at("bytes").get<std::uint64_t>();
    total += size;
    if (total > 4194304)
      invalid("cumulative source byte bound exceeded");
    auto raw = engine::read_regular_file_no_follow(root, path, 1048576,
                                                   r.deadline_unix_ms);
    if (raw.size() != size || engine::tagged_sha256(raw) != digest)
      invalid("consumed source bytes differ from manifest");
    bytes[id] = std::move(raw);
    formats[id] = format;
  }
  std::map<std::string, Json> document_cache;
  Json subjects = Json::array();
  std::set<std::string> ids;
  std::size_t assertions = 0;
  for (const auto &spec : p.at("subjects")) {
    deadline(r);
    const bool pdf =
        spec.value("interpretation_profile", std::string{}) == "pdf_opn.v1";
    const bool tables = !pdf && spec.contains("interpretation_profile");
    if (pdf) {
      fields(spec, {"id", "manufacturer", "model", "hardware_class",
                    "source_id", "heading_section", "interpretation_profile",
                    "document", "fields"});
    } else if (tables) {
      fields(spec,
             {"id", "manufacturer", "model", "hardware_class", "source_id",
              "heading_section", "interpretation_profile", "fields"});
      if (text(spec, "interpretation_profile") != "scoped_tables.v1")
        invalid("unsupported interpretation profile");
    } else {
      fields(spec, {"id", "manufacturer", "model", "hardware_class",
                    "source_id", "heading_section", "field_section", "fields"});
      text(spec, "field_section", 128);
    }
    text(spec, "heading_section", 128);
    auto id = ident(spec, "id"), model = text(spec, "model", 256),
         manufacturer = text(spec, "manufacturer", 256),
         hardware = ident(spec, "hardware_class"),
         source = ident(spec, "source_id");
    if (!ids.insert(id).second || !formats.contains(source))
      invalid("duplicate subject or missing source");
    array(spec.at("fields"), 16);
    assertions += spec.at("fields").size();
    if (assertions > 256)
      invalid("assertion bound exceeded");
    Json pdf_row;
    if (pdf) {
      if (formats.at(source) != "opaque" ||
          spec.at("heading_section") != "table8")
        invalid("PDF mapping source/heading mismatch");
      const auto &document = spec.at("document");
      fields(document, {"decoder_root", "extraction"});
      const auto &x = document.at("extraction");
      Json manifest;
      for (const auto &s : p.at("sources"))
        if (s.at("id") == source) {
          manifest = s;
          manifest.erase("format");
        }
      auto request = r;
      request.operation = "extract";
      request.payload = Json{{"source_root", root},
                             {"source", manifest},
                             {"decoder_root", document.at("decoder_root")},
                             {"decoder", x.at("decoder")},
                             {"profile", x.at("profile")}};
      const auto key = request.payload.dump();
      if (!document_cache.contains(key))
        document_cache[key] =
            symphony::knowledge::shv_pdf::handle_request(request);
      const auto &replay = document_cache.at(key);
      if (replay != x)
        invalid("PDF derivation differs from original-byte replay");
      for (const auto &row : replay.at("rows"))
        if (row.at("model") == model)
          pdf_row = row;
      if (pdf_row.is_null())
        invalid("PDF model absent from selected table");
    }
    if (!pdf && !spec.at("fields").empty() && formats.at(source) != "html")
      invalid("opaque source cannot emit interpreted assertions");
    auto normalized = spec.at("fields").empty() || tables || pdf
                          ? std::string{}
                          : html_text(bytes.at(source), model,
                                      text(spec, "heading_section", 128),
                                      text(spec, "field_section", 128));
    auto entries = pairs(normalized);
    Json values = Json::array(), introduced = nullptr;
    std::set<std::string> predicates;
    std::map<std::string, Json> table_cache;
    for (const auto &f : spec.at("fields")) {
      auto pred = ident(f, "predicate"), type = text(f, "value_type", 16),
           qualifier = text(f, "qualifier", 256);
      if (!predicates.insert(pred).second)
        invalid("duplicate predicate: conflicting input requires a later "
                "explicit conflict contract");
      Json typed;
      if (pdf) {
        fields(f,
               {"predicate", "label", "next_label", "value_type", "qualifier"});
        if (type != "string" || f.at("label") != "OPN" ||
            f.at("next_label") != "Model" ||
            qualifier != "issuer=AMD;namespace=opn;profile=1")
          invalid("unsupported PDF field mapping");
        typed = pdf_row.at("opn");
      } else if (tables) {
        auto section = text(f, "section", 128);
        if (!table_cache.contains(section))
          table_cache[section] =
              html_table(bytes.at(source), model,
                         text(spec, "heading_section", 128), section);
        typed = table_value(table_cache.at(section), f);
      } else {
        fields(f,
               {"predicate", "label", "next_label", "value_type", "qualifier"});
        auto label = text(f, "label", 256), next = text(f, "next_label", 256);
        auto it = entries.find(label);
        if (it == entries.end() || it->second.second != next)
          invalid("missing or changed adjacent source field");
        if (type == "quarter_20yy")
          invalid("quarter mapping requires explicit scoped_tables.v1 profile");
        typed = source_value(it->second.first, type);
      }
      if (pred == "model_introduction") {
        if (type == "date")
          introduced = Json{{"from", typed}, {"through", typed}};
        else if (tables && type == "quarter_20yy")
          introduced = Json{{"from", typed.at("from")},
                            {"through", typed.at("through")}};
        else
          invalid(
              "model introduction requires date or explicit quarter mapping");
      }
      values.push_back(Json{{"predicate", pred},
                            {"value", typed},
                            {"qualifier", qualifier},
                            {"source_id", source}});
    }
    std::sort(values.begin(), values.end(), [](const auto &a, const auto &b) {
      return a.at("predicate") < b.at("predicate");
    });
    subjects.push_back(Json{{"id", id},
                            {"manufacturer", manufacturer},
                            {"model", model},
                            {"hardware_class", hardware},
                            {"introduced", introduced},
                            {"assertions", values}});
  }
  std::sort(subjects.begin(), subjects.end(), [](const auto &a, const auto &b) {
    return a.at("id") < b.at("id");
  });
  return seal(Json{{"protocol", "symphony.shv.catalogue.v1"},
                   {"sources", p.at("sources")},
                   {"subjects", subjects},
                   {"mapping", p.at("subjects")}});
}
void catalogue_replay(const engine::Request &r, const Json &c,
                      const std::string &root) {
  fields(c, {"protocol", "sources", "subjects", "mapping", "digest"});
  check_seal(c);
  if (text(c, "protocol") != "symphony.shv.catalogue.v1")
    invalid("unsupported catalogue protocol");
  const auto rebuilt = catalogue_build(r, Json{{"source_root", root},
                                               {"sources", c.at("sources")},
                                               {"subjects", c.at("mapping")}});
  if (rebuilt != c)
    invalid("catalogue differs from exact source and mapping replay");
}
Json project(const Json &c) {
  Json nodes = Json::array(), edges = Json::array();
  for (const auto &s : c.at("sources"))
    nodes.push_back(Json{{"id", "source:" + s.at("id").get<std::string>()},
                         {"labels", Json::array({"evidence_source"})},
                         {"properties", s}});
  for (const auto &subject : c.at("subjects")) {
    auto id = subject.at("id").get<std::string>();
    nodes.push_back(Json{{"id", "subject:" + id},
                         {"labels", Json::array({"hardware_subject"})},
                         {"properties", subject}});
    for (const auto &a : subject.at("assertions"))
      edges.push_back(
          Json{{"id",
                "assertion:" + id + ":" + a.at("predicate").get<std::string>()},
               {"from", "subject:" + id},
               {"to", "source:" + a.at("source_id").get<std::string>()},
               {"label", "supported_by_source"},
               {"properties", a}});
  }
  for (auto *rows : {&nodes, &edges})
    std::sort(rows->begin(), rows->end(), [](const auto &a, const auto &b) {
      return a.at("id") < b.at("id");
    });
  return seal(Json{{"protocol", "symphony.graph.exchange.v1"},
                   {"owner", Json{{"engine_id", engine_id},
                                  {"engine_version", version},
                                  {"artifact_protocol", c.at("protocol")},
                                  {"artifact_digest", c.at("digest")}}},
                   {"owner_artifact", c},
                   {"nodes", nodes},
                   {"edges", edges}});
}
} // namespace symphony::knowledge::shv
