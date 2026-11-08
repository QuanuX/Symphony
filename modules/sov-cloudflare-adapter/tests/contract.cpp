#include "adapter.hpp"
#include "native_test.hpp"
using namespace native_test;
namespace {
struct Harness {
    TempDir temporary;
    fs::path executable, store;
    Json owner, installation;
    explicit Harness(const fs::path& binary) : executable(binary), store(temporary.path / "store") {
        owner = Json::parse(native_test::run({binary.string(), "--describe"}).stdout_text);
        installation = Json{{"version", "0.1.0-dev"}, {"executable_digest", digest(read(binary))}, {"receipt_digest", "sha256:" + std::string(64, 'a')}};
    }
    Json request(const std::string& op, const Json& input, Json plan = nullptr) {
        return Json{{"protocol", "symphony.sov.cloudflare-request.v1"}, {"request_id", "12345678-1234-4234-8234-123456789abc"}, {"correlation_id", "23456789-2345-4345-8345-23456789abcd"}, {"operation", op}, {"descriptor_ref", cloudflare::digest(owner, "symphony.sov.cloudflare-owner.v1")}, {"installation_ref", installation}, {"context_ref", nullptr}, {"plan_ref", plan}, {"input", input}};
    }
    ProcessResult raw(const std::string& data) { return native_test::run({executable.string()}, data, 40, {{"CLOUDFLARE_API_TOKEN", "SECRET-SENTINEL"}, {"NODE_OPTIONS", "--invalid-ambient-option"}}); }
    Json call(const std::string& op, const Json& input, Json plan = nullptr, bool ok = true) {
        auto req = request(op, input, plan); auto response = raw(req.dump());
        if (response.returncode != (ok ? 0 : 5)) std::cerr << "operation=" << op << " expected=" << (ok ? 0 : 5) << " actual=" << response.returncode << " stdout=" << response.stdout_text << " stderr=" << response.stderr_text << "\n";
        NT_REQUIRE(response.returncode == (ok ? 0 : 5));
        NT_REQUIRE(response.stdout_text.find("SENTINEL") == std::string::npos && response.stderr_text.find("SENTINEL") == std::string::npos);
        auto result = Json::parse(response.stdout_text); NT_REQUIRE(result.at("request_ref") == cloudflare::digest(req, "symphony.sov.cloudflare-request.v1")); return result;
    }
    std::pair<fs::path, std::string> fixture(const std::string& label = "bundle") {
        auto path = temporary.path / label;
        for (const auto& [p, bytes] : std::map<std::string, std::string>{{"node/bin/node", "inert fixture; never executed"}, {"node/LICENSE", "fixture"}, {"cf/package-lock.json", "{}"}, {"cf/node_modules/cf/bin/cf", "fixture"}, {"cf/node_modules/cf/package.json", "{\"version\":\"1.0.0-beta.28\"}"}, {"label", label}}) { write(path / p, bytes); ::chmod((path / p).c_str(), p == "node/bin/node" ? 0700 : 0600); }
        Json files = Json::array(); std::vector<fs::path> paths;
        for (const auto& p : fs::recursive_directory_iterator(path)) if (p.is_regular_file()) paths.push_back(p.path());
        std::sort(paths.begin(), paths.end());
        for (const auto& p : paths) { auto rel = p.lexically_relative(path).generic_string(); files.push_back(Json{{"path", rel}, {"digest", digest(read(p))}, {"size", fs::file_size(p)}, {"executable", rel == "node/bin/node"}}); }
#ifdef __APPLE__
        const char* os = "macos";
#else
        const char* os = "linux";
#endif
#if defined(__arm64__) || defined(__aarch64__)
        const char* arch = "arm64";
#else
        const char* arch = "amd64";
#endif
        Json bundle{{"protocol", "symphony.sov.cloudflare-bundle.v1"}, {"cf_version", "1.0.0-beta.28"}, {"node_version", "26.12.0"}, {"cf_integrity", owner.at("bundle_schema").at("properties").at("cf_integrity").at("const")}, {"os", os}, {"architecture", arch}, {"entrypoint", "cf/node_modules/cf/bin/cf"}, {"node", "node/bin/node"}, {"scripts_executed", false}, {"files", files}};
        write_json(path / "bundle.json", bundle); return {path, cloudflare::digest(bundle, "symphony.sov.cloudflare-bundle.v1")};
    }
    Json state() { return call("installation.list", {{"store", store.string()}}).at("data"); }
    Json plan(const std::string& action, const std::string& id, const fs::path& source = {}, Json prior = nullptr, bool ok = true) {
        return call("installation.plan", {{"store", store.string()}, {"action", action}, {"bundle_path", source.empty() ? Json(nullptr) : Json(source.string())}, {"bundle_digest", source.empty() ? Json(nullptr) : Json(id)}, {"installation_id", source.empty() ? Json(id) : Json(nullptr)}, {"expected_state", prior}}, nullptr, ok);
    }
    Json apply(const Json& p, bool ok = true) { auto d = p.at("data").at("plan_digest"); return call("installation.apply", {{"store", store.string()}, {"plan_digest", d}}, d, ok); }
};
void closed_frames(const fs::path& exe) {
    Harness h(exe); auto r = h.request("capability.list", {{"store", nullptr}, {"toolchain_id", nullptr}, {"query", nullptr}});
    for (const auto& raw : std::vector<std::string>{r.dump() + "{}", "{\"a\":1,\"a\":2}", std::string(1, char(255)), "{", "[]", std::string(8*1024*1024+1, ' ')}) NT_REQUIRE(h.raw(raw).returncode != 0);
    for (const auto* key : {"unexpected", "Input"}) { auto changed = r; changed[key] = "SECRET-SENTINEL"; auto out = h.raw(changed.dump()); NT_REQUIRE(out.returncode != 0 && out.stdout_text.find("SENTINEL") == std::string::npos); }
    h.call("capability.list", {{"store", nullptr}, {"toolchain_id", nullptr}}, nullptr, false);
    for (const auto* key : {"descriptor_ref", "context_ref"}) { auto changed = r; changed[key] = "sha256:" + std::string(64, '0'); NT_REQUIRE(Json::parse(h.raw(changed.dump()).stdout_text).at("error_code") == "binding_mismatch"); }
    r["installation_ref"]["executable_digest"] = "sha256:" + std::string(64, '0'); NT_REQUIRE(Json::parse(h.raw(r.dump()).stdout_text).at("error_code") == "binding_mismatch");
}
void descriptor_projection(const fs::path& exe) {
    Harness h(exe); NT_REQUIRE(h.call("capability.list", {{"store", nullptr}, {"toolchain_id", nullptr}, {"query", nullptr}}).at("data").at("owner") == h.owner);
    for (const auto& op : h.owner.at("operations")) {
        NT_REQUIRE(h.call("capability.template", {{"operation", op.at("operation")}}).at("data").at("input") == op.at("template"));
        NT_REQUIRE(h.call("capability.schema", {{"operation", op.at("operation")}, {"kind", "input"}, {"store", nullptr}, {"toolchain_id", nullptr}}).at("data").at("schema") == op.at("input_schema"));
    }
    NT_REQUIRE(h.call("resource.list", Json::object(), nullptr, false).at("error_code") == "invalid_control");
    h.call("made.up", Json::object(), nullptr, false);
}
void lifecycle(const fs::path& exe) {
    Harness h(exe); auto [path, id] = h.fixture(); auto p = h.plan("install", id, path); h.apply(p); NT_REQUIRE(h.state().at("state").at("selected").is_null()); h.apply(p, false);
    h.apply(h.plan("select", id, {}, h.state().at("state_digest"))); h.plan("remove", id, {}, h.state().at("state_digest"), false);
    auto [other, oid] = h.fixture("other"); auto update = h.plan("update", oid, other, h.state().at("state_digest")); auto stale = h.plan("select", id, {}, h.state().at("state_digest")); h.apply(update); h.apply(stale, false);
    h.apply(h.plan("rollback", id, {}, h.state().at("state_digest"))); h.apply(h.plan("remove", oid, {}, h.state().at("state_digest")));
    NT_REQUIRE(h.state().at("state").at("installations").at(oid).at("removed") == true && !fs::exists(h.store / "packages" / oid.substr(7)));
    h.plan("rollback", oid, {}, h.state().at("state_digest"), false); NT_REQUIRE(h.state().at("state").at("generation") == "5");
}
void integrity(const fs::path& exe) {
    Harness h(exe); auto [path,id] = h.fixture(); auto p = h.plan("install", id, path); write(path / "label", "changed"); h.apply(p, false); NT_REQUIRE(h.state().at("state").at("installations").empty());
    for (const std::string kind : {"extra", "symlink", "hardlink", "scripts"}) {
        auto [source, digest] = h.fixture(kind);
        if (kind == "extra") write(source / "unexpected", "not inventoried");
        else if (kind == "symlink") fs::create_symlink("/etc/passwd", source / "unexpected");
        else if (kind == "hardlink") fs::create_hard_link(source / "label", source / "linked");
        else { auto m = read_json(source / "bundle.json"); m["scripts_executed"] = true; write_json(source / "bundle.json", m); digest = cloudflare::digest(m, "symphony.sov.cloudflare-bundle.v1"); }
        h.plan("install", digest, source, nullptr, false);
    }
}
void context_and_no_unqualified_execution(const fs::path& exe) {
    Harness h(exe); auto [path,id] = h.fixture(); h.apply(h.plan("install", id, path));
    Json c{{"host_ref", "host:fixture"}, {"principal_uid", std::to_string(::getuid())}, {"account_id", std::string(32,'0')}, {"zone_id", nullptr}, {"credential_ref", "ssiag:fixture"}, {"toolchain_id", id}};
    auto out = h.call("context.plan", {{"store", h.store.string()}, {"context", c}, {"expected_context_ref", nullptr}});
    NT_REQUIRE(out.at("data").at("context_ref") == cloudflare::digest(c, "symphony.sov.cloudflare-context.v1") && out.at("data").at("authority_readiness") == "nonroot_local_owner");
    auto pd=out.at("data").at("plan_digest");auto applied=h.call("context.apply",{{"store",h.store.string()},{"plan_digest",pd}},pd);NT_REQUIRE(applied.at("effect_state")=="applied_verified");h.call("context.apply",{{"store",h.store.string()},{"plan_digest",pd}},pd,false);NT_REQUIRE(h.call("context.reconcile",{{"store",h.store.string()},{"plan_digest",pd}},pd).at("effect_state")=="applied_verified");
    c["principal_uid"] = std::to_string(::getuid()+1); h.call("context.inspect", {{"store", h.store.string()}, {"context", c}, {"expected_context_ref", nullptr}}, nullptr, false);
    NT_REQUIRE(h.call("capability.list", {{"store", h.store.string()}, {"toolchain_id", id}, {"query", "DNS"}}, nullptr, false).at("error_code") == "unavailable_contract");
}
void reconciliation(const fs::path& exe) {
    Harness h(exe); auto [source,id] = h.fixture(); auto p = h.plan("install", id, source); auto d = p.at("data").at("plan_digest").get<std::string>();
    auto stage = h.store / "staging" / d.substr(7); fs::create_directory(stage); ::chmod(stage.c_str(),0700); write(stage / "partial","incomplete");
    auto journal = [&](const std::string& pd) { write_json(h.store / "attempts" / (pd.substr(7)+".json"), Json{{"protocol", "symphony.sov.cloudflare-attempt.v1"}, {"plan_digest",pd}, {"stage","dispatch_started"}, {"effect_state","unresolved"}}); };
    journal(d); auto out = h.call("installation.reconcile", {{"store",h.store.string()}, {"plan_digest",d}}, d); NT_REQUIRE(out.at("effect_state") == "not_applied" && !fs::exists(stage)); h.apply(p,false);
    auto [other,oid] = h.fixture("other"); auto install = h.plan("install",oid,other);h.apply(install);auto committed = install.at("data").at("plan_digest").get<std::string>();journal(committed);
    NT_REQUIRE(h.call("installation.reconcile", {{"store",h.store.string()}, {"plan_digest",committed}},committed).at("effect_state") == "applied_verified");
    auto remove = h.plan("remove",oid,{},h.state().at("state_digest"));auto rd = remove.at("data").at("plan_digest").get<std::string>();auto package = h.store / "packages" / oid.substr(7);auto trash = h.store / "trash" / rd.substr(7);fs::rename(package,trash);journal(rd);
    NT_REQUIRE(h.call("installation.reconcile", {{"store",h.store.string()}, {"plan_digest",rd}},rd).at("effect_state") == "not_applied");NT_REQUIRE(fs::exists(package) && !fs::exists(trash));
}
void store_and_plan_substitution(const fs::path& exe) {
    Harness h(exe); auto [source,id] = h.fixture();auto target = h.temporary.path / "target";fs::create_directory(target);fs::create_directory_symlink(target,h.store);h.plan("install",id,source,nullptr,false);fs::remove(h.store);
    auto plan = h.plan("install",id,source);auto d=plan.at("data").at("plan_digest").get<std::string>();auto path=h.store / "plans" / (d.substr(7)+".json");auto changed=read_json(path);changed["action"]="remove";write_json(path,changed);h.apply(plan,false);
    write_json(path,plan.at("data").at("plan"));
    auto attempt=h.store/"attempts"/(d.substr(7)+".json");
    write_json(attempt,{{"protocol","symphony.sov.cloudflare-attempt.v1"},{"plan_digest",d},{"stage","settled"},{"effect_state","not_applied"},{"unexpected","SECRET-SENTINEL"}});
    NT_REQUIRE(h.call("installation.status",{{"store",h.store.string()},{"plan_digest",d}},nullptr,false).at("error_code")=="integrity_failure");
    h.call("installation.reconcile",{{"store",h.store.string()},{"plan_digest",d}},d,false);
}
void resource_planning(const fs::path& exe) {
    Harness h(exe); auto [source,id]=h.fixture();h.apply(h.plan("install",id,source));
    Json context{{"host_ref","host:fixture"},{"principal_uid",std::to_string(::getuid())},{"account_id",std::string(32,'0')},{"zone_id",std::string(32,'2')},{"credential_ref","ssiag:fixture-api-token"},{"toolchain_id",id}};
    auto input=[&](const Json& profile) { return Json{{"store",h.store.string()},{"context",context},{"expected_context_ref",cloudflare::digest(context,"symphony.sov.cloudflare-context.v1")},{"expected_state",h.state().at("state_digest")},{"resource_operation",profile.at("operation")},{"parameters",profile.at("template")},{"expected_observation",nullptr},{"execution",nullptr}}; };
    h.call("resource.plan",input(h.owner.at("resource_profiles").at(0)),nullptr,false);
    h.apply(h.plan("select",id,{},h.state().at("state_digest")));
    for(const auto& profile:h.owner.at("resource_profiles")) {
        auto in=input(profile);auto result=h.call("resource.plan",in);const auto& data=result.at("data");const auto& plan=data.at("plan");
        NT_REQUIRE(result.at("dispatch_state")=="not_dispatched" && data.at("provider_dispatch")==false && data.at("execution_available")==false);
        NT_REQUIRE(plan.at("input")==in && plan.at("uid")==std::to_string(::getuid()) && plan.at("authority_readiness")=="unavailable_contract");
        NT_REQUIRE(data.at("plan_digest")==cloudflare::digest(plan,"symphony.sov.cloudflare-resource-plan.v1"));
        const auto& request=plan.at("preview").at("request");
        NT_REQUIRE(request.at("method")==profile.at("method") && plan.at("preview").at("preview_argv").back()=="--dry-run");
        NT_REQUIRE(request.at("url").get<std::string>().starts_with("https://api.cloudflare.com/client/v4/"));
        if(profile.at("operation")=="dns.records.get") NT_REQUIRE(request.at("url")=="https://api.cloudflare.com/client/v4/zones/22222222222222222222222222222222/dns_records/11111111111111111111111111111111");
        const auto pd=data.at("plan_digest").get<std::string>();NT_REQUIRE(read_json(h.store/"plans"/(pd.substr(7)+".json"))==plan);
        h.call("installation.apply",{{"store",h.store.string()},{"plan_digest",pd}},pd,false);
        auto extra=in;extra["parameters"]["token"]="SECRET-SENTINEL";h.call("resource.plan",extra,nullptr,false);
        if(!in.at("parameters").at("page").is_null()) {
            for(const auto* field:{"page","per_page"}) {auto bad=in;bad["parameters"]["page"][field]=0;h.call("resource.plan",bad,nullptr,false);}
            auto bad=in;bad["parameters"]["page"]["per_page"]=101;h.call("resource.plan",bad,nullptr,false);
        }
        if(profile.at("method")=="DELETE") {auto bad=in;bad["parameters"]["confirm_delete"]=false;h.call("resource.plan",bad,nullptr,false);}
    }
    auto in=input(h.owner.at("resource_profiles").at(0));
    for(const auto* field:{"expected_state","expected_context_ref"}) {auto bad=in;bad[field]="sha256:"+std::string(64,'f');h.call("resource.plan",bad,nullptr,false);}
    auto bad=in;bad["context"]["principal_uid"]=std::to_string(::getuid()+1);bad["expected_context_ref"]=cloudflare::digest(bad.at("context"),"symphony.sov.cloudflare-context.v1");h.call("resource.plan",bad,nullptr,false);
    bad=in;bad["resource_operation"]="tunnels.token.rotate";h.call("resource.plan",bad,nullptr,false);
    for(const auto& profile:h.owner.at("resource_profiles")) {
        if(profile.at("operation")=="dns.records.create") {bad=input(profile);bad["parameters"]["body"]["type"]="A";bad["parameters"]["body"]["content"]="999.2.3.4";h.call("resource.plan",bad,nullptr,false);bad=input(profile);bad["context"]["zone_id"]=nullptr;bad["expected_context_ref"]=cloudflare::digest(bad.at("context"),"symphony.sov.cloudflare-context.v1");h.call("resource.plan",bad,nullptr,false);}
        if(profile.at("operation")=="tunnels.create") {bad=input(profile);bad["parameters"]["body"]["tunnel_secret"]="SECRET-SENTINEL";h.call("resource.plan",bad,nullptr,false);}
        if(profile.at("operation")=="tunnels.config.update") {bad=input(profile);bad["parameters"]["body"]["config"]["ingress"][0]["service"]="ssh://localhost:22";h.call("resource.plan",bad,nullptr,false);}
    }
}

}
int main(int argc, char** argv) {
    try { NT_REQUIRE(argc == 2);auto exe=fs::canonical(argv[1]);closed_frames(exe);descriptor_projection(exe);lifecycle(exe);integrity(exe);context_and_no_unqualified_execution(exe);reconciliation(exe);store_and_plan_substitution(exe);resource_planning(exe);std::cout << "8 native contract groups passed (strict controls, discovery, lifecycle, integrity, context, recovery, substitution, 23 typed resource profiles)\n";return 0; }
    catch(const std::exception& e) { std::cerr << e.what() << '\n';return 1; }
}
