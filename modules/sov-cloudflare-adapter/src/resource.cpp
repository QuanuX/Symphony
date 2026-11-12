#include "adapter.hpp"
#include <algorithm>
#include <arpa/inet.h>
#include <regex>
#include <stdexcept>

namespace cloudflare {
namespace {
void require(bool ok) { if (!ok) throw std::runtime_error("invalid resource plan"); }
void replace(std::string& text, const std::string& key, const std::string& value) {
    const auto at = text.find(key);
    if (at != std::string::npos) text.replace(at, key.size(), value);
}
}
Json resource_preview(const Json& input) {
    const Json* profile = nullptr;
    const auto declaration = owner();
    for (const auto& p : declaration.at("resource_profiles"))
        if (p.at("operation") == input.at("resource_operation")) profile = &p;
    require(profile != nullptr);
    const auto& p = *profile;
    const auto& context = input.at("context");
    const auto& parameters = input.at("parameters");
    validate(parameters, p.at("parameters_schema"));
    const auto account = context.at("account_id").get<std::string>();
    const auto zone = context.at("zone_id");
    if (p.at("scope") == "zone") require(!zone.is_null());
    const auto operation = p.at("operation").get<std::string>();
    const auto& body = parameters.at("body");
    if (operation.starts_with("dns.") && !body.is_null()) {
        const auto type = body.at("type").get<std::string>();
        const auto content = body.at("content").get<std::string>();
        unsigned char address[16]{};
        if (type == "A" || type == "AAAA") require(::inet_pton(type == "A" ? AF_INET : AF_INET6, content.c_str(), address) == 1);
        else require(std::regex_match(content, std::regex("[a-z0-9][a-z0-9.-]{0,251}[a-z0-9]")));
    }
    if (operation == "tunnels.config.update") {
        const auto& ingress = body.at("config").at("ingress");
        for (std::size_t i = 0; i < ingress.size(); ++i) {
            const bool last = i + 1 == ingress.size();
            require(last ? (ingress[i].at("hostname").is_null() && ingress[i].at("service") == "http_status:404") : !ingress[i].at("hostname").is_null());
        }
    }
    std::string path = p.at("path");
    Json paths = Json::object();
    if (operation.starts_with("zero-trust.access.applications.")) {
        replace(path, "{account_or_zone}", "accounts"); replace(path, "{account_or_zone_id}", account);
        paths["account-or-zone"] = "accounts"; paths["account-or-zone-id"] = account;
    } else if (p.at("scope") == "account") { replace(path, "{account_id}", account); }
    else { replace(path, "{zone_id}", zone.get<std::string>()); paths["zone-id"] = zone; }
    std::vector<std::string> argv = p.at("vendor_command").get<std::vector<std::string>>();
    if (!p.at("resource_parameter").is_null()) {
        auto key = p.at("resource_parameter").get<std::string>();
        const auto value = parameters.at("resource_id").get<std::string>();
        replace(path, "{" + key + "}", value);
        std::replace(key.begin(), key.end(), '_', '-'); paths[key] = value;
        argv.push_back(value);
    }
    require(path.find('{') == std::string::npos && path.find('}') == std::string::npos);
    if (p.at("scope") == "zone") { argv.push_back("--zone"); argv.push_back(zone.get<std::string>()); }
    Json query = Json::object();
    if (!parameters.at("page").is_null()) {
        query = parameters.at("page");
        argv.insert(argv.end(), {"--page", query.at("page").dump(), "--per-page", query.at("per_page").dump()});
    }
    if (!body.is_null()) { argv.push_back("--body"); argv.push_back(body.dump()); }
    // Dry run has no confirmation side effect; a future delete execution must
    // separately qualify --force semantics, not infer them from this argv.
    argv.push_back("--dry-run");
    std::string command = "cf";
    for (const auto& word : p.at("vendor_command")) command += " " + word.get<std::string>();
    Json preview{{"command", command}, {"method", p.at("method")}, {"url", "https://api.cloudflare.com/client/v4" + path}, {"pathParams", paths}, {"bodyKind", body.is_null() ? "none" : "json"}};
    if (!parameters.at("page").is_null() || (operation.starts_with("dns.") && p.at("method") != "DELETE")) preview["query"] = query;
    if (!body.is_null()) preview["body"] = body;
    return Json{{"request", preview}, {"preview_argv", argv}, {"nonsecret_environment", {{"CLOUDFLARE_ACCOUNT_ID", account}}}, {"profile_digest", digest(p, "symphony.sov.cloudflare-resource-profile.v1")}, {"output_class", p.at("output_class")}, {"required_credential_channel", p.at("required_credential_channel")}, {"required_result_channel", p.at("required_result_channel")}, {"retry_qualification", p.at("retry_qualification")}, {"precondition_mode", p.at("precondition_mode")}, {"execution_available", input.contains("execution") && !input.at("execution").is_null()}};
}
}
