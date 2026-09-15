#pragma once
#include "authoring.hpp"
namespace symphony::authoring::schema {
inline Json obj(Json properties, bool refresh_order = false) {
  if (refresh_order)
    return {{"type", "object"},
            {"additionalProperties", false},
            {"required", keys(properties)},
            {"properties", properties}};
  return {{"type", "object"},
          {"properties", properties},
          {"required", keys(properties)},
          {"additionalProperties", false}};
}
inline Json arr(const Json &items, int maximum) {
  return {{"type", "array"}, {"items", items}, {"maxItems", maximum}};
}
inline Json source_arr(const Json &items, int maximum, int minimum = 0,
                       bool unique_items = false) {
  auto value = arr(items, maximum);
  value["minItems"] = minimum;
  if (unique_items)
    value["uniqueItems"] = true;
  return value;
}
inline Json txt(int maximum = 4096) {
  return {{"type", "string"}, {"minLength", 1}, {"maxLength", maximum}};
}
inline Json ref(const std::string &name) {
  return {{"$ref", "#/$defs/" + name}};
}
inline Json nullable(const Json &value) {
  return {{"anyOf", Json::array({value, {{"type", "null"}}})}};
}
inline Json constant(const Json &value) { return {{"const", value}}; }
inline Json enumeration(std::initializer_list<Json> values) {
  Json list = Json::array();
  for (const auto &value : values)
    list.push_back(value);
  return {{"enum", list}};
}
inline Json merge(Json left, const Json &right) {
  for (auto it = right.begin(); it != right.end(); ++it)
    left[it.key()] = it.value();
  return left;
}
inline Json id() {
  return {{"type", "string"}, {"pattern", "^[A-Za-z0-9._-]{1,128}$"}};
}
inline Json digest_type() {
  return {{"type", "string"}, {"pattern", "^sha256:[0-9a-f]{64}$"}};
}
inline Json date() {
  return {{"type", "string"}, {"pattern", "^[0-9]{4}-[0-9]{2}-[0-9]{2}$"}};
}
inline Json integer(std::int64_t minimum, std::int64_t maximum) {
  return {{"type", "integer"}, {"minimum", minimum}, {"maximum", maximum}};
}
inline Json sealed(const std::string &protocol, const Json &fields) {
  return obj(merge(merge({{"protocol", constant(protocol)}}, fields),
                   {{"digest", digest_type()}}));
}
inline Json unique_array(const Json &items, int maximum, int minimum = -1) {
  auto out = arr(items, maximum);
  if (minimum >= 0)
    out["minItems"] = minimum;
  out["uniqueItems"] = true;
  return out;
}
inline Json counts(std::initializer_list<std::string> names, int maximum) {
  Json out = Json::object();
  for (const auto &name : names)
    out[name] = integer(0, maximum);
  return obj(out);
}
inline Json document(const std::string &identifier, const std::string &comment,
                     const Json &definitions) {
  return {{"$schema", "https://json-schema.org/draft/2020-12/schema"},
          {"$id", identifier},
          {"$comment", comment},
          {"$defs", definitions}};
}
inline Outputs
json_outputs(const std::vector<std::pair<std::string, Json>> &values) {
  Outputs out;
  for (const auto &[path, value] : values)
    out.emplace_back(path, value.dump(2, ' ', true) + "\n");
  return out;
}
inline Outputs kernel(const fs::path &root) {
  Json d = Json::object();
  d["Value"] = {
      {"oneOf",
       Json::array({txt(), integer(-9007199254740991LL, 9007199254740991LL),
                    unique_array(txt(128), 32)})}};
  d["DateInterval"] = obj({{"from", date()}, {"through", date()}});
  const Json base{{"id", id()},
                  {"manufacturer", txt(256)},
                  {"model", txt(256)},
                  {"hardware_class", id()},
                  {"introduced", nullable(ref("DateInterval"))}};
  d["SubjectSummary"] = obj(base);
  d["Source"] = obj({{"id", id()},
                     {"path", txt()},
                     {"bytes", integer(0, 1048576)},
                     {"digest", digest_type()},
                     {"format", enumeration({"html", "opaque"})}});
  d["MappingField"] =
      obj({{"predicate", id()},
           {"label", txt(256)},
           {"next_label", txt(256)},
           {"value_type", enumeration({"string", "integer", "date", "tokens"})},
           {"qualifier", txt(256)}});
  const auto section =
      merge(txt(128), {{"pattern", "^[a-z][a-z0-9-]*#[^#]+$"}});
  d["SubjectSpec"] = obj({{"id", id()},
                          {"manufacturer", txt(256)},
                          {"model", txt(256)},
                          {"hardware_class", id()},
                          {"source_id", id()},
                          {"heading_section", section},
                          {"field_section", section},
                          {"fields", arr(ref("MappingField"), 16)}});
  d["Assertion"] = obj({{"predicate", id()},
                        {"value", ref("Value")},
                        {"qualifier", txt(256)},
                        {"source_id", id()}});
  d["Subject"] = obj(merge(base, {{"assertions", arr(ref("Assertion"), 16)}}));
  d["Selector"] = {
      {"oneOf",
       Json::array(
           {obj({{"op", constant("all")}}),
            obj({{"op", constant("date")},
                 {"basis", constant("model_introduction")},
                 {"from", date()},
                 {"through", date()}}),
            obj({{"op", enumeration({"ids", "class"})},
                 {"values", unique_array(id(), 128)}}),
            obj({{"op", constant("manufacturer")},
                 {"values", unique_array(txt(256), 128)}}),
            obj({{"op", enumeration({"and", "or"})},
                 {"args", merge(arr(ref("Selector"), 32), {{"minItems", 1}})}}),
            obj({{"op", constant("not")}, {"arg", ref("Selector")}})})}};
  d["CoverageProfile"] =
      sealed("symphony.shv.coverage-profile.v1",
             {{"as_of", date()}, {"selector", ref("Selector")}});
  d["InspectInput"] = obj(Json::object());
  d["CoverageDefaultInput"] = obj({{"as_of", date()}});
  d["CoveragePlanInput"] = obj({{"profile", ref("CoverageProfile")},
                                {"subjects", arr(ref("SubjectSummary"), 128)}});
  d["CoverageResult"] = sealed(
      "symphony.shv.coverage-result.v1",
      {{"profile", ref("CoverageProfile")},
       {"subjects", arr(ref("SubjectSummary"), 128)},
       {"decisions", arr(obj({{"subject_id", id()},
                              {"status", enumeration({"included", "excluded",
                                                      "unresolved"})}}),
                         128)},
       {"counts", counts({"included", "excluded", "unresolved"}, 128)}});
  d["CatalogueBuildInput"] = obj({{"source_root", txt()},
                                  {"sources", arr(ref("Source"), 8)},
                                  {"subjects", arr(ref("SubjectSpec"), 32)}});
  d["Catalogue"] = sealed("symphony.shv.catalogue.v1",
                          {{"sources", arr(ref("Source"), 8)},
                           {"subjects", arr(ref("Subject"), 32)},
                           {"mapping", arr(ref("SubjectSpec"), 32)}});
  const auto ids = unique_array(id(), 128);
  d["CatalogueQueryInput"] = obj({{"source_root", txt()},
                                  {"catalogue", ref("Catalogue")},
                                  {"subject_ids", ids}});
  d["QueryResult"] = sealed("symphony.shv.query-result.v1",
                            {{"catalogue_digest", digest_type()},
                             {"subject_ids", ids},
                             {"subjects", arr(ref("Subject"), 32)},
                             {"missing_subject_ids", ids}});
  d["ScalarValue"] = d["Value"];
  d["Requirement"] = obj({{"id", id()},
                          {"predicate", id()},
                          {"operator", enumeration({"eq", "contains", "gte"})},
                          {"value", ref("ScalarValue")},
                          {"qualifier", txt(256)}});
  d["EvaluateInput"] = obj({{"source_root", txt()},
                            {"catalogue", ref("Catalogue")},
                            {"subject_ids", ids},
                            {"requirements", arr(ref("Requirement"), 32)}});
  d["Evaluation"] = sealed(
      "symphony.shv.evaluation.v1",
      {{"catalogue_digest", digest_type()},
       {"subject_ids", ids},
       {"requirements", arr(ref("Requirement"), 32)},
       {"findings",
        arr(obj({{"subject_id", id()},
                 {"requirement_id", id()},
                 {"status",
                  enumeration({"supported", "contradicted", "unresolved"})},
                 {"evidence", arr(ref("Assertion"), 1)}}),
            1024)},
       {"missing_subject_ids", ids}});
  const auto graph =
      read_json(
          root /
          "modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json")
          .at("$defs");
  for (const auto *name : {"Graph", "Node", "Edge"})
    d[name] = graph.at(name);
  d["GraphProjectInput"] =
      obj({{"source_root", txt()}, {"catalogue", ref("Catalogue")}});
  d["GraphValidateInput"] =
      obj({{"source_root", txt()}, {"graph", ref("Graph")}});
  d["GraphValidation"] = sealed("symphony.shv.graph-validation.v1",
                                {{"graph_digest", digest_type()},
                                 {"catalogue_digest", digest_type()},
                                 {"valid", constant(true)}});
  for (const auto *name :
       {"token", "version", "digest", "featureId", "operationId", "operation",
        "interaction", "limits", "Descriptor"})
    d[name] = graph.at(name);
  d["QuarterValue"] = obj(
      {{"precision", constant("quarter")},
       {"source_text", {{"type", "string"}, {"pattern", "^Q[1-4]'[0-9]{2}$"}}},
       {"from", date()},
       {"through", date()}});
  d["TableRowsValue"] =
      obj({{"columns", unique_array(txt(256), 16, 1)},
           {"rows",
            merge(arr(merge(arr({{"type", "string"}, {"maxLength", 4096}}, 16),
                            {{"minItems", 1}}),
                      32),
                  {{"minItems", 1}})}});
  d["Value"]["oneOf"].push_back(ref("QuarterValue"));
  d["Value"]["oneOf"].push_back(ref("TableRowsValue"));
  d["ScalarTableField"] = obj(
      {{"predicate", id()},
       {"section", section},
       {"label", txt(256)},
       {"next_label", nullable(txt(256))},
       {"value_type",
        enumeration({"string", "integer", "date", "tokens", "quarter_20yy"})},
       {"qualifier", txt(256)}});
  d["MatrixTableField"] = obj({{"predicate", id()},
                               {"section", section},
                               {"columns", unique_array(txt(256), 16, 1)},
                               {"value_type", constant("table_rows")},
                               {"qualifier", txt(256)}});
  d["TableSubjectSpec"] =
      obj({{"id", id()},
           {"manufacturer", txt(256)},
           {"model", txt(256)},
           {"hardware_class", id()},
           {"source_id", id()},
           {"heading_section", section},
           {"interpretation_profile", constant("scoped_tables.v1")},
           {"fields", arr({{"oneOf", Json::array({ref("ScalarTableField"),
                                                  ref("MatrixTableField")})}},
                          16)}});
  d["LegacySubjectSpec"] = d["SubjectSpec"];
  d["SubjectSpec"] = {{"oneOf", Json::array({ref("LegacySubjectSpec"),
                                             ref("TableSubjectSpec")})}};
  const auto pdf =
      read_json(root / "modules/shv-pdf-adapter/schemas/v1/pdf.schema.json");
  d["PDFSubjectSpec"] = obj(
      {{"id", id()},
       {"manufacturer", txt(256)},
       {"model", txt(256)},
       {"hardware_class", id()},
       {"source_id", id()},
       {"heading_section", constant("table8")},
       {"interpretation_profile", constant("pdf_opn.v1")},
       {"document", obj({{"decoder_root", txt()},
                         {"extraction", pdf.at("$defs").at("Extraction")}})},
       {"fields", arr(obj({{"predicate", id()},
                           {"label", constant("OPN")},
                           {"next_label", constant("Model")},
                           {"value_type", constant("string")},
                           {"qualifier",
                            constant("issuer=AMD;namespace=opn;profile=1")}}),
                      16)}});
  d["SubjectSpec"]["oneOf"].push_back(ref("PDFSubjectSpec"));
  const auto schema =
      document("symphony.shv.kernel-schema.v1",
               "Graph, Node, Edge copied verbatim from generic graph adapter "
               "contract. Structural schema checks do not replace source "
               "replay, canonical seals, sorting or runtime budgets.",
               d);
  const Json templates{
      {"inspect", Json::object()},
      {"coverage_default", {{"as_of", nullptr}}},
      {"coverage_plan", {{"profile", nullptr}, {"subjects", Json::array()}}},
      {"catalogue_build",
       {{"source_root", nullptr},
        {"sources", Json::array()},
        {"subjects", Json::array()}}},
      {"catalogue_query",
       {{"source_root", nullptr},
        {"catalogue", nullptr},
        {"subject_ids", Json::array()}}},
      {"evaluate",
       {{"source_root", nullptr},
        {"catalogue", nullptr},
        {"subject_ids", Json::array()},
        {"requirements", Json::array()}}},
      {"graph_project", {{"source_root", nullptr}, {"catalogue", nullptr}}},
      {"graph_validate", {{"source_root", nullptr}, {"graph", nullptr}}}};
  return json_outputs(
      {{"modules/shv-engine/schemas/v1/shv.schema.json", schema},
       {"modules/shv-engine/schemas/v1/shv.templates.json", templates}});
}
inline Outputs profile(const fs::path &root) {
  auto d = read_json(root / "modules/shv-engine/schemas/v1/shv.schema.json")
               .at("$defs");
  d["Metric"] =
      obj({{"predicate", id()},
           {"value_type", enumeration({"string", "integer", "date", "tokens",
                                       "quarter_20yy", "table_rows"})},
           {"qualifier", txt(256)},
           {"required", {{"type", "boolean"}}},
           {"description", txt()},
           {"extensions", {{"type", "object"}}}});
  d["ProfileCompileInput"] = obj({{"id", id()},
                                  {"revision", id()},
                                  {"hardware_class", id()},
                                  {"metrics", arr(ref("Metric"), 16)},
                                  {"extensions", {{"type", "object"}}}});
  d["ClassProfile"] = sealed("symphony.shv.class-profile.v1",
                             {{"definition", ref("ProfileCompileInput")}});
  for (const auto *base :
       {"LegacySubjectSpec", "TableSubjectSpec", "PDFSubjectSpec",
        "ScalarTableField", "MatrixTableField"}) {
    const auto name = "Declaration" + std::string(base);
    d[name] = d.at(base);
    for (const auto *key : {"heading_section", "field_section", "section"}) {
      auto &p = d[name]["properties"];
      if (p.contains(key) && p.at(key).contains("pattern"))
        p[key] = txt(128);
    }
  }
  d["DeclarationTableSubjectSpec"]["properties"]["fields"] =
      arr({{"oneOf", Json::array({ref("DeclarationScalarTableField"),
                                  ref("DeclarationMatrixTableField")})}},
          16);
  d["DeclarationPDFSubjectSpec"]["properties"]["document"] = {
      {"oneOf", Json::array({obj({{"decoder_root", txt()},
                                  {"extraction", {{"type", "object"}}}}),
                             obj({{"decoder_binding", id()},
                                  {"extraction", {{"type", "object"}}}})})}};
  d["Declaration"] = {
      {"oneOf", Json::array({ref("DeclarationLegacySubjectSpec"),
                             ref("DeclarationTableSubjectSpec"),
                             ref("DeclarationPDFSubjectSpec")})}};
  d["PortablePDFSubjectSpec"] = d["DeclarationPDFSubjectSpec"];
  d["PortablePDFSubjectSpec"]["properties"]["document"] =
      obj({{"decoder_binding", id()}, {"extraction", {{"type", "object"}}}});
  d["PortableMapping"] = {
      {"oneOf", Json::array({ref("DeclarationLegacySubjectSpec"),
                             ref("DeclarationTableSubjectSpec"),
                             ref("PortablePDFSubjectSpec")})}};
  d["MappingDiagnoseInput"] = obj({{"profile", ref("ClassProfile")},
                                   {"mapping", arr(ref("Declaration"), 32)}});
  d["Finding"] =
      obj({{"predicate", id()},
           {"status", enumeration({"matched", "mismatch", "unmapped_required",
                                   "unmapped_optional"})},
           {"expected", ref("Metric")},
           {"observed",
            {{"anyOf", Json::array({{{"type", "null"}},
                                    obj({{"value_type", txt(32)},
                                         {"qualifier", txt(256)}})})}}},
           {"differences",
            unique_array(enumeration({"value_type", "qualifier"}), 2)}});
  d["MappingDiagnostics"] = sealed(
      "symphony.shv.mapping-diagnostics.v1",
      {{"input", ref("MappingDiagnoseInput")},
       {"subjects", arr(obj({{"subject_id", id()},
                             {"status", enumeration({"conformant", "incomplete",
                                                     "not_applicable"})},
                             {"findings", arr(ref("Finding"), 16)},
                             {"extension_predicates", arr(id(), 16)}}),
                        32)},
       {"counts", counts({"conformant", "incomplete", "not_applicable"}, 32)},
       {"evidence_scope", constant("mapping_declarations_only")}});
  d["UniverseBuildInput"] = obj(
      {{"id", id()},
       {"revision", id()},
       {"kernel_version", constant("0.3.0-dev")},
       {"coverage", ref("CoverageProfile")},
       {"profiles", arr(ref("ClassProfile"), 16)},
       {"sources", arr(ref("Source"), 8)},
       {"mapping", arr(ref("PortableMapping"), 32)},
       {"locators",
        arr(obj({{"source_id", id()},
                 {"uri", txt()},
                 {"upstream_revision",
                  {{"anyOf", Json::array({{{"type", "null"}}, txt(256)})}}}}),
            8)},
       {"extensions", {{"type", "object"}}}});
  d["Universe"] = sealed("symphony.shv.universe.v1",
                         {{"definition", ref("UniverseBuildInput")}});
  d["UniverseBindInput"] =
      obj({{"universe", ref("Universe")},
           {"bindings", obj({{"source_root", txt()},
                             {"decoders",
                              {{"type", "object"},
                               {"propertyNames", id()},
                               {"additionalProperties", txt()}}}})}});
  const auto reader = obj({{"engine_id", constant("symphony-shv")},
                           {"version", constant("0.3.0-dev")},
                           {"mode", constant("compiled_exact_contract")}});
  d["UniverseBinding"] =
      sealed("symphony.shv.universe-binding.v1",
             {{"input", ref("UniverseBindInput")},
              {"reader", reader},
              {"catalogue_input", ref("CatalogueBuildInput")},
              {"catalogue", ref("Catalogue")},
              {"coverage", ref("CoverageResult")},
              {"conformance", arr(ref("MappingDiagnostics"), 16)},
              {"unprofiled_classes", unique_array(id(), 32)},
              {"canonical_apply_enabled", constant(false)}});
  d["NativeDiagnosticError"] =
      obj({{"code", txt(128)}, {"message", txt(8192)}});
  const Json error{{"anyOf", Json::array({{{"type", "null"}},
                                          ref("NativeDiagnosticError")})}};
  d["ExtractionDiagnoseInput"] =
      obj({{"source_root", txt()},
           {"sources", arr(ref("Source"), 8)},
           {"subjects", arr(ref("Declaration"), 32)}});
  d["ExtractionDiagnostics"] = sealed(
      "symphony.shv.extraction-diagnostics.v1",
      {{"input", ref("ExtractionDiagnoseInput")},
       {"reader", reader},
       {"sources", arr(obj({{"source_id", id()},
                            {"status", enumeration({"verified", "unverified"})},
                            {"error", error}}),
                       8)},
       {"subjects",
        arr(obj({{"subject_id", id()},
                 {"source_id", id()},
                 {"fields",
                  arr(obj({{"predicate", id()},
                           {"status", enumeration({"extracted", "failed",
                                                   "unavailable"})},
                           {"assertion",
                            {{"anyOf", Json::array({{{"type", "null"}},
                                                    ref("Assertion")})}}},
                           {"error", error}}),
                      16)}}),
            32)},
       {"counts", counts({"extracted", "failed", "unavailable"}, 256)},
       {"evidence_scope", constant("individual_field_source_replay")},
       {"canonical_apply_enabled", constant(false)}});
  d["ReferenceObject"] =
      obj({{"id", id()},
           {"kind", id()},
           {"digest", digest_type()},
           {"document", {{"type", Json::array({"object", "null"})}}}});
  d["ReferenceEdge"] = obj({{"from", id()}, {"to", id()}, {"pointer", txt()}});
  d["ReferencesAnalyzeInput"] =
      obj({{"objects", arr(ref("ReferenceObject"), 128)},
           {"edges", arr(ref("ReferenceEdge"), 512)},
           {"root_ids", unique_array(id(), 128)},
           {"candidate_ids", unique_array(id(), 128)}});
  d["ReferenceAnalysis"] = sealed(
      "symphony.shv.reference-analysis.v1",
      {{"input", ref("ReferencesAnalyzeInput")},
       {"edges", arr(ref("ReferenceEdge"), 512)},
       {"reachable_ids", arr(id(), 128)},
       {"candidates",
        arr(obj({{"id", id()},
                 {"status", enumeration({"reachable",
                                         "not_reachable_in_supplied_graph"})},
                 {"path",
                  {{"anyOf",
                    Json::array({{{"type", "null"}}, arr(id(), 128)})}}},
                 {"incoming_edges", arr(ref("ReferenceEdge"), 512)}}),
            128)},
       {"uninspected_object_ids", arr(id(), 128)},
       {"evidence_scope", constant("caller_selected_reference_edges")},
       {"deletion_authorized", constant(false)},
       {"canonical_apply_enabled", constant(false)}});
  std::set<std::string> needed;
  std::function<void(const std::string &)> visit;
  std::function<void(const Json &)> walk;
  visit = [&](const std::string &name) {
    if (needed.insert(name).second)
      walk(d.at(name));
  };
  walk = [&](const Json &value) {
    if (value.is_object()) {
      if (value.contains("$ref") &&
          string(value.at("$ref")).starts_with("#/$defs/"))
        visit(string(value.at("$ref")).substr(8));
      for (const auto &child : value)
        walk(child);
    } else if (value.is_array())
      for (const auto &child : value)
        walk(child);
  };
  for (const auto *name :
       {"InspectInput", "Descriptor", "ProfileCompileInput", "ClassProfile",
        "MappingDiagnoseInput", "MappingDiagnostics", "UniverseBuildInput",
        "Universe", "UniverseBindInput", "UniverseBinding",
        "ExtractionDiagnoseInput", "ExtractionDiagnostics",
        "ReferencesAnalyzeInput", "ReferenceAnalysis"})
    visit(name);
  Json selected = Json::object();
  for (auto it = d.begin(); it != d.end(); ++it)
    if (needed.contains(it.key()))
      selected[it.key()] = it.value();
  const auto schema = document(
      "symphony.shv.profile-schema.v1",
      "Structural checks only. Runtime enforces seals, correspondence, bounds, "
      "unique identities and source replay. Declaration checks do not "
      "interpret selectors or PDF extraction objects. The compiled exact "
      "kernel validates them at bind.",
      selected);
  const Json templates{
      {"profile_compile",
       {{"id", nullptr},
        {"revision", nullptr},
        {"hardware_class", nullptr},
        {"metrics", Json::array()},
        {"extensions", Json::object()}}},
      {"mapping_diagnose", {{"profile", nullptr}, {"mapping", Json::array()}}},
      {"universe_build",
       {{"id", nullptr},
        {"revision", nullptr},
        {"kernel_version", "0.3.0-dev"},
        {"coverage", nullptr},
        {"profiles", Json::array()},
        {"sources", Json::array()},
        {"mapping", Json::array()},
        {"locators", Json::array()},
        {"extensions", Json::object()}}},
      {"universe_bind",
       {{"universe", nullptr},
        {"bindings",
         {{"source_root", nullptr}, {"decoders", Json::object()}}}}},
      {"extraction_diagnose",
       {{"source_root", nullptr},
        {"sources", Json::array()},
        {"subjects", Json::array()}}},
      {"references_analyze",
       {{"objects", Json::array()},
        {"edges", Json::array()},
        {"root_ids", Json::array()},
        {"candidate_ids", Json::array()}}}};
  return json_outputs(
      {{"modules/shv-profile-engine/schemas/v1/profile.schema.json", schema},
       {"modules/shv-profile-engine/schemas/v1/profile.templates.json",
        templates}});
}
inline Outputs source(const fs::path &root) {
  const auto g =
      read_json(
          root /
          "modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json")
          .at("$defs");
  Json d = Json::object();
  for (const auto *name :
       {"Node", "Edge", "Graph", "token", "version", "digest", "featureId",
        "operationId", "operation", "interaction", "limits", "Descriptor"})
    d[name] = g.at(name);
  const auto uri = merge(
      txt(), {{"pattern",
               "^https?://[A-Za-z0-9][A-Za-z0-9.-]*(?::[1-9][0-9]{0,4})?(?:[/"
               "?][!-~]*)?$"}});
  d["Locator"] = obj({{"id", id()},
                      {"uri", uri},
                      {"format", enumeration({"html", "opaque"})}});
  d["SourceDefinition"] =
      obj({{"source_id", id()},
           {"publisher", txt(256)},
           {"authority_role", txt(256)},
           {"subject_ids", source_arr(id(), 32, 1, true)},
           {"locators", source_arr(ref("Locator"), 16, 1)}});
  d["Source"] = sealed("symphony.shv.source-revision.v1",
                       {{"definition", ref("SourceDefinition")},
                        {"generation", integer(1, 32)},
                        {"previous_digest", nullable(digest_type())}});
  d["InspectInput"] = obj(Json::object());
  d["SourcePlanInput"] = obj({{"operation_id", id()},
                              {"current", nullable(ref("Source"))},
                              {"desired", ref("SourceDefinition")},
                              {"reason", txt()}});
  d["SourcePlan"] =
      sealed("symphony.shv.source-plan.v1",
             {{"operation_id", id()},
              {"expected_state_digest", nullable(digest_type())},
              {"change_kind",
               enumeration({"onboard", "authority_change", "relocation"})},
              {"reason", txt()},
              {"source", ref("Source")}});
  d["SourceReduceInput"] =
      obj({{"current", nullable(ref("Source"))}, {"plan", ref("SourcePlan")}});
  d["SourceTransition"] =
      sealed("symphony.shv.source-transition.v1",
             {{"operation_id", id()},
              {"expected_state_digest", nullable(digest_type())},
              {"source", ref("Source")}});
  d["SourceStatusInput"] = obj({{"history", source_arr(ref("Source"), 32, 1)}});
  d["SourceStatus"] =
      sealed("symphony.shv.source-status.v1",
             {{"source", ref("Source")},
              {"history_digests", source_arr(digest_type(), 32, 1, true)}});
  d["Manifest"] =
      read_json(root / "modules/shv-engine/schemas/v1/shv.schema.json")
          .at("$defs")
          .at("Source");
  d["UpstreamRevision"] = obj({{"scheme", txt(128)}, {"value", txt(512)}});
  const Json capture{
      {"source", ref("Source")},
      {"locator_id", id()},
      {"resolved_uri", uri},
      {"redirect_chain", source_arr(uri, 8, 0, true)},
      {"observed_at",
       {{"type", "string"},
        {"pattern",
         "^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"}}},
      {"upstream_revision", nullable(ref("UpstreamRevision"))},
      {"manifest", ref("Manifest")},
      {"completeness", enumeration({"complete", "partial"})},
      {"issues", source_arr(txt(512), 32, 0, true)}};
  d["CaptureImportInput"] = obj(merge({{"source_root", txt()}}, capture));
  d["Capture"] = sealed("symphony.shv.source-capture.v1", capture);
  d["CaptureCompareInput"] = obj({{"source_root", txt()},
                                  {"previous", ref("Capture")},
                                  {"current", ref("Capture")}});
  d["CaptureComparison"] = sealed(
      "symphony.shv.capture-comparison.v1",
      {{"previous_digest", digest_type()},
       {"current_digest", digest_type()},
       {"same_logical_source", {{"type", "boolean"}}},
       {"changes",
        source_arr(enumeration({"source_identity", "source_revision", "body",
                                "representation", "acquisition_route",
                                "upstream_revision", "completeness",
                                "observation_time", "manifest_identity"}),
                   9, 0, true)}});
  d["Bundle"] = sealed("symphony.shv.source-bundle.v1",
                       {{"captures", source_arr(ref("Capture"), 8)}});
  d["GraphProjectInput"] = obj(
      {{"source_root", txt()}, {"captures", source_arr(ref("Capture"), 8)}});
  d["GraphValidateInput"] =
      obj({{"source_root", txt()}, {"graph", ref("Graph")}});
  d["GraphValidation"] = sealed("symphony.shv.source-graph-validation.v1",
                                {{"graph_digest", digest_type()},
                                 {"bundle_digest", digest_type()},
                                 {"valid", constant(true)}});
  const auto schema = document(
      "symphony.shv.source-schema.v1",
      "Strict structural discovery. Runtime additionally checks UTF-8 byte "
      "lengths, finite URI syntax/port bounds, exact seals, lineage, "
      "uniqueness by key, routes/calendar, consumed file bytes and graph "
      "correspondence. Generic graph/descriptor definitions copied verbatim.",
      d);
  Json import{{"source_root", nullptr}};
  for (auto it = capture.begin(); it != capture.end(); ++it)
    import[it.key()] = (it.key() == "redirect_chain" || it.key() == "issues")
                           ? Json::array()
                           : Json(nullptr);
  const Json templates{
      {"inspect", Json::object()},
      {"source_plan",
       {{"operation_id", nullptr},
        {"current", nullptr},
        {"desired", nullptr},
        {"reason", nullptr}}},
      {"source_reduce", {{"current", nullptr}, {"plan", nullptr}}},
      {"source_status", {{"history", Json::array()}}},
      {"capture_import", import},
      {"capture_compare",
       {{"source_root", nullptr}, {"previous", nullptr}, {"current", nullptr}}},
      {"graph_project",
       {{"source_root", nullptr}, {"captures", Json::array()}}},
      {"graph_validate", {{"source_root", nullptr}, {"graph", nullptr}}}};
  return json_outputs(
      {{"modules/shv-source-engine/schemas/v1/source.schema.json", schema},
       {"modules/shv-source-engine/schemas/v1/source.templates.json",
        templates}});
}
inline Outputs activation(const fs::path &root) {
  const auto source_path =
      root / "modules/shv-source-engine/schemas/v1/source.schema.json";
  auto d = read_json(source_path).at("$defs");
  const Json uuid{
      {"type", "string"},
      {"pattern",
       "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"}};
  Json installation = Json::object();
  for (const auto *name : {"Role", "ModuleID", "EngineID", "Version", "Prefix",
                           "ReceiptPath", "ReceiptDigest", "ReceiptProtocol",
                           "ExecutablePath", "ExecutableDigest"})
    installation[name] = {{"type", "string"}, {"minLength", 1}};
  d["ActivationInstallation"] = obj(installation);
  d["ActivationAuthorization"] = read_json(
      root / "knowledge/ssiag/schemas/v1/authorization-decision.schema.json");
  d["ActivationCapability"] =
      read_json(root / "knowledge/ssiag/schemas/v1/capability.schema.json");
  d["ActivationIntent"] = obj({{"operation_id", id()},
                               {"plan", ref("SourcePlan")},
                               {"transition", ref("SourceTransition")},
                               {"installation", ref("ActivationInstallation")},
                               {"digest", digest_type()}});
  d["ActivationAttempt"] =
      obj({{"intent", ref("ActivationIntent")},
           {"status", enumeration({"prepared", "authorized", "committed"})},
           {"correlation_id", uuid},
           {"authorization", nullable(ref("ActivationAuthorization"))},
           {"prior_authorizations", arr(ref("ActivationAuthorization"), 64)}});
  d["ActivationProposalInput"] = obj({{"operation_id", id()},
                                      {"desired", ref("SourceDefinition")},
                                      {"reason", txt()}});
  d["ActivationStore"] =
      obj({{"protocol", constant("symphony.qxctl.shv-source-store.v1")},
           {"tops_id", uuid},
           {"source_id", id()},
           {"source", nullable(ref("Source"))},
           {"state_digest", nullable(digest_type())},
           {"head_operation_id", nullable(id())},
           {"operations",
            {{"type", "object"},
             {"propertyNames", id()},
             {"maxProperties", 128},
             {"additionalProperties", ref("ActivationAttempt")}}},
           {"digest", digest_type()}});
  d["ActivationResult"] = obj(
      {{"protocol", constant("symphony.qxctl.shv-source-activation-result.v1")},
       {"operation", enumeration({"apply", "recover", "status"})},
       {"tops_id", uuid},
       {"source_id", id()},
       {"source", nullable(ref("Source"))},
       {"state_digest", nullable(digest_type())},
       {"head_operation_id", nullable(id())},
       {"history", arr(ref("Source"), 32)},
       {"attempt", nullable(ref("ActivationAttempt"))},
       {"owner_result",
        {{"anyOf", Json::array({{{"type", "null"}},
                                ref("SourceTransition"),
                                ref("SourceStatus")})}}},
       {"canonical_apply_enabled", constant(false)},
       {"authorization_audit", constant("ssiag_policy_decision_only")},
       {"source_write_stav_receipt", {{"type", "null"}}},
       {"digest", digest_type()}});
  d["ActivationSchema"] = obj(
      {{"protocol", constant("symphony.qxctl.shv-source-activation-schema.v1")},
       {"installation", ref("ActivationInstallation")},
       {"origin", constant("qxctl_embedded")},
       {"schema", {{"type", "object"}}},
       {"digest", digest_type()}});
  d["ActivationTemplate"] =
      obj({{"protocol",
            constant("symphony.qxctl.shv-source-activation-template.v1")},
           {"installation", ref("ActivationInstallation")},
           {"operation", enumeration({"propose", "apply"})},
           {"template", {{"type", "object"}}},
           {"status", constant("unanswered_template_not_validated_input")},
           {"digest", digest_type()}});
  d["ActivationError"] =
      obj({{"protocol", constant("symphony.qxctl.error.v1")},
           {"outcome", constant("error")},
           {"command_id", nullable({{"type", "string"}})},
           {"error", obj({{"code", {{"type", "string"}}},
                          {"message", {{"type", "string"}}},
                          {"engine_code", nullable({{"type", "string"}})}})},
           {"exit_code", integer(1, 125)}});
  return json_outputs(
      {{"tools/qxctl/cmd/qxctl/shv_activation.schema.json",
        document("symphony.qxctl.shv-source-activation-schema-document.v1",
                 "CLI-owned activation definitions. Source definitions copied "
                 "from SHV source engine0.1.0-dev (" +
                     digest_bytes(read_bytes(source_path)) +
                     "); SSIAG decision/capability schemas retain their owner "
                     "IDs. Shape checks do not authenticate authority, "
                     "validate seals or replace source reduction.",
                 d)}});
}
inline Json scoped(const fs::path &path, const std::string &name) {
  auto value = read_json(path);
  value.erase("$id");
  std::function<void(Json &)> walk = [&](Json &node) {
    if (node.is_object()) {
      for (auto it = node.begin(); it != node.end(); ++it) {
        if (it.key() == "$ref") {
          require(it.value().is_string(), "schema reference must be a string");
          if (string(it.value()).starts_with("#/$defs/"))
            it.value() =
                "#/$defs/" + name + "/$defs/" + string(it.value()).substr(8);
        } else
          walk(it.value());
      }
    } else if (node.is_array())
      for (auto &child : node)
        walk(child);
  };
  walk(value);
  return value;
}
inline Outputs refresh(const fs::path &root) {
  const auto object = [](const Json &fields) { return obj(fields, true); };
  Json d{
      {"Kernel", scoped(root / "modules/shv-engine/schemas/v1/shv.schema.json",
                        "Kernel")},
      {"SourceOwner",
       scoped(root / "modules/shv-source-engine/schemas/v1/source.schema.json",
              "SourceOwner")}};
  d["Installation"] =
      read_json(root / "tools/qxctl/cmd/qxctl/shv_activation.schema.json")
          .at("$defs")
          .at("ActivationInstallation");
  auto capture = d.at("SourceOwner").at("$defs").at("CaptureImportInput");
  Json required = Json::array();
  for (const auto &name : capture.at("required"))
    if (name != "source_root" && name != "source")
      required.push_back(name);
  capture["required"] = required;
  capture["properties"].erase("source_root");
  capture["properties"].erase("source");
  d["CaptureSpec"] = capture;
  const Json capture_array{{"type", "array"},
                           {"minItems", 1},
                           {"maxItems", 8},
                           {"items", ref("CaptureSpec")}};
  const auto request =
      object({{"expected_source_digest", ref("SourceOwner/$defs/digest")},
              {"captures", capture_array},
              {"mapping",
               {{"type", "array"},
                {"maxItems", 32},
                {"items", ref("Kernel/$defs/SubjectSpec")}}},
              {"profile", ref("Kernel/$defs/CoverageProfile")},
              {"subject_ids",
               {{"type", "array"},
                {"maxItems", 32},
                {"uniqueItems", true},
                {"items", ref("Kernel/$defs/token")}}},
              {"requirements",
               {{"type", "array"},
                {"maxItems", 32},
                {"items", ref("Kernel/$defs/Requirement")}}}});
  d["Request"] = request;
  d["Bundle"] =
      object({{"protocol", constant("symphony.qxctl.shv-refresh-bundle.v1")},
              {"tops_id", {{"type", "string"}, {"format", "uuid"}}},
              {"source_id", ref("SourceOwner/$defs/token")},
              {"source", ref("SourceOwner/$defs/Source")},
              {"source_installation", ref("Installation")},
              {"kernel_installation", ref("Installation")},
              {"request", ref("Request")},
              {"captures",
               {{"type", "array"},
                {"minItems", 1},
                {"maxItems", 8},
                {"items", ref("SourceOwner/$defs/Capture")}}},
              {"catalogue", ref("Kernel/$defs/Catalogue")},
              {"coverage", ref("Kernel/$defs/CoverageResult")},
              {"evaluation", ref("Kernel/$defs/Evaluation")},
              {"source_graph", ref("SourceOwner/$defs/Graph")},
              {"catalogue_graph", ref("Kernel/$defs/Graph")},
              {"digest", ref("SourceOwner/$defs/digest")}});
  d["Verification"] = object(
      {{"protocol", constant("symphony.qxctl.shv-refresh-verification.v1")},
       {"bundle_digest", ref("SourceOwner/$defs/digest")},
       {"selected_source_digest", ref("SourceOwner/$defs/digest")},
       {"current_source_digest", ref("SourceOwner/$defs/digest")},
       {"source_is_current", {{"type", "boolean"}}},
       {"valid", constant(true)},
       {"digest", ref("SourceOwner/$defs/digest")}});
  const Json base{{"source_installation", ref("Installation")},
                  {"kernel_installation", ref("Installation")},
                  {"digest", ref("SourceOwner/$defs/digest")}};
  d["Schema"] = object(merge(
      base, {{"protocol", constant("symphony.qxctl.shv-refresh-schema.v1")},
             {"origin", constant("qxctl_embedded")},
             {"schema", {{"type", "object"}}}}));
  const auto blank_properties = [](const Json &properties) {
    Json blank = Json::object();
    for (auto it = properties.begin(); it != properties.end(); ++it)
      blank[it.key()] = {{"type", "null"}};
    return blank;
  };
  d["Template"] = object(merge(
      base,
      {{"protocol", constant("symphony.qxctl.shv-refresh-template.v1")},
       {"status", constant("unanswered_template_not_validated_input")},
       {"template", object(blank_properties(request.at("properties")))}}));
  Json endpoint_properties = Json::object();
  for (const auto *key :
       {"bundle_path", "source_root", "state_root", "tops_id", "source_id",
        "source_prefix", "source_version", "kernel_prefix", "kernel_version"})
    endpoint_properties[key] = {{"type", "string"}, {"minLength", 1}};
  const auto endpoint = object(endpoint_properties);
  d["ComparisonEndpoint"] = endpoint;
  d["ComparisonRequest"] = object({{"previous", ref("ComparisonEndpoint")},
                                   {"current", ref("ComparisonEndpoint")}});
  const Json change_base{{"kind", enumeration({"added", "removed", "changed"})},
                         {"subject_id", ref("Kernel/$defs/token")}};
  d["SubjectChange"] = object(
      merge(change_base,
            {{"previous", nullable(ref("Kernel/$defs/SubjectSummary"))},
             {"current", nullable(ref("Kernel/$defs/SubjectSummary"))}}));
  d["AssertionChange"] = object(merge(
      change_base, {{"predicate", ref("Kernel/$defs/token")},
                    {"previous", nullable(ref("Kernel/$defs/Assertion"))},
                    {"current", nullable(ref("Kernel/$defs/Assertion"))}}));
  d["Dimension"] = object(
      {{"dimension",
        enumeration({"source", "source_installation", "kernel_installation",
                     "mapping", "profile", "subject_ids", "requirements",
                     "capture_bodies", "capture_manifests",
                     "capture_observations", "catalogue", "coverage",
                     "evaluation", "source_graph", "catalogue_graph"})},
       {"changed", {{"type", "boolean"}}},
       {"previous_digest", ref("SourceOwner/$defs/digest")},
       {"current_digest", ref("SourceOwner/$defs/digest")}});
  d["Comparison"] = object(
      {{"protocol", constant("symphony.qxctl.shv-refresh-comparison.v1")},
       {"tops_id", {{"type", "string"}, {"format", "uuid"}}},
       {"source_id", ref("SourceOwner/$defs/token")},
       {"previous_bundle_digest", ref("SourceOwner/$defs/digest")},
       {"current_bundle_digest", ref("SourceOwner/$defs/digest")},
       {"same_bundle", {{"type", "boolean"}}},
       {"previous_replay", ref("Verification")},
       {"current_replay", ref("Verification")},
       {"observation_scope", constant("sequential_source_replays")},
       {"causal_attribution", constant("not_inferred")},
       {"dimensions",
        {{"type", "array"},
         {"minItems", 15},
         {"maxItems", 15},
         {"items", ref("Dimension")}}},
       {"subject_changes",
        {{"type", "array"}, {"maxItems", 64}, {"items", ref("SubjectChange")}}},
       {"assertion_changes",
        {{"type", "array"},
         {"maxItems", 512},
         {"items", ref("AssertionChange")}}},
       {"digest", ref("SourceOwner/$defs/digest")}});
  const auto old = d.at("Template").at("properties").at("template");
  const auto blank = object(blank_properties(endpoint.at("properties")));
  d["Template"]["properties"]["template"] = {
      {"oneOf",
       Json::array({old, object({{"previous", blank}, {"current", blank}})})}};
  return json_outputs(
      {{"tools/qxctl/cmd/qxctl/shv_refresh.schema.json",
        {{"$schema", "https://json-schema.org/draft/2020-12/schema"},
         {"$defs", d}}}});
}
inline void generate(const Arguments &args,
                     const std::function<Outputs(const fs::path &)> &builder) {
  const auto outputs = builder(args.root);
  emit(args.root, outputs, args.check);
  std::cout << (args.check ? "Checked " : "Generated ") << outputs.size()
            << " schema/template resources\n";
}
} // namespace symphony::authoring::schema
