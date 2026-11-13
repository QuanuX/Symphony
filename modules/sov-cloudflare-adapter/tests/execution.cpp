#include "execution_fixture.hpp"
#include <atomic>

namespace {
cloudflare::HTTPResult response(const Json &data) {
  return {true, 200, {{"success", true}, {"result", data}}, true};
}
void no_observation(const Json &out) {
  NT_REQUIRE(out.at("data").at("observation").is_null() &&
             out.at("data").at("attempt").at("observation_ref").is_null());
}
void delete_recovery_authority(const fs::path &provider, const fs::path &self) {
  Harness h(provider, self);
  std::atomic<bool> allow_collection{false};
  Authority authority(h.temp.path / "a.sock", 4, [&](const Json &intent) {
    if (intent.at("operation") == "dns.records.list")
      return allow_collection.load() && intent.at("resource_id").is_null();
    return (intent.at("operation") == "dns.records.get" ||
            intent.at("operation") == "dns.records.delete") &&
           intent.at("resource_id") == std::string(32, '1');
  });
  auto row = h.input("dns.records.update").at("parameters").at("body");
  row["id"] = std::string(32, '1');
  unsigned calls = 0, mutations = 0;
  bool deleted = false;
  RuntimeHooks hooks;
  hooks.http = [&](const Json &preview, const std::string &, const fs::path &,
                   const fs::path &, std::int64_t) {
    ++calls;
    const auto &request = preview.at("request");
    if (request.at("method") == "DELETE") {
      ++mutations;
      deleted = true;
      throw std::runtime_error("reply lost after delete");
    }
    if (request.at("command").get<std::string>().ends_with(" list")) {
      NT_REQUIRE(deleted && allow_collection.load());
      return response(Json::array());
    }
    return response(row);
  };
  auto before = h.run(h.input("dns.records.get"), hooks);
  auto remove = h.input("dns.records.delete");
  remove["expected_observation"] = before.at("data").at("attempt").at("observation_ref");
  auto uncertain = h.run(remove, hooks);
  NT_REQUIRE(uncertain.at("result").at("effect_state") == "unresolved" &&
             mutations == 1);
  const auto before_denial = calls;
  auto denied = h.run(remove, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(denied.at("result").at("effect_state") == "unresolved" &&
             calls == before_denial && mutations == 1);
  allow_collection = true;
  auto accepted = h.run(remove, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(accepted.at("result").at("effect_state") == "applied_verified" &&
             calls == before_denial + 1 && mutations == 1);
  authority.check();
  NT_REQUIRE(authority.intents.size() == 4);
  for (std::size_t i : {2U, 3U}) {
    NT_REQUIRE(authority.intents[i].at("operation") == "dns.records.list" &&
               authority.intents[i].at("resource_id").is_null() &&
               authority.intents[i].at("account_id") == h.context.at("account_id") &&
               authority.intents[i].at("zone_id") == h.context.at("zone_id"));
  }
}
void object_identity(const fs::path &provider, const fs::path &self) {
  Harness h(provider, self);
  Authority authority(h.temp.path / "a.sock", 6);
  auto current = h.input("dns.records.update").at("parameters").at("body");
  current["id"] = std::string(32, '1');
  bool wrong = true, wrong_after_mutation = false;
  unsigned mutations = 0;
  RuntimeHooks hooks;
  hooks.http = [&](const Json &preview, const std::string &, const fs::path &,
                   const fs::path &, std::int64_t) {
    const auto &request = preview.at("request");
    if (request.at("method") != "GET") {
      ++mutations;
      current = request.at("body");
      current["id"] = std::string(32, '1');
      wrong = wrong_after_mutation;
      return response(current);
    }
    auto observed = current;
    if (wrong)
      observed["id"] = std::string(32, '2');
    return response(observed);
  };
  auto bad_read = h.run(h.input("dns.records.get"), hooks);
  NT_REQUIRE(bad_read.at("result").at("outcome") == "failed");
  no_observation(bad_read);
  wrong = false;
  auto before = h.run(h.input("dns.records.get"), hooks);
  auto update = h.input("dns.records.update");
  update["parameters"]["body"]["content"] = "changed.example.test";
  update["expected_observation"] = before.at("data").at("attempt").at("observation_ref");
  wrong = true;
  auto bad_precondition = h.run(update, hooks);
  NT_REQUIRE(bad_precondition.at("result").at("effect_state") == "not_applied" &&
             mutations == 0);
  no_observation(bad_precondition);
  wrong = false;
  wrong_after_mutation = true;
  auto uncertain = h.run(update, hooks);
  NT_REQUIRE(uncertain.at("result").at("effect_state") == "unresolved" &&
             mutations == 1);
  no_observation(uncertain);
  auto bad_recovery = h.run(update, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(bad_recovery.at("result").at("effect_state") == "unresolved" &&
             mutations == 1);
  no_observation(bad_recovery);
  wrong = false;
  auto recovered = h.run(update, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(recovered.at("result").at("effect_state") == "applied_verified" &&
             recovered.at("data").at("observation").at("snapshot").at("data").at("id") ==
                 update.at("parameters").at("resource_id") &&
             mutations == 1);
  authority.check();
}
void created_identity(const fs::path &provider, const fs::path &self) {
  Harness h(provider, self);
  Authority authority(h.temp.path / "a.sock", 3);
  Json created = nullptr;
  bool wrong = true;
  unsigned mutations = 0;
  RuntimeHooks hooks;
  hooks.http = [&](const Json &preview, const std::string &, const fs::path &,
                   const fs::path &, std::int64_t) {
    const auto &request = preview.at("request");
    if (request.at("method") == "POST") {
      ++mutations;
      created = request.at("body");
      created["id"] = std::string(32, '1');
      return response(created);
    }
    if (request.at("command").get<std::string>().ends_with(" list"))
      return response(Json::array());
    auto observed = created;
    if (wrong)
      observed["id"] = std::string(32, '2');
    return response(observed);
  };
  auto before = h.run(h.input("dns.records.list"), hooks);
  auto create = h.input("dns.records.create");
  create["expected_observation"] = before.at("data").at("attempt").at("observation_ref");
  auto uncertain = h.run(create, hooks);
  NT_REQUIRE(uncertain.at("result").at("effect_state") == "unresolved" &&
             uncertain.at("data").at("attempt").at("resource_id") == std::string(32, '1') &&
             mutations == 1);
  no_observation(uncertain);
  wrong = false;
  auto recovered = h.run(create, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(recovered.at("result").at("effect_state") == "applied_verified" &&
             mutations == 1);
  authority.check();
}
void configuration_identity(const fs::path &provider, const fs::path &self) {
  Harness h(provider, self);
  Authority authority(h.temp.path / "a.sock", 6);
  auto update = h.input("tunnels.config.update");
  Json current = update.at("parameters").at("body");
  const auto tunnel = update.at("parameters").at("resource_id");
  bool identities = false, wrong_tunnel = false, wrong_account = false;
  unsigned mutations = 0;
  RuntimeHooks hooks;
  hooks.http = [&](const Json &preview, const std::string &, const fs::path &,
                   const fs::path &, std::int64_t) {
    const auto &request = preview.at("request");
    if (request.at("method") == "PUT") {
      ++mutations;
      current = request.at("body");
      wrong_tunnel = true;
      return response(current);
    }
    auto observed = current;
    if (identities) {
      observed["tunnel_id"] = wrong_tunnel
          ? Json("22222222-2222-4222-8222-222222222222") : tunnel;
      observed["account_id"] = wrong_account
          ? Json(std::string(32, '3')) : h.context.at("account_id");
    }
    return response(observed);
  };
  // Configuration responses have no generic id. The endpoint's optional
  // identities may be absent, but contradictory identities are not accepted.
  auto omitted = h.run(h.input("tunnels.config.get"), hooks);
  NT_REQUIRE(omitted.at("result").at("outcome") == "completed");
  identities = true;
  auto before = h.run(h.input("tunnels.config.get"), hooks);
  NT_REQUIRE(before.at("result").at("outcome") == "completed");
  wrong_tunnel = true;
  auto bad_tunnel = h.run(h.input("tunnels.config.get"), hooks);
  NT_REQUIRE(bad_tunnel.at("result").at("outcome") == "failed");
  no_observation(bad_tunnel);
  wrong_tunnel = false;
  wrong_account = true;
  auto bad_account = h.run(h.input("tunnels.config.get"), hooks);
  NT_REQUIRE(bad_account.at("result").at("outcome") == "failed");
  no_observation(bad_account);
  wrong_account = false;
  update["expected_observation"] = before.at("data").at("attempt").at("observation_ref");
  update["parameters"]["body"]["config"]["ingress"].insert(
      update["parameters"]["body"]["config"]["ingress"].begin(),
      Json{{"hostname", "ssh.example.invalid"}, {"service", "ssh://localhost:22"}});
  auto uncertain = h.run(update, hooks);
  NT_REQUIRE(uncertain.at("result").at("effect_state") == "unresolved" &&
             mutations == 1);
  no_observation(uncertain);
  wrong_tunnel = false;
  auto recovered = h.run(update, hooks, "resource.reconcile", uncertain.at("plan"));
  NT_REQUIRE(recovered.at("result").at("effect_state") == "applied_verified" &&
             mutations == 1);
  authority.check();
}
} // namespace

int main(int argc, char **argv) {
  try {
    NT_REQUIRE(argc == 2);
    const auto provider = fs::canonical(argv[1]);
    const auto self = n::executable(::getpid());
    delete_recovery_authority(provider, self);
    object_identity(provider, self);
    created_identity(provider, self);
    configuration_identity(provider, self);
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
        << "Native credential/export/import and resource execution: exact "
           "recovery permission, object/configuration identity, mutation "
           "readback, replay refusal, stale/scope refusal, pagination and "
           "secret exclusion passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
