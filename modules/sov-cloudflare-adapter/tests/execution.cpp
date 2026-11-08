#include "execution_fixture.hpp"
int main(int argc, char **argv) {
  try {
    NT_REQUIRE(argc == 2);
    Harness h(fs::canonical(argv[1]), n::executable(::getpid()));
    Authority authority(h.temp.path / "a.sock", 12);
    int calls = 0;
    Json rows = Json::array();
    bool fail = false;
    RuntimeHooks hooks;
    hooks.http = [&](const Json &p, const std::string &token, const fs::path &,
                     const fs::path &, std::int64_t) {
      NT_REQUIRE(token == "SECRET-SENTINEL");
      ++calls;
      auto req = p.at("request");
      auto method = req.at("method");
      if (fail)
        throw std::runtime_error("lost response");
      Json result = rows;
      if (method == "POST") {
        auto row = req.at("body");
        row["id"] = std::string(32, '1');
        rows.push_back(row);
        result = row;
      } else if (method == "GET" &&
                 !req.at("command").get<std::string>().ends_with(" list"))
        result = rows.at(0);
      return cloudflare::HTTPResult{true, 200,
                                    Json{{"success", true},
                                         {"result", result},
                                         {"result_info",
                                          {{"page", 1},
                                           {"total_count", rows.size()},
                                           {"total_pages", 1}}}},
                                    true};
    };
    auto listing = h.input("dns.records.list");
    auto initial = h.run(listing, hooks);
    NT_REQUIRE(initial.at("result").at("outcome") == "completed");
    n::put(h.root / "resource-attempts" /
               (std::string(64, 'f') + ".json.new-999"),
           "unpublished atomic temporary");
    auto create = h.input("dns.records.create");
    create["expected_observation"] =
        initial.at("data").at("attempt").at("observation_ref");
    auto made = h.run(create, hooks);
    NT_REQUIRE(made.at("result").at("effect_state") == "applied_verified" &&
               calls == 4);
    bool replay = false;
    try {
      h.run(create, hooks, "resource.apply", made.at("plan"));
    } catch (...) {
      replay = true;
    }
    NT_REQUIRE(replay && calls == 4);
    auto stale = h.run(create, hooks);
    NT_REQUIRE(stale.at("result").at("effect_state") == "not_applied" &&
               calls == 5);
    auto read = h.run(listing, hooks);
    auto edit = h.input(
        "dns.records.update"); // Require a resource-scoped prior observation,
                               // not collection observation.
    edit["expected_observation"] =
        read.at("data").at("attempt").at("observation_ref");
    auto wrongscope = h.run(edit, hooks);
    NT_REQUIRE(wrongscope.at("result").at("effect_state") == "not_applied");
    // Pagination does not turn a bounded first page into complete inventory.
    hooks.http = [&](const Json &, const std::string &, const fs::path &,
                     const fs::path &, std::int64_t) {
      return cloudflare::HTTPResult{
          true,
          200,
          {{"success", true},
           {"result", rows},
           {"result_info",
            {{"page", 1}, {"total_pages", 2}, {"total_count", 2}}}},
          true};
    };
    listing["execution"]["max_pages"] = 1;
    listing["parameters"]["page"]["per_page"] = 1;
    auto partial = h.run(listing, hooks);
    NT_REQUIRE(
        partial.at("data").at("observation").at("coverage").at("complete") ==
        false);
    // Raw token output is imported by the native owner and never placed in
    // control JSON.
    hooks.http = [&](const Json &, const std::string &, const fs::path &,
                     const fs::path &, std::int64_t) {
      return cloudflare::HTTPResult{
          true, 200, {{"success", true}, {"result", "TOKEN-SENTINEL"}}, true};
    };
    auto token = h.input("tunnels.token.get");
    token["execution"]["result_ref"] = "ssiag-native:tunnel";
    auto imported = h.run(token, hooks);
    NT_REQUIRE(imported.at("data").at("credential_ref") ==
               "ssiag-native:tunnel");
    NT_REQUIRE(n::read(h.vault / "credentials/tunnel/value", 16384, true) ==
               "TOKEN-SENTINEL");
    // API error bodies and unknown provider fields never become public
    // artifacts.
    hooks.http = [&](const Json &, const std::string &, const fs::path &,
                     const fs::path &, std::int64_t) {
      return cloudflare::HTTPResult{
          true, 500, {{"message", "SECRET-SENTINEL"}}, false};
    };
    auto failed = h.run(h.input("dns.records.list"), hooks);
    NT_REQUIRE(failed.at("result").at("outcome") == "failed");
    hooks.http = [&](const Json &, const std::string &, const fs::path &,
                     const fs::path &, std::int64_t) {
      auto row = rows[0];
      row["private_field"] = "SECRET-SENTINEL";
      return cloudflare::HTTPResult{
          true, 200, {{"success", true}, {"result", Json::array({row})}}, true};
    };
    auto safe = h.run(h.input("dns.records.list"), hooks);
    NT_REQUIRE(safe.at("result").at("outcome") == "completed");
    int mutations = 0;
    bool lose = true;
    hooks.http = [&](const Json &p, const std::string &, const fs::path &,
                     const fs::path &, std::int64_t) {
      auto r = p.at("request");
      if (r.at("method") != "GET") {
        ++mutations;
        rows[0] = r.at("body");
        rows[0]["id"] = std::string(32, '1');
        if (lose)
          throw std::runtime_error("lost after effect");
      }
      return cloudflare::HTTPResult{
          true, 200, {{"success", true}, {"result", rows[0]}}, true};
    };
    auto before = h.run(h.input("dns.records.get"), hooks);
    auto change = h.input("dns.records.update");
    change["parameters"]["body"]["content"] = "changed.example.test";
    change["expected_observation"] =
        before.at("data").at("attempt").at("observation_ref");
    auto unknown = h.run(change, hooks);
    NT_REQUIRE(unknown.at("result").at("outcome") == "unknown" &&
               mutations == 1);
    lose = false;
    auto recovered =
        h.run(change, hooks, "resource.reconcile", unknown.at("plan"));
    NT_REQUIRE(recovered.at("result").at("effect_state") ==
                   "applied_verified" &&
               mutations == 1);
    authority.check();
    auto metadata = n::parse(
        n::read(h.vault / "credentials/api/metadata.json", 32768, true));
    metadata["expires_unix_ms"] = "0";
    n::atomic(h.vault / "credentials/api/metadata.json", metadata);
    auto revoked = h.run(h.input("dns.records.get"), hooks);
    NT_REQUIRE(revoked.at("result").at("effect_state") == "not_applied" &&
               mutations == 1);

    auto uncertain = imported.at("data").at("attempt");
    uncertain["credential_ref"] = nullptr;
    uncertain["outcome"] = "unknown";
    uncertain["effect_state"] = "unresolved";
    auto pd = imported.at("data").at("plan_digest").get<std::string>();
    n::atomic(h.root / "resource-attempts" / (pd.substr(7) + ".json"),
              uncertain);
    auto count_uses = [&] {
      return std::distance(fs::directory_iterator(h.vault / "uses"),
                           fs::directory_iterator{});
    };
    auto uses = count_uses();
    auto pending_import =
        h.run(token, hooks, "resource.reconcile", imported.at("plan"));
    NT_REQUIRE(pending_import.at("result").at("outcome") == "unknown" &&
               pending_import.at("result").at("effect_state") == "unresolved" &&
               count_uses() == uses);
    std::cout
        << "Native credential/export/import and resource execution: 12 "
           "admitted uses; mutation readback, replay refusal, stale/scope "
           "refusal, pagination and secret exclusion passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
