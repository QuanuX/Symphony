#pragma once
#include "catalog.hpp"
#include "symphony/knowledge/engine/digest.hpp"

namespace symphony::srrapiv::test_fixture {
using symphony::knowledge::engine::tagged_sha256;
inline void seal(Json& v, const std::string& field) {
    v.erase(field);
    v[field] = tagged_sha256(v.dump());
}
inline Json artifact(const std::string& sdk = "13.7.0.0", const std::string& state = "state.v1",
              const std::string& journal = "journal.v1") {
    Json a = {{"sdk_version",sdk},{"product","rithmic-rapi-plus-cpp"},
      {"archive_digest",tagged_sha256(sdk + "-archive")},{"header_digest",tagged_sha256(sdk + "-header")},
      {"library_digest",tagged_sha256(sdk + "-library")},{"adapter_version","0.1.0"},
      {"adapter_digest",tagged_sha256(sdk + "-adapter")},
      {"platform",{{"os","darwin"},{"architecture","arm64"},{"compiler","clang-17"},
                   {"abi","libcxx-test"},{"dependency_digest",tagged_sha256("deps")}}},
      {"contracts",{{"api_mapping","rapi-mapping.v1"},{"control","srrapiv-control.v1"},
                    {"event","srrapiv-event.v1"},{"state",state},{"journal",journal},
                    {"state_reader","srrapiv-reader.v1"},{"readable_state_schemas",Json::array({state})},
                    {"readable_journal_schemas",Json::array({journal})}}},
      {"connection",{{"profile_digest",tagged_sha256("profile")},{"trust_digest",tagged_sha256("trust")}}},
      {"capabilities",Json::array({"getVersion","subscribe","sendOrder:limit"})}};
    seal(a,"artifact_id");
    return a;
}
inline Json binding(const Json& a) {
    Json b = {{"schema","symphony.srrapiv.binding.v1"},{"artifact_id",a.at("artifact_id")},
      {"strategy_id","strategy:example"},{"environment","Rithmic Test"},{"account_ref","account:example"},
      {"credential_reference",{{"provider_id","ssiag:provider:example"},{"resource_id","credential:example"}}},
      {"required_capabilities",Json::array({"getVersion","subscribe"})},
      {"required_evidence_scope","provider_test"},{"feed_profile_digest",tagged_sha256("raw-feed")},
      {"metadata_digest",tagged_sha256("metadata")}};
    seal(b,"binding_id");
    return b;
}
inline Json input(const Json& a) {
    return {{"catalog",{{"schema","symphony.srrapiv.catalog.v1"},{"artifacts",Json::array({a})}}},
            {"binding",binding(a)},{"compatibility_evidence",Json::array()}};
}
inline Json sample_binding_input() { return input(artifact()); }
}
