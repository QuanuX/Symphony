#include "adapter.hpp"
#include "native_test.hpp"
using namespace native_test;
int main(int argc, char** argv) {
    try {
        NT_REQUIRE(argc == 4);
        auto qxctl=fs::canonical(argv[1]), prefix=fs::canonical(argv[2]), bundle_path=fs::canonical(argv[3]);
        TempDir temp("cf-qualified-");auto store=temp.path/"store",project=temp.path/"project";
        write(project/".env","CLOUDFLARE_API_TOKEN=SECRET-SENTINEL\n");
        write(project/"node_modules/cf/package.json","{\"name\":\"cf\",\"version\":\"99.0.0\",\"bin\":\"evil.js\"}");
        write(project/"node_modules/cf/evil.js","console.log('PROJECT-DELEGATION-SENTINEL'); process.exit(0)");
        std::map<std::string,std::string> environment{{"CLOUDFLARE_API_TOKEN","SECRET-SENTINEL"},{"CLOUDFLARE_ACCOUNT_ID","ambient"},{"NODE_OPTIONS","--invalid-ambient-option"},{"CF_DELEGATION","0"},{"HTTPS_PROXY","http://127.0.0.1:9"}};
        auto invoke=[&](const std::string& family,const std::string& leaf,const Json& input,bool ok=true,const std::string& format="json") {
            auto path=temp.path/"input.json";write_json(path,input);
            auto result=native_test::run({qxctl.string(),"sov","cloudflare","cf",family,leaf,"--adapter-prefix",prefix.string(),"--adapter-version","0.1.0-dev","--input",path.string(),"--format",format},"",120,environment,project);
            NT_REQUIRE(result.returncode==(ok?0:5));NT_REQUIRE(result.stdout_text.find("SENTINEL")==std::string::npos && result.stderr_text.find("SENTINEL")==std::string::npos);
            return format=="json"?Json::parse(result.stdout_text):Json(result.stdout_text);
        };
        Json checks=Json::array();
        auto catalogue=invoke("capability","list",{{"store",nullptr},{"toolchain_id",nullptr},{"query",nullptr}});auto owner=catalogue.at("data").at("owner");
        auto adapter=prefix/"libexec/symphony/sov-cloudflare-adapter/0.1.0-dev/symphony-sov-cloudflare-adapter";
        NT_REQUIRE(owner==Json::parse(native_test::run({adapter.string(),"--describe"}).stdout_text));checks.push_back("native/qxctl owner parity");
        auto text=invoke("capability","describe",{{"operation","resource.list"}},true,"text").get<std::string>();NT_REQUIRE(text.find("unavailable_contract")!=std::string::npos && text.find("qualification")!=std::string::npos);checks.push_back("complete safe text presentation");
        auto bundle=read_json(bundle_path/"bundle.json");auto id=cloudflare::digest(bundle,"symphony.sov.cloudflare-bundle.v1");
        NT_REQUIRE(std::find(owner.at("qualified_offline_toolchains").begin(),owner.at("qualified_offline_toolchains").end(),Json(id))!=owner.at("qualified_offline_toolchains").end());
        auto plan=invoke("installation","plan",{{"store",store.string()},{"action","install"},{"bundle_path",bundle_path.string()},{"bundle_digest",id},{"installation_id",nullptr},{"expected_state",nullptr}});
        auto pd=plan.at("data").at("plan_digest");NT_REQUIRE(pd==cloudflare::digest(plan.at("data").at("plan"),"symphony.sov.cloudflare-plan.v1"));
        auto applied=invoke("installation","apply",{{"store",store.string()},{"plan_digest",pd}});NT_REQUIRE(applied.at("effect_state")=="applied_verified" && applied.at("data").at("state").at("selected").is_null());checks.push_back("exact real cf/Node import and plan digest");
        NT_REQUIRE(invoke("installation","inspect",{{"store",store.string()},{"installation_id",id}}).at("data").at("bundle")==bundle);checks.push_back("complete installed dependency inventory");
        auto found=invoke("capability","list",{{"store",store.string()},{"toolchain_id",id},{"query","DNS records"}});
        NT_REQUIRE(found.at("dispatch_state")=="response_received" && found.at("data").at("completeness")=="bounded_search_not_inventory" && found.at("data").at("vendor_result").size()<=5);
        bool dns=false;for(const auto& row:found.at("data").at("vendor_result")) if(row.at("command")=="cf dns records list") dns=true;NT_REQUIRE(dns);checks.push_back("real CLI search with hostile ambient settings");
        auto schema=invoke("capability","schema",{{"operation","tunnels.list"},{"kind","vendor"},{"store",store.string()},{"toolchain_id",id}});bool matched=false;
        for(const auto& candidate:owner.at("vendor_candidates")) if(candidate.at("operation")=="tunnels.list") matched=schema.at("data").at("vendor_result")==candidate.at("schema");NT_REQUIRE(matched);checks.push_back("real CLI schema matches C0 expectation");
        auto package=store/"packages"/id.substr(7);
        for(const auto* relative:{"cf/node_modules/cf/package.json","node/bin/node"}) {
            auto path=package/relative;auto original=read(path);write(path,original+"x");
            auto rejected=invoke("capability","list",{{"store",store.string()},{"toolchain_id",id},{"query","DNS"}},false);
            NT_REQUIRE(rejected.at("error_code")=="integrity_failure" && rejected.at("dispatch_state")=="not_dispatched");write(path,original);
        }
        checks.push_back("dependency/runtime drift refused before child dispatch");
        Json context{{"host_ref","host:qualification"},{"principal_uid",std::to_string(::getuid())},{"account_id",std::string(32,'0')},{"zone_id",nullptr},{"credential_ref",nullptr},{"toolchain_id",id}};
        auto proposal=invoke("context","plan",{{"store",store.string()},{"context",context},{"expected_context_ref",nullptr}});
        NT_REQUIRE(proposal.at("data").at("authority_readiness")=="unavailable_contract" && proposal.at("data").at("context_ref")==cloudflare::digest(context,"symphony.sov.cloudflare-context.v1"));checks.push_back("exact context proposal without provider authority");
        NT_REQUIRE(fs::is_empty(store/"staging") && !fs::exists(project/".cloudflare"));checks.push_back("probe cleanup without ambient project preparation");
        std::cout<<Json{{"qualification","offline_real_cli_only"},{"bundle_digest",id},{"checks",checks},{"passed",checks.size()},{"provider_calls",0},{"ssh_connections",0}}.dump(2)<<'\n';return 0;
    } catch(const std::exception& e) { std::cerr<<e.what()<<'\n';return 1; }
}
