#include "maintenance_support.hpp"
#include <array>
#include <regex>
using namespace scv_maintenance;
namespace {
void private_json(const fs::path &path, const J &value) {
  fs::create_directories(path.parent_path());
  auto temporary = fs::path(path.string() + ".writing");
  std::string bytes = value.dump(2, ' ', false) + "\n";
  int fd = ::open(temporary.c_str(), O_WRONLY | O_CREAT | O_TRUNC | O_NOFOLLOW,
                  0600);
  nt::require(fd >= 0, "cannot write " + temporary.string());
  NT_REQUIRE(::fchmod(fd, 0600) == 0);
  std::size_t offset = 0;
  while (offset < bytes.size()) {
    auto n = ::write(fd, bytes.data() + offset, bytes.size() - offset);
    if (n < 0 && errno == EINTR)
      continue;
    if (n <= 0) {
      ::close(fd);
      throw std::runtime_error("short evidence write");
    }
    offset += static_cast<std::size_t>(n);
  }
  if (::fsync(fd) != 0) {
    ::close(fd);
    throw std::runtime_error("evidence fsync failed");
  }
  ::close(fd);
  fs::rename(temporary, path);
}
void private_directory(const fs::path &path) {
  fs::create_directories(path);
  NT_REQUIRE(::chmod(path.c_str(), 0700) == 0);
}
std::string uuid4() {
  std::array<unsigned char, 16> bytes{};
  int fd = ::open("/dev/urandom", O_RDONLY | O_CLOEXEC);
  NT_REQUIRE(fd >= 0);
  std::size_t count = 0;
  while (count < bytes.size()) {
    auto n = ::read(fd, bytes.data() + count, bytes.size() - count);
    if (n < 0 && errno == EINTR)
      continue;
    NT_REQUIRE(n > 0);
    count += static_cast<std::size_t>(n);
  }
  ::close(fd);
  bytes[6] = (bytes[6] & 15) | 64;
  bytes[8] = (bytes[8] & 63) | 128;
  std::ostringstream out;
  for (std::size_t i = 0; i < bytes.size(); ++i) {
    if (i == 4 || i == 6 || i == 8 || i == 10)
      out << '-';
    out << std::hex << std::setw(2) << std::setfill('0') << unsigned(bytes[i]);
  }
  return out.str();
}
bool canonical_uuid(const std::string &value) {
  static const std::regex pattern(
      "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}");
  return std::regex_match(value, pattern);
}
std::string trim(std::string value) {
  while (!value.empty() &&
         std::isspace(static_cast<unsigned char>(value.back())))
    value.pop_back();
  auto at = value.find_first_not_of(" \r\n\t");
  return at == std::string::npos ? "" : value.substr(at);
}
struct Harness {
  fs::path output, state_path, repo, cli, prefix;
  J state = nullptr;
  Harness(const fs::path &selected_output, const fs::path &selected_repo,
          const fs::path &selected_cli, const fs::path &selected_prefix)
      : output(fs::weakly_canonical(fs::absolute(selected_output))),
        state_path(output / "RUNTIME.json") {
    if (fs::exists(state_path))
      state = nt::read_json(state_path);
    auto select = [&](const std::string &key, const fs::path &explicit_value) {
      std::string stored = state.is_null() ? "" : state.value(key, "");
      nt::require(!explicit_value.empty() || !stored.empty(),
                  "explicit " + key +
                      " selection is required for a fresh specimen");
      auto value = fs::canonical(explicit_value.empty() ? fs::path(stored)
                                                        : explicit_value);
      nt::require(
          stored.empty() || explicit_value.empty() || value.string() == stored,
          "retained " + key + " selection differs from the explicit argument");
      return value;
    };
    repo = select("repository", selected_repo);
    cli = select("qxctl", selected_cli);
    prefix = select("scv_prefix", selected_prefix);
  }
  void save() { private_json(state_path, state); }
  std::map<std::string, std::string> environment() const {
    std::map<std::string, std::string> values;
    for (char **current = ::environ; current && *current; ++current) {
      std::string entry = *current;
      auto split = entry.find('=');
      if (split == std::string::npos)
        continue;
      auto name = entry.substr(0, split);
      if (name.starts_with("SYMPHONY_SSIAG_") ||
          name.starts_with("SYMPHONY_STAV_"))
        continue;
      values[name] = entry.substr(split + 1);
    }
    for (const auto &[key, value] : state.at("environment").items())
      values[key] = value.get<std::string>();
    const char *selected = std::getenv("GOCACHE");
    values["GOCACHE"] =
        selected ? selected : (output / "build/go-cache").string();
    return values;
  }
  J command(const std::string &label, const std::vector<std::string> &argv,
            const fs::path &selected_cwd = {}, bool expected_success = true,
            int timeout = 30) {
    auto log_path = output / "COMMANDS.json";
    auto commands = fs::exists(log_path) ? nt::read_json(log_path) : array();
    std::ostringstream stem;
    stem << std::setw(3) << std::setfill('0') << commands.size() + 1 << '-'
         << label;
    auto started = now();
    auto cwd = selected_cwd.empty() ? repo : selected_cwd;
    auto completed = nt::run(argv, "", timeout, environment(), cwd, true);
    auto stdout_path = output / (stem.str() + "-stdout.json"),
         stderr_path = output / (stem.str() + "-stderr.txt");
    nt::write(stdout_path, completed.stdout_text);
    nt::write(stderr_path, completed.stderr_text);
    NT_REQUIRE(::chmod(stdout_path.c_str(), 0600) == 0 &&
               ::chmod(stderr_path.c_str(), 0600) == 0);
    commands.push_back({{"label", label},
                        {"argv", argv},
                        {"cwd", cwd.string()},
                        {"started_at", started},
                        {"completed_at", now()},
                        {"exit_code", completed.returncode},
                        {"expected_success", expected_success},
                        {"stdout", stdout_path.string()},
                        {"stderr", stderr_path.string()},
                        {"stdout_digest", nt::digest(completed.stdout_text)},
                        {"stderr_digest", nt::digest(completed.stderr_text)}});
    private_json(log_path, commands);
    nt::require((completed.returncode == 0) == expected_success,
                label + ": unexpected exit " +
                    std::to_string(completed.returncode) + "; see " +
                    stderr_path.string());
    if (!expected_success)
      return {{"stdout", stdout_path.string()},
              {"stderr", stderr_path.string()},
              {"exit_code", completed.returncode}};
    try {
      return J::parse(completed.stdout_text);
    } catch (const J::parse_error &) {
      return completed.stdout_text;
    }
  }
  J qx(const std::string &label, std::vector<std::string> args,
       bool expected_success = true) {
    nt::require(sha(nt::read(cli)) ==
                    state.at("qxctl_sha256").get<std::string>(),
                "selected qxctl identity changed");
    args.insert(args.begin(), cli.string());
    return command(label, args, {}, expected_success);
  }
  void setup() {
    nt::require(!fs::exists(output), "setup requires a fresh output directory; "
                                     "use start for a retained setup");
    private_directory(output);
    std::string pattern =
        (fs::is_directory("/private/tmp") ? "/private/tmp/" : "/tmp/") +
        std::string("s12.XXXXXX");
    std::vector<char> buffer(pattern.begin(), pattern.end());
    buffer.push_back('\0');
    auto made = ::mkdtemp(buffer.data());
    nt::require(made != nullptr, "cannot allocate private runtime");
    fs::path runtime = fs::canonical(made);
    NT_REQUIRE(::chmod(runtime.c_str(), 0700) == 0);
    auto tops = uuid4();
    auto config = output / "config", state_root = output / "state",
         install_prefix = output / "installed";
    for (auto path : {config, state_root, install_prefix, output / "build",
                      output / "source-state"})
      private_directory(path);
    auto ssiag_config = config / "symphony" / tops / "ssiag/config.json",
         stav_config =
             config / "symphony" / tops / "stav/append-authority.json",
         ssiag_socket = runtime / "symphony" / tops / "ssiag/ssiag.sock",
         stav_socket = runtime / "symphony" / tops / "stav/append.sock",
         ledger = state_root / "symphony" / tops / "stav/ledger-v1.stavlog";
    auto uid = ::geteuid(), gid = ::getegid();
    state = {
        {"protocol", "local.scv.authority-runtime.v1"},
        {"created_at", now()},
        {"phase", "initializing"},
        {"tops_id", tops},
        {"domain", "schv-do"},
        {"source_id", "do-app-platform-limits"},
        {"locator_id", "docs"},
        {"source_root", (output / "source-state").string()},
        {"scv_prefix", prefix.string()},
        {"scv_version", "0.10.0-dev"},
        {"repository", repo.string()},
        {"qxctl", cli.string()},
        {"qxctl_sha256", sha(nt::read(cli))},
        {"uid", uid},
        {"gid", gid},
        {"subject_id", "scv-maintenance-owner"},
        {"environment",
         {{"XDG_CONFIG_HOME", config.string()},
          {"XDG_STATE_HOME", state_root.string()},
          {"XDG_RUNTIME_DIR", runtime.string()},
          {"SYMPHONY_STAV_CONFIG", stav_config.string()}}},
        {"paths",
         {{"prefix", install_prefix.string()},
          {"ssiag_config", ssiag_config.string()},
          {"stav_config", stav_config.string()},
          {"ssiag_socket", ssiag_socket.string()},
          {"stav_socket", stav_socket.string()},
          {"ledger", ledger.string()}}},
        {"services", object()},
        {"process_history", array()},
        {"scope",
         "Fresh same-UID diagnostic services using real kernel-peer "
         "authentication and real STAV storage. No existing host service, "
         "account, provider, keyring, grant or source state is changed. This "
         "is not distinct-account isolation or production supervision."}};
    save();
    private_json(
        ssiag_config,
        {{"schema", "symphony.ssiag.config.v1"},
         {"mode", "user"},
         {"tops", {{"id", tops}, {"name", "SCV source maintenance specimen"}}},
         {"listen", {{"network", "unix"}, {"address", ssiag_socket.string()}}},
         {"authentication",
          {{"mechanism", "unix_peer_credentials"},
           {"service",
            {{"id", "symphony.ssiag.service"},
             {"kind", "symphony.identity.service"},
             {"uid", uid},
             {"gid", gid}}},
           {"subjects", J::array({{{"id", state["subject_id"]},
                                   {"kind", "symphony.identity.operator"},
                                   {"uid", uid},
                                   {"gid", gid}}})}}},
         {"authorization",
          {{"default_effect", "deny"},
           {"max_capability_seconds", 60},
           {"grants", array()}}},
         {"providers", array()}});
    J stav_authority = {
        {"uid", uid},
        {"gid", gid},
        {"subject",
         {{"id", "stav-authority"}, {"kind", "symphony.identity.service"}}}};
    J stav_producer = {
        {"uid", uid},
        {"gid", gid},
        {"subject",
         {{"id", "ssiag-service"}, {"kind", "symphony.identity.service"}}},
        {"producer", {{"id", "ssiag"}, {"kind", "symphony.stav.producer"}}},
        {"permissions",
         J::array({{{"event_class", "symphony.ssiag.policy.decision"},
                    {"operation_id", "symphony.ssiag.authorize"}}})}};
    J stav_reader = {{"uid", uid},
                     {"gid", gid},
                     {"subject",
                      {{"id", "scv-maintenance-reader"},
                       {"kind", "symphony.identity.operator"}}},
                     {"classifications", {"administrative_metadata"}}};
    J stav_authentication = {{"mechanism", "kernel-peer-credentials"},
                             {"authority", stav_authority},
                             {"producers", J::array({stav_producer})},
                             {"readers", J::array({stav_reader})}};
    private_json(
        stav_config,
        {{"schema", "symphony.stav.append-authority.config.v1"},
         {"mode", "user"},
         {"tops_id", tops},
         {"listen", {{"network", "unix"}, {"address", stav_socket.string()}}},
         {"ledger",
          {{"durability", "fsync-before-receipt"},
           {"max_bytes", 1048576},
           {"path", ledger.string()},
           {"recovery", "preserve-incomplete-tail"},
           {"retention", "preserve_all"},
           {"rotation", "disabled"}}},
         {"authentication", stav_authentication}});
    for (auto path : {ssiag_config.parent_path(), stav_config.parent_path(),
                      ssiag_socket.parent_path(), stav_socket.parent_path(),
                      ledger.parent_path()})
      private_directory(path);
    for (auto row : std::vector<std::array<std::string, 3>>{
             {"ssiag", "secure-identity-access-governance", "symphony-ssiag"},
             {"stav", "stav-append-authority",
              "symphony-stav-append-authority"}}) {
      auto name = row[0], module = row[1], executable = row[2];
      auto binary = output / "build" / executable;
      command("build-" + name,
              {"go", "build", "-o", binary.string(), "./cmd/" + executable},
              repo / "modules" / module, true, 120);
      std::vector<std::string> args = {binary.string()};
      if (name == "ssiag")
        args.insert(args.end(), {"package", "install"});
      else
        args.insert(args.end(), {"install", "--scope", "user"});
      args.insert(args.end(),
                  {"--prefix", install_prefix.string(), "--version", "dev"});
      command("install-" + name, args);
      auto installed = install_prefix / "libexec/symphony" / module / "dev" /
                       executable,
           receipt = install_prefix / "share/symphony/receipts" / module /
                     "dev/install-receipt.json";
      auto value = nt::read_json(receipt);
      NT_REQUIRE(value["version"] == "dev" &&
                 value["protocol"] == "symphony.knowledge.install-receipt.v2");
      NT_REQUIRE(nt::read(binary) == nt::read(installed));
      state["services"][name] = {
          {"binary", installed.string()},
          {"binary_digest", nt::digest(nt::read(installed))},
          {"receipt", receipt.string()},
          {"receipt_file_digest", nt::digest(nt::read(receipt))},
          {"receipt_digest", value["receipt_digest"]},
          {"version", "dev"},
          {"pid", nullptr}};
      save();
    }
    auto revision = nt::run({"git", "rev-parse", "HEAD"}, "", 30, {}, repo);
    NT_REQUIRE(revision.returncode == 0);
    state["source_commit"] = trim(revision.stdout_text);
    state["phase"] = "configured";
    save();
    start();
  }
  bool process_matches(const J &service) {
    if (!service.contains("pid") || service["pid"].is_null() ||
        service["pid"] == 0)
      return false;
    auto observed =
        nt::run({"/bin/ps", "-p", std::to_string(service["pid"].get<int>()),
                 "-o", "command="});
    auto command_line = trim(observed.stdout_text);
    return observed.returncode == 0 &&
           command_line.starts_with(service["binary"].get<std::string>() +
                                    " serve ") &&
           command_line.find(state["tops_id"].get<std::string>()) !=
               std::string::npos;
  }
  void start_service(const std::string &name) {
    auto &service = state["services"][name];
    if (process_matches(service))
      return;
    auto binary = service["binary"].get<std::string>();
    nt::require(nt::digest(nt::read(binary)) ==
                    service["binary_digest"].get<std::string>(),
                "service executable changed");
    int attempt = 1;
    for (auto event : state["process_history"])
      if (event["service"] == name)
        ++attempt;
    auto stdout_path = output / (name + "-serve-" + std::to_string(attempt) +
                                 "-stdout.txt"),
         stderr_path = output / (name + "-serve-" + std::to_string(attempt) +
                                 "-stderr.txt");
    std::vector<std::string> argv = {
        binary,      "serve",
        "--scope",   "user",
        "--tops-id", state["tops_id"],
        "--config",  state["paths"][name + "_config"]};
    auto env = environment();
    std::vector<std::string> environment_entries;
    for (const auto &[key, value] : env)
      environment_entries.push_back(key + "=" + value);
    std::vector<char *> environment_pointers;
    for (auto &entry : environment_entries)
      environment_pointers.push_back(entry.data());
    environment_pointers.push_back(nullptr);
    std::vector<char *> args;
    for (auto &arg : argv)
      args.push_back(arg.data());
    args.push_back(nullptr);
    pid_t pid = ::fork();
    NT_REQUIRE(pid >= 0);
    if (pid == 0) {
      if (::setsid() < 0)
        _exit(126);
      int in = ::open("/dev/null", O_RDONLY),
          out = ::open(stdout_path.c_str(), O_WRONLY | O_CREAT | O_TRUNC, 0600),
          err = ::open(stderr_path.c_str(), O_WRONLY | O_CREAT | O_TRUNC, 0600);
      if (in < 0 || out < 0 || err < 0)
        _exit(126);
      if (::dup2(in, 0) < 0 || ::dup2(out, 1) < 0 || ::dup2(err, 2) < 0)
        _exit(126);
      long maximum = ::sysconf(_SC_OPEN_MAX);
      for (int fd = 3; fd < (maximum > 0 ? maximum : 1024); ++fd)
        ::close(fd);
      if (::chdir(repo.c_str()) != 0)
        _exit(126);
      ::execve(binary.c_str(), args.data(), environment_pointers.data());
      _exit(127);
    }
    service["pid"] = pid;
    state["process_history"].push_back({{"service", name},
                                        {"pid", pid},
                                        {"argv", argv},
                                        {"started_at", now()},
                                        {"stdout", stdout_path.string()},
                                        {"stderr", stderr_path.string()},
                                        {"stopped_at", nullptr}});
    save();
    auto socket = fs::path(state["paths"][name + "_socket"].get<std::string>());
    for (int i = 0; i < 100; ++i) {
      int status = 0;
      auto done = ::waitpid(pid, &status, WNOHANG);
      if (done == pid)
        throw std::runtime_error(name + " exited; inspect " +
                                 stderr_path.string());
      struct stat info{};
      if (::stat(socket.c_str(), &info) == 0 && S_ISSOCK(info.st_mode))
        return;
      std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }
    throw std::runtime_error(name + " did not create its selected socket");
  }
  void start() {
    for (std::string name : {"stav", "ssiag"})
      start_service(name);
    std::string tops = state["tops_id"];
    qx("ready-stav",
       {"stav", "status", "--scope", "user", "--tops-id", tops, "--json"});
    qx("ready-ssiag",
       {"ssiag", "status", "--scope", "user", "--tops-id", tops, "--json"});
    state["phase"] = "ready";
    save();
    std::cout << J({{"phase", "ready"},
                    {"runtime", state_path.string()},
                    {"tops_id", tops}})
                     .dump()
              << '\n';
  }
  void stop_service(const std::string &name) {
    auto &service = state["services"][name];
    if (!process_matches(service)) {
      nt::require(
          !fs::exists(
              fs::path(state["paths"][name + "_socket"].get<std::string>())),
          "refusing to stop unverified PID with a remaining selected socket");
      return;
    }
    auto pid = service["pid"].get<pid_t>();
    NT_REQUIRE(::kill(pid, SIGTERM) == 0);
    bool stopped = false;
    for (int i = 0; i < 100; ++i) {
      if (!process_matches(service)) {
        stopped = true;
        break;
      }
      std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }
    nt::require(stopped, "owned service did not stop; no force-kill performed");
    int status = 0;
    (void)::waitpid(pid, &status, WNOHANG);
    for (auto it = state["process_history"].rbegin();
         it != state["process_history"].rend(); ++it)
      if ((*it)["service"] == name && (*it)["pid"] == pid &&
          (*it)["stopped_at"].is_null()) {
        (*it)["stopped_at"] = now();
        break;
      }
    service["pid"] = nullptr;
    save();
  }
  J audit(const std::string &label) {
    std::string tops = state["tops_id"];
    auto query = qx(label + "-audit-query",
                    {"stav", "query", "--scope", "user", "--tops-id", tops,
                     "--limit", "100", "--json"}),
         verify =
             qx(label + "-audit-verify", {"stav", "verify", "--scope", "user",
                                          "--tops-id", tops, "--json"});
    nt::require(verify["result"]["state"] == "verified",
                "STAV chain verification failed");
    private_json(
        output / (label + "-audit.json"),
        {{"query", query},
         {"verification", verify},
         {"ledger_digest",
          nt::digest(nt::read(state["paths"]["ledger"].get<std::string>()))}});
    return query;
  }
  void stop(const std::string &label = "final") {
    J error = nullptr;
    try {
      audit(label);
    } catch (const std::exception &failure) {
      error = failure.what();
      private_json(output / "FINAL_AUDIT_UNAVAILABLE.json",
                   {{"observed_at", now()}, {"error", error}});
    }
    for (std::string name : {"ssiag", "stav"})
      stop_service(name);
    state["phase"] = "stopped";
    state["final_audit_error"] = error;
    save();
    std::cout
        << J({{"phase", "stopped"}, {"runtime", state_path.string()}}).dump()
        << '\n';
  }
  J source(const std::string &label, const std::string &operation,
           J value = nullptr, const std::string &operation_id = "",
           bool expected_success = true) {
    std::vector<std::string> args = {"scv",
                                     "source",
                                     operation,
                                     "--domain",
                                     state["domain"],
                                     "--prefix",
                                     state["scv_prefix"],
                                     "--version",
                                     state["scv_version"],
                                     "--tops-id",
                                     state["tops_id"],
                                     "--state-root",
                                     state["source_root"],
                                     "--source-id",
                                     state["source_id"],
                                     "--json"};
    if (!value.is_null()) {
      auto path = output / (label + "-input.json");
      private_json(path, value);
      args.insert(args.end(), {"--input", path.string()});
    }
    if (!operation_id.empty())
      args.insert(args.end(), {"--operation-id", operation_id});
    return qx(label, args, expected_success);
  }
  void adoption(const fs::path &source_input) {
    nt::require(!fs::exists(output / "ADOPTION.json"),
                "adoption already recorded; no implicit replay or overwrite");
    auto selected = nt::read_json(source_input), initial = selected["current"],
         desired = selected["desired"];
    NT_REQUIRE(initial["source_id"] == desired["source_id"] &&
               desired["source_id"] == state["source_id"]);
    auto old_desired = without(
        initial, {"protocol", "digest", "generation", "predecessor_digest"});
    J ids = object();
    for (std::string name : {"onboard", "relocate", "stale", "permission"})
      ids[name] = "inc12-do-" + name;
    state["operation_ids"] = ids;
    save();
    std::string onboard_id = ids["onboard"], relocate_id = ids["relocate"],
                stale_id = ids["stale"];
    auto onboard =
        source("onboard-proposal", "propose",
               {{"operation_id", onboard_id},
                {"desired", old_desired},
                {"reason", "Explicit isolated adoption of the previously "
                           "retained official source declaration"}});
    NT_REQUIRE(onboard["source"] == initial);
    source("onboard-denied", "apply", onboard, "", false);
    auto denied = source("denied-source-status", "status", nullptr, onboard_id);
    NT_REQUIRE(denied["state_digest"].is_null() &&
               denied["attempt"]["status"] == "prepared");
    auto denied_audit = audit("denied"),
         denied_events = denied_audit["entries"];
    NT_REQUIRE(denied_events.size() == 1 &&
               denied_events[0]["projection"]["outcome"] == "denied");
    std::string tops = state["tops_id"];
    auto policy = qx("policy-status", {"ssiag", "policy", "status", "--scope",
                                       "user", "--tops-id", tops, "--json"});
    std::string resource =
        "symphony.scv.source:" +
        sha(nt::canonical({{"tops_id", tops},
                           {"domain", state["domain"]},
                           {"source_id", state["source_id"]}}));
    J grants = {{"default_effect", "deny"},
                {"max_capability_seconds", 60},
                {"grants", array()}};
    for (std::string kind : {"onboard", "relocate"})
      grants["grants"].push_back({{"id", "scv-maintenance-" + kind},
                                  {"subject_id", state["subject_id"]},
                                  {"authority_basis", "granted_permission"},
                                  {"operation", "symphony.scv.source." + kind},
                                  {"resource", resource},
                                  {"audience", "qxctl"},
                                  {"scope", "tops:" + tops}});
    auto policy_input = output / "exact-source-grants.json";
    private_json(policy_input, grants);
    auto proposal =
        qx("policy-propose",
           {"ssiag", "policy", "propose", "--scope", "user", "--tops-id", tops,
            "--operation-id", ids["permission"], "--expected-policy-digest",
            policy["policy_digest"], "--authority-basis", "host_owner",
            "--input", policy_input.string(), "--json"});
    auto proposal_path = output / "policy-proposal.json";
    private_json(proposal_path, proposal);
    qx("policy-apply",
       {"ssiag", "policy", "apply", "--scope", "user", "--tops-id", tops,
        "--input", proposal_path.string(), "--json"});
    auto first = source("onboard-recover", "recover", nullptr, onboard_id);
    NT_REQUIRE(first["state_digest"] == initial["digest"] &&
               first["attempt"]["status"] == "committed");
    auto relocation = source("relocation-proposal", "propose",
                             {{"operation_id", relocate_id},
                              {"desired", desired},
                              {"reason", selected["reason"]}}),
         stale = source("stale-proposal", "propose",
                        {{"operation_id", stale_id},
                         {"desired", desired},
                         {"reason", selected["reason"]}});
    NT_REQUIRE(relocation["change_kind"] == "relocate" &&
               relocation["expected_state_digest"] == initial["digest"]);
    audit("before-outage");
    stop_service("stav");
    source("relocation-audit-unavailable", "apply", relocation, "", false);
    auto pending =
        source("pending-source-status", "status", nullptr, relocate_id);
    NT_REQUIRE(pending["state_digest"] == initial["digest"] &&
               pending["attempt"]["status"] == "prepared");
    start_service("stav");
    auto adopted =
        source("relocation-recover", "recover", nullptr, relocate_id);
    NT_REQUIRE(adopted["state_digest"] == relocation["source"]["digest"] &&
               adopted["attempt"]["status"] == "committed");
    auto ledger_path = fs::path(state["paths"]["ledger"].get<std::string>());
    auto ledger_before = nt::read(ledger_path);
    auto replay =
        source("relocation-committed-replay", "recover", nullptr, relocate_id);
    NT_REQUIRE(replay == adopted && nt::read(ledger_path) == ledger_before);
    source("stale-plan-rejected", "apply", stale, "", false);
    auto final =
        source("adopted-source-status", "status", nullptr, relocate_id);
    NT_REQUIRE(final["state_digest"] == relocation["source"]["digest"] &&
               final["owner_result"]["source"] == relocation["source"]);
    NT_REQUIRE(nt::read(ledger_path) == ledger_before);
    auto ledger = audit("adoption");
    NT_REQUIRE(ledger["entries"].size() == 4);
    J events = array();
    for (auto entry : ledger["entries"])
      if (entry["projection"]["target"]["id"] == resource)
        events.push_back(entry);
    std::string onboard_correlation =
                    first["attempt"]["authorization"]["correlation_id"],
                relocate_correlation =
                    adopted["attempt"]["authorization"]["correlation_id"];
    NT_REQUIRE(onboard_correlation != relocate_correlation &&
               canonical_uuid(onboard_correlation) &&
               canonical_uuid(relocate_correlation));
    J observed = array();
    for (auto entry : events)
      observed.push_back({entry["projection"]["correlation_id"],
                          entry["projection"]["outcome"]});
    NT_REQUIRE(observed == J::array({{onboard_correlation, "denied"},
                                     {onboard_correlation, "allowed"},
                                     {relocate_correlation, "allowed"}}));
    for (auto entry : events) {
      auto projection = entry["projection"];
      NT_REQUIRE(entry["verification_state"] == "verified" &&
                 projection["actor"]["id"] == state["subject_id"]);
      NT_REQUIRE(projection["event_class"] ==
                     "symphony.ssiag.policy.decision" &&
                 projection["operation_id"] == "symphony.ssiag.authorize");
    }
    for (auto result : {first, adopted}) {
      auto decision = result["attempt"]["authorization"];
      J matches = array();
      for (auto entry : events)
        if (entry["projection"]["request_id"] == decision["request_id"])
          matches.push_back(entry);
      NT_REQUIRE(matches.size() == 1 &&
                 matches[0]["projection"]["outcome"] == "allowed");
      NT_REQUIRE(matches[0]["projection"]["correlation_id"] ==
                     decision["correlation_id"] &&
                 decision["target"]["resource"] == resource);
    }
    auto adopted_path = output / "ADOPTED_SOURCE.json";
    private_json(adopted_path, final["owner_result"]["source"]);
    private_json(
        output / "ADOPTION.json",
        {{"protocol", "local.scv.real-source-adoption.v1"},
         {"status", "verified"},
         {"completed_at", now()},
         {"runtime", state_path.string()},
         {"source_input", fs::canonical(source_input).string()},
         {"source_input_digest", nt::digest(nt::read(source_input))},
         {"initial_source_digest", initial["digest"]},
         {"adopted_source_digest", final["state_digest"]},
         {"adopted_source", adopted_path.string()},
         {"source_resource", resource},
         {"onboard_operation_id", onboard_id},
         {"relocate_operation_id", relocate_id},
         {"onboard_audit_correlation_id", onboard_correlation},
         {"relocate_audit_correlation_id", relocate_correlation},
         {"denied_kept_prepared_intent_and_absent_head", true},
         {"grants_applied_through_real_qxctl_ssiag_policy", true},
         {"stav_unavailable_kept_prepared_intent_and_initial_head", true},
         {"recovery_committed_successor", true},
         {"committed_retry_and_stale_rejection_added_no_audit_event_or_write",
          true},
         {"real_stav_query", ledger},
         {"authorization_audit",
          "real_ssiag_policy_decision_committed_to_real_stav"},
         {"source_authorization_events", events},
         {"source_write_receipt", nullptr},
         {"source_write_evidence",
          "exact protected source journal transition and committed state"},
         {"continuity_semantics",
          "The authored declaration and selected grant do not independently "
          "establish publisher authority or content sufficiency."}});
    state["phase"] = "adopted";
    state["adopted_source"] = adopted_path.string();
    state["adopted_source_digest"] = final["state_digest"];
    save();
    std::cout << J({{"status", "verified"},
                    {"adopted_source", adopted_path.string()},
                    {"runtime", state_path.string()}})
                     .dump()
              << '\n';
  }
  void legacy_recovery(const fs::path &selected_client) {
    nt::require(
        state["phase"] == "stopped" && !fs::exists(output / "ADOPTION.json"),
        "legacy recovery requires the stopped, unsuccessful original specimen");
    nt::require(!selected_client.empty(),
                "legacy recovery requires an explicit corrected --cli");
    auto client = fs::canonical(selected_client);
    auto client_digest = nt::digest(nt::read(client));
    auto retained = output / "legacy-recovery";
    nt::require(fs::create_directory(retained),
                "legacy recovery evidence exists; refusing overwrite");
    NT_REQUIRE(::chmod(retained.c_str(), 0700) == 0);
    auto before_runtime = nt::read(state_path);
    nt::write(retained / "RUNTIME-before.json", before_runtime);
    std::vector<fs::path> documents;
    for (const auto &entry : fs::recursive_directory_iterator(
             fs::path(state["source_root"].get<std::string>())))
      if (entry.path().filename() == "state.json")
        documents.push_back(entry.path());
    nt::require(documents.size() == 1,
                "legacy specimen source document is ambiguous");
    auto document_path = documents[0];
    auto before_bytes = nt::read(document_path);
    nt::write(retained / "source-state-before.json", before_bytes);
    auto before = J::parse(before_bytes);
    NT_REQUIRE(before["protocol"] == "symphony.qxctl.scv-source-store.v1" &&
               before["state_digest"].is_null());
    std::string operation = "inc12-do-onboard";
    auto attempt = before["operations"][operation];
    NT_REQUIRE(attempt["status"] == "prepared" &&
               !attempt.contains("correlation_id"));
    try {
      start();
      auto ledger_before =
          nt::read(state["paths"]["ledger"].get<std::string>());
      auto result = command(
          "legacy-source-recover",
          {client.string(), "scv", "source", "recover", "--domain",
           state["domain"], "--prefix", state["scv_prefix"], "--version",
           state["scv_version"], "--tops-id", state["tops_id"], "--state-root",
           state["source_root"], "--source-id", state["source_id"],
           "--operation-id", operation, "--json"});
      NT_REQUIRE(nt::digest(nt::read(client)) == client_digest);
      auto after_bytes = nt::read(document_path);
      nt::write(retained / "source-state-after.json", after_bytes);
      auto after = J::parse(after_bytes);
      NT_REQUIRE(after["protocol"] == "symphony.qxctl.scv-source-store.v2" &&
                 after["operations"][operation]["intent"] == attempt["intent"]);
      NT_REQUIRE(result["attempt"]["status"] == "committed" &&
                 result["state_digest"] ==
                     attempt["intent"]["transition"]["state_digest"]);
      std::string correlation =
          after["operations"][operation]["correlation_id"];
      NT_REQUIRE(canonical_uuid(correlation));
      auto decision = result["attempt"]["authorization"];
      NT_REQUIRE(decision["correlation_id"] == correlation);
      auto page = audit("legacy-recovery");
      J events = array();
      for (auto entry : page["entries"])
        if (entry["projection"]["request_id"] == decision["request_id"])
          events.push_back(entry);
      NT_REQUIRE(events.size() == 1 &&
                 events[0]["projection"]["outcome"] == "allowed");
      NT_REQUIRE(events[0]["projection"]["correlation_id"] == correlation &&
                 nt::read(retained / "source-state-before.json") ==
                     before_bytes);
      private_json(
          retained / "VERIFICATION.json",
          {{"protocol", "local.scv.real-legacy-source-recovery.v1"},
           {"status", "verified"},
           {"completed_at", now()},
           {"original_cli", state["qxctl"]},
           {"original_cli_digest",
            "sha256:" + state["qxctl_sha256"].get<std::string>()},
           {"corrected_cli", client.string()},
           {"corrected_cli_digest", client_digest},
           {"original_runtime_digest", nt::digest(before_runtime)},
           {"original_state_digest", nt::digest(before_bytes)},
           {"result_state_digest", nt::digest(after_bytes)},
           {"ledger_before_digest", nt::digest(ledger_before)},
           {"operation_id", operation},
           {"correlation_id", correlation},
           {"original_intent_unchanged", true},
           {"source_generation", after["source"]["generation"]},
           {"source_digest", after["state_digest"]},
           {"real_source_authorization_event", events[0]},
           {"source_write_receipt", nullptr},
           {"scope",
            "A genuine previously failed v1 pending operation recovered "
            "through the exact original native installation and real isolated "
            "SSIAG/STAV; copied before bytes remain immutable."}});
    } catch (...) {
      auto failure = std::current_exception();
      stop("legacy-final");
      std::rethrow_exception(failure);
    }
    stop("legacy-final");
  }
};
} // namespace
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::require(
        argc >= 2,
        "Select setup, start, adoption, audit, stop or legacy-recovery");
    std::string action = argv[1];
    std::set<std::string> allowed = {"setup", "start", "adoption",
                                     "audit", "stop",  "legacy-recovery"};
    nt::require(allowed.contains(action), "Unknown authority action");
    nt::Arguments args(argc - 1, argv + 1);
    auto out = args.require("out"), repo = args.require("repo"),
         cli = args.require("cli"), prefix = args.require("prefix");
    auto input = args.get("source-input", args.get("source-plan"));
    nt::require(action != "adoption" || !input.empty(),
                "adoption requires --source-plan pointing to the selected "
                "source-plan input");
    Harness harness(out, repo,
                    action == "legacy-recovery" ? fs::path() : fs::path(cli),
                    prefix);
    try {
      if (action == "setup")
        harness.setup();
      else if (action == "start")
        harness.start();
      else if (action == "adoption")
        harness.adoption(input);
      else if (action == "audit")
        harness.audit("inspection");
      else if (action == "stop")
        harness.stop();
      else
        harness.legacy_recovery(cli);
    } catch (const std::exception &error) {
      if (fs::exists(harness.output)) {
        auto time = std::chrono::system_clock::to_time_t(
            std::chrono::system_clock::now());
        std::tm local{};
        ::localtime_r(&time, &local);
        std::ostringstream name;
        name << "FAILURE-" << std::put_time(&local, "%Y%m%dT%H%M%S") << ".json";
        private_json(harness.output / name.str(),
                     {{"action", action},
                      {"observed_at", now()},
                      {"error", error.what()},
                      {"runtime", harness.state_path.string()}});
      }
      throw;
    }
  });
}
