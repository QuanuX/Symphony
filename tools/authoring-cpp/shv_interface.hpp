#pragma once
#include "authoring.hpp"

namespace symphony::authoring::shv {
inline const std::string declaration_fields =
    "protocol module_id engine_id namespace current_version releases "
    "contract_versions operations schemas companions embedded_dependencies";
inline const std::string operation_fields =
    "engine_operation_id operation_name availability feature_ids "
    "administrative_interactions administration_disposition input_protocol "
    "output_protocol mutability idempotency expected_state_required "
    "authorization_requirement recovery_operation_id direct_invocation "
    "thermal_path";
inline Json names(const Json &operations) {
  require(operations.is_array(), "operation list required");
  Json result = Json::array();
  for (const auto &op : operations)
    result.push_back(op.at("operation_name"));
  return result;
}
inline const Json &latest(const Json &descriptors) {
  require(descriptors.is_object() && !descriptors.empty(),
          "missing descriptor history");
  return descriptors.back();
}
inline void resources(const Json &d, const fs::path &root) {
  unique(d.at("schemas"), "schema");
  unique(d.at("companions"), "companion");
  for (const auto &name : d.at("companions"))
    local(root, name);
  for (const auto &name : d.at("schemas"))
    local_refs(read_json(local(root, name)));
}
inline void validate_v1(const Json &d, const fs::path &root) {
  exact(d, declaration_fields, "declaration");
  const auto lock = read_json(fs::path(SYMPHONY_AUTHORING_ROOT) /
                              "tools/shv-interface-codegen/history-lock.json");
  const auto module = string(d.at("module_id"));
  require(lock.contains(module), "unknown frozen owner");
  require(d.at("protocol") == "symphony.shv.owner-interface.v1",
          "declaration protocol");
  for (const auto *key : {"namespace", "engine_id", "embedded_dependencies"})
    require(same(d.at(key), lock.at(module).at(key)),
            "frozen owner identity changed");
  const auto history_path = local(
      root, "modules/" + module + "/tests/fixtures/interface-history.v1.json");
  require(digest_bytes(read_bytes(history_path)) ==
              "sha256:" + string(lock.at(module).at("history_digest")),
          "frozen history changed");
  const auto hist = read_json(history_path).at("descriptors");
  const auto current = word(d.at("current_version"), R"(0\.[1-9][0-9]*\.0-dev)",
                            "exact new release");
  require(!hist.contains(current), "exact new release required");
  auto expected_releases = hist;
  expected_releases[current] = nullptr;
  require(
      nlohmann::json(keys(d.at("releases"))).get<std::set<std::string>>() ==
          nlohmann::json(keys(expected_releases)).get<std::set<std::string>>(),
      "release map differs");
  const auto operation_names = names(d.at("operations"));
  unique(operation_names, "operation");
  for (auto it = hist.begin(); it != hist.end(); ++it) {
    require(
        same(d.at("releases").at(it.key()), names(it.value().at("operations"))),
        "historical release changed");
    Json selected = Json::array();
    for (const auto &op : d.at("operations"))
      if (contains(d.at("releases").at(it.key()), op.at("operation_name")))
        selected.push_back(op);
    require(same(selected, it.value().at("operations")),
            "historical operation change");
  }
  require(
      same(d.at("operations"), latest(hist).at("operations")) &&
          same(d.at("releases").at(current), operation_names) &&
          same(d.at("contract_versions"), latest(hist).at("contract_versions")),
      "metadata migration changes semantics");
  std::vector<std::string> schema_names;
  const auto prefix = fs::path("modules") / module / "schemas/v1";
  for (const auto &entry : fs::directory_iterator(root / prefix))
    if (entry.path().extension() == ".json")
      schema_names.push_back(
          (prefix / entry.path().filename()).generic_string());
  std::sort(schema_names.begin(), schema_names.end());
  require(!schema_names.empty() && same(d.at("schemas"), Json(schema_names)),
          "schema inventory differs");
  require(same(d.at("companions"),
               Json::array(
                   {"knowledge/shv/" + std::string(module == "shv-source-engine"
                                                       ? "SOURCES.md"
                                                       : "PUBLICATION.md")})),
          "companion inventory differs");
  resources(d, root);
}
inline Json load_v1(const fs::path &root, const std::string &owner) {
  const auto declaration =
      read_json(local(root, "modules/" + owner + "/OWNER-INTERFACE.json"));
  require(declaration.at("module_id") == owner,
          "selected owner differs from declaration");
  validate_v1(declaration, root);
  return declaration;
}
inline void validate_profile(const Json &d, const fs::path &root) {
  exact(d,
        "protocol module_id engine_id current_version embedded_kernel_version "
        "contract_versions releases operations schemas companions",
        "profile declaration");
  require(d.at("protocol") == "symphony.shv.profile-owner-interface.v1" &&
              d.at("module_id") == "shv-profile-engine" &&
              d.at("engine_id") == "symphony-shv-profile",
          "profile owner identity");
  require(d.at("current_version") == "0.3.0-dev" &&
              d.at("embedded_kernel_version") == "0.3.0-dev",
          "profile exact version");
  require(same(d.at("contract_versions"),
               Json::array({"knowledge/shv/PROFILES.md@v1"})),
          "profile contracts changed");
  exact(d.at("releases"), "0.1.0-dev 0.2.0-dev 0.3.0-dev", "profile releases");
  const auto history_path =
      root /
      "modules/shv-profile-engine/tests/fixtures/interface-history.v1.json";
  require(
      digest_bytes(read_bytes(history_path)) ==
          "sha256:"
          "9381ad350d97731ac211dd30284f92995fc92aad729a9ad06a8057d10444ff5e",
      "frozen history changed");
  const auto hist = read_json(history_path).at("descriptors");
  const auto operation_names = names(d.at("operations"));
  unique(operation_names, "operation");
  for (auto it = hist.begin(); it != hist.end(); ++it) {
    require(
        same(d.at("releases").at(it.key()), names(it.value().at("operations"))),
        "historical release changed");
    Json selected = Json::array();
    for (const auto &op : d.at("operations"))
      if (contains(d.at("releases").at(it.key()), op.at("operation_name")))
        selected.push_back(op);
    require(same(selected, it.value().at("operations")),
            "historical metadata changed");
  }
  require(same(d.at("releases").at("0.3.0-dev"), operation_names) &&
              same(operation_names, d.at("releases").at("0.2.0-dev")),
          "profile operation admission changed");
  require(same(d.at("schemas"),
               Json::array(
                   {"modules/shv-profile-engine/schemas/v1/profile.schema.json",
                    "modules/shv-profile-engine/schemas/v1/"
                    "profile.templates.json"})),
          "profile schema inventory differs");
  require(same(d.at("companions"), Json::array({"knowledge/shv/PROFILES.md"})),
          "profile companion inventory differs");
  resources(d, root);
}
inline std::pair<Json, Json> load(const fs::path &root,
                                  const fs::path &registration) {
  const auto r = read_json(registration);
  exact(
      r,
      "protocol declaration history history_digest go_title go_stem cpp_output "
      "go_output cmake_output embedded_compatibility_version embedded_macro",
      "registration");
  require(r.at("protocol") == "symphony.shv.interface-registration.v1",
          "registration shape");
  const auto d = read_json(local(root, r.at("declaration")));
  exact(d, declaration_fields, "declaration");
  require(d.at("protocol") == "symphony.shv.owner-interface.v2",
          "declaration shape");
  for (const auto *key : {"module_id", "engine_id"})
    word(d.at(key), "[a-z][a-z0-9-]*", "owner identifier");
  word(d.at("namespace"), R"([A-Za-z_]\w*(::[A-Za-z_]\w*)*)", "C++ namespace");
  for (const auto *key : {"go_title", "go_stem", "embedded_macro"})
    word(r.at(key), R"([A-Za-z_]\w*)", "output identifier");
  const auto hp = local(root, r.at("history"));
  require(digest_bytes(read_bytes(hp)) == string(r.at("history_digest")),
          "frozen history changed");
  const auto hist = read_json(hp).at("descriptors");
  require(hist.is_object(), "descriptor history object");
  const auto current = word(d.at("current_version"),
                            R"(\d+\.\d+\.\d+(?:-dev)?)", "exact new release");
  require(!hist.contains(current), "exact new release required");
  const auto &ops = d.at("operations");
  const auto operation_names = names(ops);
  unique(operation_names, "operation");
  require(!ops.empty(), "unique operations required");
  const auto contracts = strings(d.at("contract_versions"));
  for (const auto &contract : contracts)
    require(!contract.empty(), "contract references required");
  Json ids = Json::array();
  for (const auto &op : ops) {
    exact(op, operation_fields, "operation");
    ids.push_back(op.at("engine_operation_id"));
    require(op.at("expected_state_required").is_boolean(),
            "expected-state flag");
    for (const auto *key : {"feature_ids", "administrative_interactions"})
      static_cast<void>(strings(op.at(key)));
    for (auto it = op.begin(); it != op.end(); ++it)
      if (it.key() != "feature_ids" &&
          it.key() != "administrative_interactions" &&
          it.key() != "expected_state_required" &&
          it.key() != "recovery_operation_id")
        require(it.value().is_string(), "operation string");
    require(op.at("recovery_operation_id").is_null() ||
                op.at("recovery_operation_id").is_string(),
            "recovery identity");
  }
  unique(ids, "operation identity");
  auto expected_releases = hist;
  expected_releases[current] = nullptr;
  require(nlohmann::json(keys(d.at("releases"))).get<std::set<std::string>>() ==
                  nlohmann::json(keys(expected_releases))
                      .get<std::set<std::string>>() &&
              same(d.at("releases").at(current), operation_names),
          "release map differs");
  for (auto it = hist.begin(); it != hist.end(); ++it) {
    const auto &desc = it.value();
    require(desc.at("module_id") == d.at("module_id") &&
                desc.at("engine_id") == d.at("engine_id") &&
                desc.at("engine_version") == it.key(),
            "history owner differs");
    require(same(d.at("releases").at(it.key()), names(desc.at("operations"))),
            "historical release changed");
  }
  if (!hist.empty())
    require(same(ops, latest(hist).at("operations")) &&
                same(d.at("contract_versions"),
                     latest(hist).at("contract_versions")),
            "metadata migration changes semantics");
  const auto &embedded = r.at("embedded_compatibility_version");
  require(embedded.is_null() ||
              (embedded.is_string() && hist.contains(string(embedded))),
          "unproved embedded version");
  if (!embedded.is_null())
    require(same(hist.at(string(embedded)).at("operations"), ops),
            "embedded operations differ");
  require(d.at("embedded_dependencies").is_array(), "dependency list");
  for (const auto &dep : d.at("embedded_dependencies")) {
    exact(dep, "engine_id version", "dependency identity");
    for (const auto &value : dep)
      require(value.is_string() && !string(value).empty(),
              "dependency identity");
  }
  resources(d, root);
  std::set<std::string> sources{string(r.at("declaration")),
                                string(r.at("history"))};
  for (const auto &name : d.at("schemas"))
    sources.insert(string(name));
  for (const auto &name : d.at("companions"))
    sources.insert(string(name));
  std::set<std::string> outputs;
  for (const auto *key : {"cpp_output", "go_output", "cmake_output"}) {
    const auto name = string(r.at(key));
    require(outputs.insert(name).second && !sources.contains(name),
            "output collision");
    local(root, name, false);
  }
  return {r, d};
}
inline std::string cmake_resources(const Json &d, const std::string &prefix) {
  std::string out;
  for (const auto *key : {"schemas", "companions"}) {
    out += "set(" + prefix +
           (std::string(key) == "schemas" ? "_SCHEMA_FILES" : "_COMPANIONS") +
           "\n";
    for (const auto &path : d.at(key))
      out += " \"${SYMPHONY_REPOSITORY_ROOT}/" + string(path) + "\"\n";
    out += ")\n";
  }
  return out;
}
inline std::string go_tables(const Json &d, const std::string &outputs_name,
                             const std::string &admission_name) {
  std::string go = "var " + outputs_name + " = map[string]string{\n";
  for (const auto &op : d.at("operations"))
    go += quote(op.at("operation_name")) + ": " +
          quote(op.at("output_protocol")) + ",\n";
  go += "}\nvar " + admission_name + " = map[string]map[string]bool{\n";
  for (auto it = d.at("releases").begin(); it != d.at("releases").end(); ++it)
    go += quote(it.key()) + ": {" + quoted_join(it.value(), ", ", ": true") +
          "},\n";
  return go + "}\n";
}
inline std::string plain_version(const Json &d) {
  auto version = string(d.at("current_version"));
  if (version.ends_with("-dev"))
    version.resize(version.size() - 4);
  return version;
}
inline Outputs owner_outputs(const Json &r, const Json &d) {
  std::string version =
      "inline constexpr auto version = " + quote(d.at("current_version")) +
      ";\n";
  if (!r.at("embedded_compatibility_version").is_null())
    version = "#ifdef " + string(r.at("embedded_macro")) +
              "\ninline constexpr auto version = " +
              quote(r.at("embedded_compatibility_version")) + ";\n#else\n" +
              version + "#endif\n";
  std::string cpp =
      "// Generated by owner_codegen.cpp; do not edit.\n#pragma once\n#include "
      "\"symphony/knowledge/engine/operation.hpp\"\nnamespace " +
      string(d.at("namespace")) +
      " {\nnamespace engine = symphony::knowledge::engine;\n" + version +
      "inline std::vector<engine::OperationSpec> interface_operations() { "
      "return {\n";
  for (const auto &op : d.at("operations")) {
    std::vector<std::string> values;
    for (const auto *key :
         {"engine_operation_id", "operation_name", "availability"})
      values.push_back(quote(op.at(key)));
    values.push_back("false");
    values.push_back("true");
    for (const auto *key : {"feature_ids", "administrative_interactions"})
      values.push_back("{" + quoted_join(op.at(key)) + "}");
    for (const auto *key :
         {"administration_disposition", "input_protocol", "output_protocol",
          "mutability", "idempotency", "expected_state_required",
          "authorization_requirement"})
      values.push_back(quote(op.at(key)));
    values.push_back(quote(op.at("recovery_operation_id").is_null()
                               ? Json("")
                               : op.at("recovery_operation_id")));
    values.push_back(quote(op.at("direct_invocation")));
    values.push_back(quote(op.at("thermal_path")));
    cpp += "  {" + join(values, ", ") + "},\n";
  }
  cpp += "}; }\n}\n";
  const auto title = string(r.at("go_title"));
  const auto module = string(d.at("module_id"));
  const std::string go =
      "// Code generated by owner_codegen.cpp for " + module +
      "; DO NOT EDIT.\npackage knowledgeengine\nconst SHV" + title +
      "InterfaceVersion = " + quote(d.at("current_version")) + "\nconst shv" +
      title + "InterfaceDigest = " + quote(digest(d)) + "\n" +
      go_tables(d, "shv" + title + "InterfaceOutputs",
                "shv" + title + "InterfaceAdmission");
  const std::string cmake = "# Generated by owner_codegen.cpp for " + module +
                            "; do not edit.\nset(SHV_OWNER_VERSION \"" +
                            plain_version(d) + "\")\n" +
                            cmake_resources(d, "SHV_OWNER");
  return {{string(r.at("cpp_output")), cpp},
          {string(r.at("go_output")), gofmt(go)},
          {string(r.at("cmake_output")), cmake}};
}
inline Outputs v1_outputs(const Json &d) {
  const auto module = string(d.at("module_id"));
  auto short_name = module.substr(4);
  if (short_name.ends_with("-engine"))
    short_name.resize(short_name.size() - 7);
  auto title = short_name;
  title[0] =
      static_cast<char>(std::toupper(static_cast<unsigned char>(title[0])));
  std::string cpp =
      "// Generated by tools/shv-interface-codegen/generate.cpp; do not "
      "edit.\n#pragma once\n#include <array>\n#include <vector>\n#include "
      "<string>\nnamespace " +
      string(d.at("namespace")) +
      " {\ninline constexpr auto version = " + quote(d.at("current_version")) +
      ";\nstruct InterfaceOperation { const char *id; const char *name; const "
      "char *input; const char *output; std::vector<std::string> interactions; "
      "bool expected_state; };\ninline const std::array<InterfaceOperation, " +
      std::to_string(d.at("operations").size()) +
      "> interface_operations = {{\n";
  for (const auto &op : d.at("operations")) {
    std::vector<std::string> values;
    for (const auto *key : {"engine_operation_id", "operation_name",
                            "input_protocol", "output_protocol"})
      values.push_back(quote(op.at(key)));
    cpp += "  {" + join(values, ", ") + ", {" +
           quoted_join(op.at("administrative_interactions"), ", ") + "}, " +
           quote(op.at("expected_state_required")) + "},\n";
  }
  cpp += "}};\n}\n";
  const auto go =
      "// Code generated by tools/shv-interface-codegen/generate.cpp; DO NOT "
      "EDIT.\npackage knowledgeengine\nconst SHV" +
      title + "InterfaceVersion = " + quote(d.at("current_version")) +
      "\nconst shv" + title + "InterfaceDigest = " + quote(digest(d)) + "\n" +
      go_tables(d, "shv" + title + "Outputs", "shv" + title + "Admission");
  const auto cmake = "# Generated by tools/shv-interface-codegen/generate.cpp; "
                     "do not edit.\nset(SHV_OWNER_VERSION \"" +
                     plain_version(d) + "\")\n" +
                     cmake_resources(d, "SHV_OWNER");
  return {{"modules/" + module + "/src/interface.generated.hpp", cpp},
          {"tools/qxctl/internal/knowledgeengine/shv_" + short_name +
               "_interface_generated.go",
           gofmt(go)},
          {"cmake/Shv" + title + "Interface.generated.cmake", cmake}};
}
inline Outputs profile_outputs(const Json &d) {
  std::string cpp =
      "// Generated by tools/generate_interface.cpp; do not edit.\n#pragma "
      "once\n#include <array>\nnamespace symphony::knowledge::shv_profile "
      "{\ninline constexpr auto version = " +
      quote(d.at("current_version")) +
      ";\ninline constexpr auto embedded_kernel_version = " +
      quote(d.at("embedded_kernel_version")) +
      ";\nstruct InterfaceOperation { const char *name; const char *input; "
      "const char *output; const char *interaction; };\ninline constexpr "
      "std::array<InterfaceOperation, " +
      std::to_string(d.at("operations").size()) +
      "> interface_operations = {{\n";
  for (const auto &op : d.at("operations")) {
    std::vector<std::string> values;
    for (const auto *key :
         {"operation_name", "input_protocol", "output_protocol"})
      values.push_back(quote(op.at(key)));
    values.push_back(quote(op.at("administrative_interactions").at(0)));
    cpp += "  {" + join(values, ", ") + "},\n";
  }
  cpp += "}};\n}\n";
  const auto go =
      "// Code generated by shv-profile-engine/tools/generate_interface.cpp; "
      "DO NOT EDIT.\npackage knowledgeengine\n\nconst "
      "SHVProfileInterfaceVersion = " +
      quote(d.at("current_version")) +
      "\nconst shvProfileInterfaceDigest = " + quote(digest(d)) + "\n" +
      go_tables(d, "shvProfileOutputs", "shvProfileAdmission");
  const auto cmake = "# Generated by tools/generate_interface.cpp; do not "
                     "edit.\nset(SHV_PROFILE_VERSION \"" +
                     plain_version(d) + "\")\n" +
                     cmake_resources(d, "SHV_PROFILE");
  return {{"modules/shv-profile-engine/src/interface.generated.hpp", cpp},
          {"tools/qxctl/internal/knowledgeengine/"
           "shv_profile_interface_generated.go",
           gofmt(go)},
          {"cmake/ShvProfileInterface.generated.cmake", cmake}};
}
inline void owner_main(const Arguments &args) {
  require(!args.registration.empty(), "--registration is required");
  const auto [r, d] = load(args.root, fs::absolute(args.registration));
  emit(args.root, owner_outputs(r, d), args.check, true);
  std::cout << string(d.at("module_id"))
            << (args.check ? " checked\n" : " generated\n");
}
} // namespace symphony::authoring::shv
