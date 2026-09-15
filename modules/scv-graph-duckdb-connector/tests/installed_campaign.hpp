#pragma once
#include "connector_test_support.hpp"
#include <random>

namespace scv_test {
inline std::string uuid() {
  std::random_device random;
  unsigned char bytes[16];
  for (auto &b : bytes)
    b = static_cast<unsigned char>(random());
  bytes[6] = (bytes[6] & 15) | 64;
  bytes[8] = (bytes[8] & 63) | 128;
  std::ostringstream out;
  out << std::hex << std::setfill('0');
  for (std::size_t i = 0; i < 16; ++i) {
    if (i == 4 || i == 6 || i == 8 || i == 10)
      out << '-';
    out << std::setw(2) << static_cast<unsigned>(bytes[i]);
  }
  return out.str();
}
struct RestorePath {
  fs::path original, hidden;
  RestorePath(fs::path from, fs::path to)
      : original(std::move(from)), hidden(std::move(to)) {
    fs::rename(original, hidden);
  }
  ~RestorePath() {
    std::error_code ec;
    fs::rename(hidden, original, ec);
  }
};
struct Campaign {
  nt::Arguments args;
  std::string connector_version, tops, namespace_name = "research-a",
                                       time = "2026-09-13T05:00:00Z";
  fs::path qxctl, connector_prefix, owner_prefix, out, root;
  J graph, calls = array(), assertions = array(), summary;
  explicit Campaign(const nt::Arguments &value)
      : args(value),
        connector_version(args.get("connector-version", "0.1.0-dev")),
        tops(uuid()), qxctl(fs::canonical(args.require("qxctl"))),
        connector_prefix(fs::canonical(args.require("connector-prefix"))),
        owner_prefix(fs::canonical(args.require("owner-prefix"))),
        out(fs::absolute(args.require("out"))), root(out / "index"),
        graph(nt::read_json(args.require("graph"))) {
    NT_REQUIRE(!fs::exists(out));
    private_dir(out);
    private_dir(root);
    out = fs::canonical(out);
    root = out / "index";
    summary = {{"status", "running"},
               {"qxctl", qxctl.string()},
               {"qxctl_sha256", nt::digest(nt::read(qxctl)).substr(7)},
               {"graph_digest", graph.at("digest")},
               {"tops_id", tops},
               {"network_requests", 0},
               {"selected_head_mutations", 0},
               {"recovery_scope",
                "Committed native prepare, process exit, new-process qxctl "
                "recovery; crash tests are separate."},
               {"query_times", "Explicit simulated valid/expired times against "
                               "retained evidence, not refreshed retrievals."}};
    save();
  }
  virtual ~Campaign() = default;
  void save() {
    nt::write_json(out / "SUMMARY.json",
                   nt::updated(summary, {{"calls", calls.size()},
                                         {"assertions", assertions.size()}}));
    nt::write_json(out / "COMMANDS.json", calls);
    nt::write_json(out / "ASSERTIONS.json", assertions);
  }
  void check(bool value, const std::string &explanation) {
    nt::require(value, explanation);
    assertions.push_back(explanation);
    save();
  }
  J execute(const std::string &name, const std::vector<std::string> &command,
            const J &payload = nullptr, const fs::path &cwd = {},
            bool okay = true) {
    auto p = nt::run(command, payload.is_null() ? "" : nt::canonical(payload),
                     45, {}, cwd);
    nt::write(out / (name + "-stdout.json"), p.stdout_text);
    nt::write(out / (name + "-stderr.txt"), p.stderr_text);
    calls.push_back({{"name", name},
                     {"command", command},
                     {"cwd", cwd.empty() ? J(nullptr) : J(cwd.string())},
                     {"exit_code", p.returncode},
                     {"expected_success", okay}});
    save();
    nt::require((p.returncode == 0) == okay,
                name + ": " + std::to_string(p.returncode) + " " +
                    p.stdout_text.substr(0, 1200) + " " +
                    p.stderr_text.substr(0, 800));
    auto value = J::parse(p.stdout_text);
    if (okay) {
      if (value.value("protocol", std::string()) ==
          "symphony.knowledge.engine-process.v1")
        return value.at("result");
      NT_REQUIRE(value == nt::seal(value));
    }
    return value;
  }
  J qx(const std::string &name, const std::string &leaf,
       const J &value = nullptr, const J &options = object()) {
    std::vector<std::string> command = {qxctl.string(),
                                        "scv",
                                        "graph-index",
                                        leaf,
                                        "--connector-prefix",
                                        connector_prefix.string(),
                                        "--connector-version",
                                        connector_version,
                                        "--json"};
    if (leaf != "inspect")
      command.insert(command.end(),
                     {"--index-root", options.value("root", root.string()),
                      "--tops-id", options.value("tops", tops), "--namespace",
                      options.value("namespace", namespace_name)});
    if (!value.is_null()) {
      auto path = out / (name + "-input.json");
      nt::write(path, nt::canonical(value));
      command.insert(command.end(), {"--input", path.string()});
    }
    if (leaf == "status" || leaf == "recover") {
      auto split = name.find("__");
      command.insert(command.end(),
                     {"--operation-id", split == std::string::npos
                                            ? name
                                            : name.substr(split + 2)});
    }
    if (leaf == "import")
      command.insert(command.end(),
                     {"--domain", "scv", "--prefix",
                      options.value("owner", owner_prefix.string()),
                      "--version", "0.10.0-dev"});
    return execute(name, command, nullptr, {}, options.value("ok", true));
  }
  J query(const std::string &name, const J &snapshot,
          const J &options = object()) {
    return qx(name, "query",
              {{"snapshot_digest", snapshot},
               {"kind", options.value("kind", "nodes")},
               {"filters", options.value("filters", object())},
               {"cursor", options.value("cursor", J(nullptr))},
               {"limit", options.value("limit", 2)},
               {"query_time", options.value("query_time", time)}},
              options);
  }
  J native(const std::string &name, const std::string &operation,
           const J &payload, const J &installation, bool okay = true) {
    auto request =
        nt::request(installation.at("EngineID"), operation, payload, uuid());
    request["correlation_id"] = uuid();
    nt::write(out / (name + "-request.json"), nt::canonical(request));
    return execute(name, {installation.at("ExecutablePath").get<std::string>()},
                   request, root, okay);
  }
  fs::path clone_owner() {
    auto dest = out / "original-owner-copy";
    NT_REQUIRE(!fs::exists(dest));
    private_dir(dest);
    fs::path relative =
        "share/symphony/receipts/scv-engine/0.10.0-dev/install-receipt.json";
    auto receipt = nt::read_json(owner_prefix / relative);
    J paths = array();
    for (const auto &item : receipt.at("files"))
      paths.push_back(item.at("path"));
    paths.push_back(relative.string());
    for (const auto &item : paths) {
      fs::path path = item.get<std::string>();
      fs::create_directories((dest / path).parent_path());
      fs::copy_file(owner_prefix / path, dest / path);
      fs::permissions(dest / path,
                      fs::status(owner_prefix / path).permissions());
      fs::last_write_time(dest / path,
                          fs::last_write_time(owner_prefix / path));
    }
    return dest;
  }
  void test_installed_graph_index_recovery_and_retrieval() {
    qx("inspect", "inspect");
    J source = {{"operation_id", "initial-index"},
                {"graph", graph},
                {"query_time", time}};
    auto first = qx("import-initial", "import", source),
         record = first["connector_result"],
         snapshot = record["intent"]["snapshot"], sd = snapshot["digest"];
    check(snapshot["backend"] == "duckdb" && snapshot["graph"] == graph,
          "Default backend is reported as DuckDB and complete native graph is "
          "preserved");
    check(record["state"] == "committed" && record["index_verified"] == true,
          "Import reports only complete committed index");
    check(first["owner_evaluation"]["graph_digest"] == graph["digest"],
          "Import binds native semantic evaluation to retained graph");
    check(qx("import-retry", "import", source)["connector_result"] == record,
          "Exact operation retry preserves logical record and snapshot");
    qx("operation-reuse-conflict", "import",
       nt::updated(source, {{"query_time", "2026-09-13T05:00:01Z"}}),
       {{"ok", false}});
    check(qx("status__initial-index", "status")["owner_evaluation"].is_null(),
          "Status does not execute semantic owner");
    auto p1 = query("nodes-page-1", sd)["connector_result"],
         p2 = query("nodes-page-2", sd,
                    {{"cursor", p1["next_cursor"]}})["connector_result"],
         expected = graph["native_nodes"];
    std::sort(expected.begin(), expected.end(), [](const J &a, const J &b) {
      return a["node_id"].get<std::string>() < b["node_id"].get<std::string>();
    });
    J got = array(), first_four = array();
    for (const auto &page : J::array({p1, p2}))
      for (const auto &r : page["rows"])
        got.push_back(r["value"]);
    for (std::size_t i = 0; i < std::min<std::size_t>(4, expected.size()); ++i)
      first_four.push_back(expected[i]);
    check(got == first_four, "Native indexed pagination equals original native "
                             "node values and byte order");
    check(p1["matched_count"] == expected.size(),
          "Indexed query reports complete matched count before pagination");
    query("cursor-other-query", sd,
          {{"filters", {{"kind", expected[0]["kind"]}}},
           {"cursor", p1["next_cursor"]},
           {"ok", false}});
    query("cursor-missing-key", sd,
          {{"cursor",
            {{"query_digest", p1["next_cursor"]["query_digest"]},
             {"after_key", "absent"}}},
           {"ok", false}});
    auto last = query(
        "nodes-last-page", sd,
        {{"cursor",
          {{"query_digest", p1["next_cursor"]["query_digest"]},
           {"after_key", expected.at(expected.size() -
                                     2)["node_id"]}}}})["connector_result"];
    check(last["rows"].size() == 1 && last["next_cursor"].is_null(),
          "Last page has explicit null continuation");
    auto claim = graph["claims"][0];
    auto claims =
        query("claim-exact-scope", sd,
              {{"kind", "claims"}, {"filters", {{"scope", claim["scope"]}}}});
    check(claims["connector_result"]["rows"][0]["value"] == claim,
          "Claim retrieval preserves exact scope, supports and dependencies");
    auto empty = query("claim-absence", sd,
                       {{"kind", "claims"},
                        {"filters", {{"subject", "caller-absent-subject"}}}});
    check(empty["connector_result"]["matched_count"] == 0 &&
              empty["owner_evaluation"] == first["owner_evaluation"],
          "Structural absence does not replace complete native semantic "
          "evaluation");
    auto edge = graph["native_edges"][0],
         edges = query("edge-exact-filter", sd,
                       {{"kind", "edges"},
                        {"filters", edge},
                        {"limit", 128}})["connector_result"];
    check(edges["rows"] == J::array({{{"key", digest(edge)}, {"value", edge}}}),
          "Structural edge is preserved with canonical identity");
    auto expired =
        query("explicit-expired-evidence", sd,
              {{"kind", "claims"}, {"query_time", "2026-09-15T05:00:00Z"}});
    check(expired["connector_result"]["snapshot"] == snapshot &&
              expired["owner_evaluation"] != first["owner_evaluation"],
          "Explicit later evaluation changes native freshness result without "
          "renewing or rewriting snapshot");
    check(qx("export-original", "export",
             {{"snapshot_digest", sd},
              {"query_time", time}})["connector_result"]["snapshot"] ==
              snapshot,
          "Export retains exact graph and validating installations");
    query("namespace-hidden", sd, {{"namespace", "research-b"}, {"ok", false}});
    qx("tops-hidden__initial-index", "status", nullptr,
       {{"tops", uuid()}, {"ok", false}});
    auto other = qx("namespace-import", "import", source,
                    {{"namespace",
                      "research-b"}})["connector_result"]["intent"]["snapshot"];
    check(other["digest"] != sd && other["graph"] == snapshot["graph"],
          "Namespace changes index identity without changing native graph "
          "identity");
    J input = {{"tops_id", tops},
               {"namespace", "pending-recovery"},
               {"operation_id", "pending-index"},
               {"graph", graph},
               {"owner", snapshot["owner"]},
               {"connector", snapshot["connector"]},
               {"query_time", time}};
    auto prepared = native("native-prepare-and-exit", "prepare", input,
                           snapshot["connector"]);
    check(prepared["state"] == "prepared" &&
              prepared["index_verified"] == false,
          "Durable prepare exists independently of published snapshot");
    query("unpublished-snapshot", prepared["snapshot_digest"],
          {{"namespace", "pending-recovery"}, {"ok", false}});
    auto pending = qx("pending-status__pending-index", "status", nullptr,
                      {{"namespace", "pending-recovery"}});
    check(pending["connector_result"] == prepared,
          "New process observes exact durable prepared intent");
    auto recovered = qx("recover__pending-index", "recover", nullptr,
                        {{"namespace", "pending-recovery"}});
    check(recovered["connector_result"]["state"] == "committed" &&
              recovered["connector_result"]["intent"] == prepared["intent"],
          "qxctl recovery replays retained owner and commits original intent "
          "exactly");
    check(qx("recover-again__pending-index", "recover", nullptr,
             {{"namespace", "pending-recovery"}})["connector_result"] ==
              recovered["connector_result"],
          "Repeated recovery produces no replacement snapshot");
    auto owner = clone_owner();
    auto owned = qx("owner-copy-import", "import",
                    nt::updated(source, {{"operation_id", "owner-copy"}}),
                    {{"owner", owner.string()}}),
         copied = owned["connector_result"]["intent"]["snapshot"];
    fs::path executable = copied["owner"]["ExecutablePath"].get<std::string>();
    {
      RestorePath restore(executable, executable.string() + ".offline");
      qx("missing-owner-status__owner-copy", "status");
      query("missing-owner-query", copied["digest"], {{"ok", false}});
      qx("missing-owner-export", "export",
         {{"snapshot_digest", copied["digest"]}, {"query_time", time}},
         {{"ok", false}});
      qx("missing-owner-recover__owner-copy", "recover", nullptr,
         {{"ok", false}});
    }
    check(true, "Missing exact owner permits index status but blocks semantic "
                "query, export and recovery");
    summary.update(J{{"status", "passed"},
                     {"snapshot_digest", sd},
                     {"snapshot_counts", record["counts"]},
                     {"original_owner", snapshot["owner"]},
                     {"connector", snapshot["connector"]}});
    save();
  }
  template <class F> void checked_run(F run) {
    try {
      run();
    } catch (...) {
      summary["status"] = "failed";
      save();
      throw;
    }
  }
  void print() const {
    std::cout << J{{"status", "passed"},
                   {"calls", calls.size()},
                   {"assertions", assertions.size()}}
                     .dump()
              << "\n";
  }
};
} // namespace scv_test
