#pragma once
#include "authoring.hpp"
namespace symphony::authoring::scv {
inline const std::string manifest_path = "knowledge/scv/OWNER-INTERFACE.json";
inline const std::string history_path =
    "modules/scv-engine/tests/fixtures/interface-history.v1.json";
inline const std::string cpp_path =
    "modules/scv-engine/src/interface.generated.inc";
inline const std::string go_path =
    "tools/qxctl/internal/knowledgeengine/scv_interface_generated.go";
inline const std::string cmake_path = "cmake/ScvInterface.generated.cmake";
inline Json versions(const Json &m) {
  Json out = Json::array();
  for (const auto &release : m.at("releases"))
    out.push_back(release.at("version"));
  return out;
}
inline void validate(const Json &m) {
  exact(m,
        "protocol current_release domains releases operations schema_catalog",
        "manifest");
  require(m.at("protocol") == "symphony.scv.owner-interface.v1",
          "unsupported owner interface protocol");
  const auto &domains = m.at("domains");
  require(domains.is_array() && !domains.empty() && domains.size() <= 64,
          "invalid domain inventory");
  Json domain_names = Json::array();
  for (const auto &d : domains) {
    exact(d, "name introduced_in", "domain");
    domain_names.push_back(word(
        d.at("name"), "(?:scv|schv|scev|(?:schv|scev)-[a-z0-9][a-z0-9-]{0,63})",
        "domain"));
  }
  unique(domain_names, "domain");
  const auto &releases = m.at("releases");
  require(releases.is_array() && !releases.empty() && releases.size() <= 64,
          "invalid release inventory");
  for (const auto &r : releases) {
    exact(r, "version surfaces companions contract_versions", "release");
    word(r.at("version"), R"([0-9]+\.[0-9]+\.[0-9]+-dev)", "exact release");
    for (const auto *field : {"surfaces", "companions", "contract_versions"})
      unique(r.at(field), field);
    for (const auto &surface : r.at("surfaces"))
      require(contains(Json::array({"corpus", "workflow", "artifact", "schema",
                                    "interface", "composition_workflow",
                                    "composition_bundle_workflow",
                                    "composition_bundle_obligation_link"}),
                       surface),
              "unknown adapter surface");
    require(!r.at("companions").empty(), "release lacks owner companions");
    Json contracts = Json::array({"knowledge/SPEC.md@v1"});
    for (const auto &companion : r.at("companions")) {
      word(companion, R"([A-Z][A-Z0-9-]*\.md)", "owner companion");
      contracts.push_back("knowledge/scv/" + string(companion) + "@v1");
    }
    require(same(r.at("contract_versions"), contracts),
            "contract inventory differs from ordered owner companions");
  }
  const auto v = versions(m);
  unique(v, "release");
  for (const auto &domain : domains)
    require(contains(v, domain.at("introduced_in")),
            "domain names an undeclared release");
  require(m.at("current_release") == v.back(),
          "current release must select the last declared exact release");
  const auto &ops = m.at("operations");
  require(ops.is_array() && !ops.empty() && ops.size() <= 128,
          "invalid operation inventory");
  Json names = Json::array(), kinds = Json::array();
  for (const auto &op : ops) {
    exact(op,
          "name introduced_in input_protocol output_protocol interactions "
          "mutability expected_state handler artifact",
          "operation");
    const auto name =
        word(op.at("name"), "[a-z][a-z0-9]*(?:_[a-z0-9]+)*", "operation name");
    names.push_back(name);
    require(contains(v, op.at("introduced_in")),
            "operation names an undeclared release");
    for (const auto *field : {"input_protocol", "output_protocol"})
      word(op.at(field), R"(symphony\.[a-z0-9.-]+\.v[1-9][0-9]*)",
           "operation protocol");
    auto protocol_name = name;
    std::replace(protocol_name.begin(), protocol_name.end(), '_', '-');
    require(op.at("input_protocol") ==
                "symphony.scv." + protocol_name + "-input.v1",
            "operation input protocol identity mismatch");
    unique(op.at("interactions"), "interaction");
    require(!op.at("interactions").empty(), "missing operation interactions");
    for (const auto &interaction : op.at("interactions"))
      require(contains(Json::array({"inspect", "discover", "propose", "apply",
                                    "recover", "invoke", "query", "validate"}),
                       interaction),
              "unknown operation interaction");
    require(
        contains(Json::array({"read_only", "proposal_only", "evidence_only"}),
                 op.at("mutability")) &&
            op.at("expected_state").is_boolean(),
        "operation broadens mutation authority");
    require(
        contains(Json::array({"inspect", "handle_source", "handle_knowledge",
                              "handle_corpus", "handle_interpretation",
                              "handle_coverage", "handle_pack",
                              "handle_composition", "handle_bundle"}),
                 op.at("handler")),
        "unknown compiled handler");
    const auto &artifact = op.at("artifact");
    if (!artifact.is_null()) {
      exact(artifact, "kind introduced_in", "artifact admission");
      kinds.push_back(
          word(artifact.at("kind"), "[a-z][a-z0-9_]*", "artifact kind"));
      require(contains(v, artifact.at("introduced_in")) &&
                  ordinal(v, artifact.at("introduced_in")) >=
                      ordinal(v, op.at("introduced_in")),
              "artifact admission precedes native operation");
      require(contains(releases.at(ordinal(v, artifact.at("introduced_in")))
                           .at("surfaces"),
                       "artifact"),
              "artifact admitted without artifact adapter");
    }
  }
  unique(names, "operation");
  unique(kinds, "artifact kind");
  require(names.at(0) == "inspect" && ops.at(0).at("handler") == "inspect",
          "inspect must remain the descriptor operation");
  require(m.at("schema_catalog") ==
              "knowledge/scv/schemas/v1/schema-catalog.json",
          "schema catalog must use the owned SCV resource location");
}
inline Json projection(const Json &m, const Json &version) {
  const auto v = versions(m);
  const auto index = ordinal(v, version);
  const auto &release = m.at("releases").at(index);
  Json operations = Json::array(), domains = Json::array();
  for (const auto &op : m.at("operations"))
    if (ordinal(v, op.at("introduced_in")) <= index) {
      auto projected = op;
      projected.erase("introduced_in");
      projected.erase("artifact");
      const auto &artifact = op.at("artifact");
      projected["artifact_kind"] =
          !artifact.is_null() &&
                  ordinal(v, artifact.at("introduced_in")) <= index
              ? artifact.at("kind")
              : Json(nullptr);
      operations.push_back(projected);
    }
  for (const auto &domain : m.at("domains"))
    if (ordinal(v, domain.at("introduced_in")) <= index)
      domains.push_back(domain.at("name"));
  return {{"version", version},
          {"domains", domains},
          {"surfaces", release.at("surfaces")},
          {"companions", release.at("companions")},
          {"contract_versions", release.at("contract_versions")},
          {"operations", operations}};
}
inline Json release_manifest(const Json &m, const Json &version) {
  const auto v = versions(m);
  const auto index = ordinal(v, version);
  auto result = m;
  result["current_release"] = version;
  result["releases"] = Json::array();
  result["operations"] = Json::array();
  result["domains"] = Json::array();
  for (std::size_t i = 0; i <= index; ++i)
    result["releases"].push_back(m.at("releases").at(i));
  for (auto op : m.at("operations"))
    if (ordinal(v, op.at("introduced_in")) <= index) {
      if (!op.at("artifact").is_null() &&
          ordinal(v, op.at("artifact").at("introduced_in")) > index)
        op["artifact"] = nullptr;
      result["operations"].push_back(op);
    }
  for (const auto &domain : m.at("domains"))
    if (ordinal(v, domain.at("introduced_in")) <= index)
      result["domains"].push_back(domain);
  return result;
}
inline void check_history(const Json &m, const fs::path &root) {
  const auto history = read_json(root / history_path, 1048576);
  exact(history, "protocol releases", "frozen history");
  require(history.at("protocol") == "symphony.scv.owner-interface-history.v1",
          "invalid frozen interface history");
  require(history.at("releases").is_array(),
          "frozen history releases must be an array");
  for (const auto &previous : history.at("releases"))
    require(same(projection(m, previous.at("version")), previous),
            "frozen interface changed: " + string(previous.at("version")));
}
inline std::string schema_name(const Json &value) {
  return word(value, R"([a-z0-9-]+\.schema\.json)",
              "schema reference escapes owned schema directory");
}
inline Json schema_inventory(const Json &m, const fs::path &root) {
  const auto catalog_path = root / string(m.at("schema_catalog"));
  const auto catalog = read_json(catalog_path, 1048576);
  require(catalog.is_object() &&
              catalog.value("protocol", "") ==
                  "symphony.scv.schema-catalog.v1" &&
              catalog.at("engine_version") == m.at("current_release"),
          "catalog does not identify the current exact release");
  const auto &entries = catalog.at("entries");
  require(entries.is_array() && !entries.empty(), "empty schema catalog");
  std::map<std::string, Json> indexed, documents;
  std::set<std::string> files{catalog_path.filename().string()};
  for (const auto &entry : entries) {
    require(entry.is_object(), "invalid catalog entry");
    static_cast<void>(strings(entry.at("operations")));
    const auto protocol = string(entry.at("protocol"));
    require(indexed.emplace(protocol, entry).second,
            "duplicate or missing catalog protocol");
    files.insert(schema_name(entry.at("file")));
  }
  for (const auto &op : projection(m, m.at("current_release")).at("operations"))
    for (const auto &[field, kind] :
         std::vector<std::pair<std::string, std::string>>{
             {"input_protocol", "input"}, {"output_protocol", "output"}}) {
      const auto found = indexed.find(string(op.at(field)));
      require(found != indexed.end() && found->second.at("kind") == kind &&
                  contains(found->second.at("operations"), op.at("name")),
              "catalog does not bind " + string(op.at("name")) + " " + kind +
                  " protocol");
    }
  const auto document = [&](const std::string &name) -> const Json & {
    if (!documents.contains(name))
      documents[name] = read_json(
          local(catalog_path.parent_path(), schema_name(name)), 1048576);
    return documents.at(name);
  };
  const auto resolve = [&](const std::string &name, const Json &fragment) {
    const auto &doc = document(name);
    require(fragment.is_string(), "invalid schema fragment");
    const auto pointer = string(fragment);
    if (pointer.empty() || pointer == "#")
      return;
    require(pointer.starts_with("#/"), "invalid schema fragment");
    const Json *value = &doc;
    std::size_t start = 2;
    while (true) {
      const auto end = pointer.find('/', start);
      const auto token = pointer.substr(start, end - start);
      std::string decoded;
      for (std::size_t i = 0; i < token.size(); ++i) {
        if (token[i] != '~') {
          decoded += token[i];
          continue;
        }
        require(i + 1 < token.size() &&
                    (token[i + 1] == '0' || token[i + 1] == '1'),
                "invalid schema pointer escape");
        decoded += token[++i] == '0' ? '~' : '/';
      }
      require(value->is_object() && value->contains(decoded),
              "unresolvable schema fragment");
      value = &value->at(decoded);
      if (end == std::string::npos)
        break;
      start = end + 1;
    }
    require(value->is_object() || value->is_boolean(),
            "schema fragment does not select a schema");
  };
  for (const auto &entry : entries)
    resolve(string(entry.at("file")), entry.value("fragment", Json(nullptr)));
  std::vector<std::string> pending;
  for (const auto &file : files)
    if (file != catalog_path.filename().string())
      pending.push_back(file);
  std::set<std::string> seen;
  while (!pending.empty()) {
    const auto name = pending.back();
    pending.pop_back();
    if (!seen.insert(name).second)
      continue;
    require(seen.size() <= 64, "schema reference closure exceeds 64 files");
    const auto &doc = document(name);
    std::function<void(const Json &)> walk = [&](const Json &value) {
      if (value.is_object()) {
        if (value.contains("$ref")) {
          const auto ref = string(value.at("$ref"));
          const auto hash = ref.find('#');
          const auto file_part = ref.substr(0, hash);
          const auto target = file_part.empty() ? name : schema_name(file_part);
          resolve(target, hash == std::string::npos ? "" : ref.substr(hash));
          if (!file_part.empty())
            pending.push_back(target);
        }
        for (const auto &child : value)
          walk(child);
      } else if (value.is_array())
        for (const auto &child : value)
          walk(child);
    };
    walk(doc);
  }
  files.insert(seen.begin(), seen.end());
  std::set<std::string> available;
  for (const auto &entry : fs::directory_iterator(catalog_path.parent_path()))
    if (entry.path().filename().string().ends_with(".schema.json"))
      available.insert(entry.path().filename().string());
  auto expected = files;
  expected.erase(catalog_path.filename().string());
  require(available == expected,
          "schema inventory contains an unreferenced or absent schema file");
  return Json(std::vector<std::string>(files.begin(), files.end()));
}
inline Outputs render(const Json &m, const fs::path &root,
                      bool metadata_only = false) {
  validate(m);
  check_history(m, root);
  const auto current = projection(m, m.at("current_release"));
  const auto fingerprint = digest(m, false);
  const std::string banner =
      "Generated by modules/scv-engine/tools/generate_interface.cpp; DO NOT "
      "EDIT.";
  std::string cpp = "// " + banner +
                    "\n// Owner interface SHA-256: " + fingerprint +
                    "\nconst std::vector<Operation>& registry() {\n    static "
                    "const std::vector<Operation> operations{\n";
  for (const auto &op : current.at("operations"))
    cpp += "        {" + quote(op.at("name")) + ", " +
           quote(op.at("input_protocol")) + ", " +
           quote(op.at("output_protocol")) + ", {" +
           quoted_join(op.at("interactions"), ", ") + "}, " +
           quote(op.at("mutability")) + ", " + quote(op.at("expected_state")) +
           ", " + string(op.at("handler")) + "},\n";
  cpp += "    };\n    return operations;\n}\nJson "
         "interface_contract_versions() {\n    return Json::array({" +
         quoted_join(current.at("contract_versions"), ", ") + "});\n}\n";
  std::string go =
      "// Code generated by modules/scv-engine/tools/generate_interface.cpp; "
      "DO NOT EDIT.\n// Owner interface SHA-256: " +
      fingerprint + R"(
package knowledgeengine

type SCVOperationInterface struct {
InputProtocol string
OutputProtocol string
Interactions []string
Mutability string
ExpectedState bool
}
type scvReleaseInterface struct {
operations map[string]bool
artifacts map[string]bool
surfaces map[string]bool
domains map[string]bool
companions []string
}
)";
  go += "var scvInterfaceDomains = []string{" +
        quoted_join(current.at("domains")) +
        "}\nvar scvInterfaceOperations = map[string]SCVOperationInterface{\n";
  for (const auto &op : m.at("operations"))
    go += quote(op.at("name")) +
          ": {InputProtocol:" + quote(op.at("input_protocol")) +
          ",OutputProtocol:" + quote(op.at("output_protocol")) +
          ",Interactions:[]string{" + quoted_join(op.at("interactions")) +
          "},Mutability:" + quote(op.at("mutability")) +
          ",ExpectedState:" + quote(op.at("expected_state")) + "},\n";
  go += "}\nvar scvInterfaceArtifacts = map[string]struct{kind, minimumRelease "
        "string}{\n";
  for (const auto &op : m.at("operations"))
    if (!op.at("artifact").is_null())
      go += quote(op.at("name")) + ": {" + quote(op.at("artifact").at("kind")) +
            "," + quote(op.at("artifact").at("introduced_in")) + "},\n";
  go += "}\nvar scvInterfaceReleases = map[string]scvReleaseInterface{\n";
  for (const auto &release : m.at("releases")) {
    const auto p = projection(m, release.at("version"));
    Json operations = Json::array(), artifacts = Json::array();
    for (const auto &op : p.at("operations")) {
      operations.push_back(op.at("name"));
      if (!op.at("artifact_kind").is_null())
        artifacts.push_back(op.at("name"));
    }
    go +=
        quote(release.at("version")) + ": {operations:map[string]bool{" +
        quoted_join(operations, ",", ":true") + "},artifacts:map[string]bool{" +
        quoted_join(artifacts, ",", ":true") + "},surfaces:map[string]bool{" +
        quoted_join(release.at("surfaces"), ",", ":true") +
        "},domains:map[string]bool{" +
        quoted_join(p.at("domains"), ",", ":true") + "},companions:[]string{" +
        quoted_join(release.at("companions")) + "}},\n";
  }
  go += R"(}

// Interface admission is exact and generated; validation behavior remains independent.
func SCVDomains() []string { return append([]string(nil), scvInterfaceDomains...) }
func SCVOperationMetadata(operation string) (SCVOperationInterface, bool) {
 value,ok:=scvInterfaceOperations[operation]; value.Interactions=append([]string(nil),value.Interactions...); return value,ok
}
func SCVResultProtocol(operation string) (string,bool) { value,ok:=scvInterfaceOperations[operation];return value.OutputProtocol,ok }
func SCVOperationSupported(version,operation string) bool { return scvInterfaceReleases[version].operations[operation] }
func SCVSupports(version,surface string) bool { return scvInterfaceReleases[version].surfaces[surface] }
func SCVDomainSupported(version,domain string) bool { return scvInterfaceReleases[version].domains[domain] }
func SCVArtifactKind(operation string) string { return scvInterfaceArtifacts[operation].kind }
func SCVArtifactMinimumRelease(operation string) string { return scvInterfaceArtifacts[operation].minimumRelease }
func SCVArtifactSupported(version,operation string) bool { return scvInterfaceReleases[version].artifacts[operation] }
func SCVOperationInteraction(operation string) string { value:=scvInterfaceOperations[operation];if len(value.Interactions)==0{return ""};return value.Interactions[0] }
func scvInterfaceOperationCount(version string) int { return len(scvInterfaceReleases[version].operations) }
func scvInterfaceCompanions(version string) []string { return append([]string(nil),scvInterfaceReleases[version].companions...) }

)";
  Json artifacts = Json::array();
  for (const auto &op : m.at("operations"))
    if (!op.at("artifact").is_null())
      artifacts.push_back(op.at("name"));
  go += "func SCVArtifactOperations() []string { return []string{" +
        quoted_join(artifacts) +
        "} }\nvar scvInterfaceDefinitionDigests = map[string]string{\n";
  for (const auto &release : m.at("releases"))
    if (contains(release.at("surfaces"), "interface"))
      go += quote(release.at("version")) + ":" +
            quote(digest(release_manifest(m, release.at("version")), false)) +
            ",\n";
  go += "}";
  Outputs output{{cpp_path, cpp}, {go_path, gofmt(go)}};
  if (!metadata_only) {
    const auto files = schema_inventory(m, root);
    std::string cmake =
        "# " + banner + "\n# Owner interface SHA-256: " + fingerprint +
        "\nset(SCV_INTERFACE_VERSION " + quote(m.at("current_release")) +
        ")\nset(SCV_INTERFACE_DOMAINS " +
        join(strings(current.at("domains")), " ") +
        ")\nset(SCV_INTERFACE_SCHEMA_FILES\n";
    for (const auto &name : files)
      cmake += "    \"${SYMPHONY_REPOSITORY_ROOT}/knowledge/scv/schemas/v1/" +
               string(name) + "\"\n";
    cmake += ")\nset(SCV_INTERFACE_COMPANION_FILES\n";
    for (const auto &name : current.at("companions"))
      cmake += "    \"${SYMPHONY_REPOSITORY_ROOT}/knowledge/scv/" +
               string(name) + "\"\n";
    cmake += ")\nset(SCV_INTERFACE_MANIFEST \"${SYMPHONY_REPOSITORY_ROOT}/" +
             manifest_path + "\")\n";
    output.emplace_back(cmake_path, cmake);
  }
  return output;
}
} // namespace symphony::authoring::scv
