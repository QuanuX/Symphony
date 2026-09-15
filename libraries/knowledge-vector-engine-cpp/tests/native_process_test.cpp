#include "support/native_test.hpp"

using namespace native_test;

namespace {
struct Descriptor {
  int value;
  ~Descriptor() { ::close(value); }
};

int probe(int argc, char **argv) {
  const std::string mode = argv[1];
  if (mode == "--probe-descriptor") {
    require(argc == 3, "Probe requires a descriptor");
    errno = 0;
    const int result = ::fcntl(std::stoi(argv[2]), F_GETFD);
    require(result == -1 && errno == EBADF, "Unrelated descriptor inherited");
    std::cout << "descriptor closed";
    return 0;
  }
  if (mode == "--probe-capture") {
    const std::string input{std::istreambuf_iterator<char>(std::cin), {}};
    std::cout << "received:" << input;
    std::cerr << "expected diagnostic";
    return 7;
  }
  if (mode == "--probe-environment") {
    require(argc == 3, "Probe requires the expected inheritance mode");
    const char *override_value = ::getenv("SYMPHONY_NATIVE_PROCESS_OVERRIDE");
    require(override_value != nullptr && std::string(override_value) == "child",
            "Child environment override missing");
    const char *inherited = ::getenv("SYMPHONY_NATIVE_PROCESS_INHERITED");
    if (std::string(argv[2]) == "clear")
      require(inherited == nullptr, "Unwanted parent environment inherited");
    else
      require(inherited != nullptr && std::string(inherited) == "parent",
              "Parent environment was not inherited");
    std::cout << "environment correct";
    return 0;
  }
  if (mode == "--probe-descendant") {
    require(argc == 3, "Probe requires a mutation path");
    const fs::path mutation = argv[2];
    const pid_t descendant = ::fork();
    require(descendant >= 0, "Descendant fork failed");
    if (descendant == 0) {
      std::this_thread::sleep_for(std::chrono::milliseconds(350));
      write(mutation, "descendant mutated the fixture");
      std::cout << "late output" << std::flush;
      std::cerr << "late diagnostic" << std::flush;
      _exit(0);
    }
    write(mutation.string() + ".pid", std::to_string(descendant));
    std::cout << "{\"status\":\"ok\"}" << std::flush;
    return 0;
  }
  throw std::runtime_error("Unknown helper probe");
}

void test_unrelated_descriptor_is_closed(const fs::path &self) {
  TempDir fixture;
  Descriptor original{::open((fixture.path / "owned").c_str(),
                             O_RDWR | O_CREAT | O_TRUNC, 0600)};
  require(original.value >= 0, "Cannot open parent-owned file");
  Descriptor inherited{::fcntl(original.value, F_DUPFD, 80)};
  require(inherited.value >= 80, "Cannot reserve inheritable descriptor");
  NT_REQUIRE((::fcntl(inherited.value, F_GETFD) & FD_CLOEXEC) == 0);
  const auto result = run(
      {self.string(), "--probe-descriptor", std::to_string(inherited.value)});
  NT_REQUIRE(result.returncode == 0);
  NT_REQUIRE(result.stdout_text == "descriptor closed");
  NT_REQUIRE(result.stderr_text.empty());
  NT_REQUIRE(::fcntl(inherited.value, F_GETFD) >= 0);
}

void test_nonzero_exit_and_capture_are_preserved(const fs::path &self) {
  const std::string input("binary\0input\n", 13);
  const auto result = run({self.string(), "--probe-capture"}, input);
  NT_REQUIRE(result.returncode == 7);
  NT_REQUIRE(result.stdout_text == "received:" + input);
  NT_REQUIRE(result.stderr_text == "expected diagnostic");
}

void test_explicit_environment_and_path_lookup(const fs::path &self) {
  TempDir fixture;
  fs::create_symlink(self, fixture.path / "native-environment-probe");
  require(::setenv("SYMPHONY_NATIVE_PROCESS_INHERITED", "parent", 1) == 0 &&
              ::setenv("SYMPHONY_NATIVE_PROCESS_OVERRIDE", "parent", 1) == 0,
          "Cannot prepare parent environment");
  for (const bool clear : {false, true}) {
    const auto result = run({"native-environment-probe", "--probe-environment",
                             clear ? "clear" : "inherit"},
                            "", 40,
                            {{"PATH", fixture.path.string()},
                             {"SYMPHONY_NATIVE_PROCESS_OVERRIDE", "child"}},
                            {}, clear);
    NT_REQUIRE(result.returncode == 0);
    NT_REQUIRE(result.stdout_text == "environment correct");
    NT_REQUIRE(result.stderr_text.empty());
    NT_REQUIRE(std::string(::getenv("SYMPHONY_NATIVE_PROCESS_OVERRIDE")) ==
               "parent");
    NT_REQUIRE(std::string(::getenv("SYMPHONY_NATIVE_PROCESS_INHERITED")) ==
               "parent");
  }
  ::unsetenv("SYMPHONY_NATIVE_PROCESS_INHERITED");
  ::unsetenv("SYMPHONY_NATIVE_PROCESS_OVERRIDE");
}

void test_running_descendant_is_rejected_and_cleaned(const fs::path &self) {
  TempDir fixture;
  const auto mutation = fixture.path / "unexpected-mutation";
  // Cleanup still runs if this regression fails, but only after checking that
  // run() itself prevented the delayed mutation and capture writes.
  struct ProbeCleanup {
    fs::path pid_path;
    ~ProbeCleanup() {
      try {
        if (fs::exists(pid_path))
          ::kill(static_cast<pid_t>(std::stol(read(pid_path))), SIGKILL);
      } catch (...) {
      }
    }
  } cleanup{mutation.string() + ".pid"};
  bool rejected = false;
  try {
    static_cast<void>(
        run({self.string(), "--probe-descendant", mutation.string()}));
  } catch (const std::exception &error) {
    rejected =
        std::string(error.what()).find("Process left running descendants") !=
        std::string::npos;
  }
  std::this_thread::sleep_for(std::chrono::milliseconds(600));
  NT_REQUIRE(fs::is_regular_file(cleanup.pid_path));
  NT_REQUIRE(rejected);
  NT_REQUIRE(!fs::exists(mutation));
}
} // namespace

int main(int argc, char **argv) {
  if (argc > 1 && std::string(argv[1]).starts_with("--probe-")) {
    try {
      return probe(argc, argv);
    } catch (const std::exception &error) {
      std::cerr << error.what();
      return 9;
    }
  }
  return test_main([&] {
    const auto self = fs::canonical(argv[0]);
    test_unrelated_descriptor_is_closed(self);
    std::cout << "PASS test_unrelated_descriptor_is_closed\n";
    test_nonzero_exit_and_capture_are_preserved(self);
    std::cout << "PASS test_nonzero_exit_and_capture_are_preserved\n";
    test_explicit_environment_and_path_lookup(self);
    std::cout << "PASS test_explicit_environment_and_path_lookup\n";
    test_running_descendant_is_rejected_and_cleaned(self);
    std::cout << "PASS test_running_descendant_is_rejected_and_cleaned\n";
  });
}
