#include <symphony/knowledge/engine/manifest_discovery.hpp>

#include <filesystem>
#include <fstream>
#include <iostream>
#include <map>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>

namespace fs = std::filesystem;

namespace {

std::string read(const fs::path &path) {
  std::ifstream input(path, std::ios::binary);
  if (!input) {
    throw std::runtime_error("Cannot read " + path.string());
  }
  std::string content{std::istreambuf_iterator<char>(input),
                      std::istreambuf_iterator<char>()};
  if (input.bad()) {
    throw std::runtime_error("Read failed for " + path.string());
  }
  return content;
}

void write(const fs::path &path, const std::string &content) {
  fs::create_directories(path.parent_path());
  std::ofstream output(path, std::ios::binary | std::ios::trunc);
  output << content;
  output.close();
  if (!output) {
    throw std::runtime_error("Cannot write " + path.string());
  }
}

void replace(std::string &text, std::string_view before,
             std::string_view after) {
  std::size_t position = 0;
  while ((position = text.find(before, position)) != std::string::npos) {
    text.replace(position, before.size(), after);
    position += after.size();
  }
}

void modernize_existing_headings(const fs::path &fixture) {
  for (const auto &relative : {"README.md", "knowledge/platform/INTENT.md"}) {
    const auto path = fixture / relative;
    if (!fs::exists(path)) {
      continue;
    }
    auto content = read(path);
    // Rename only existing headings: a deliberately missing anchor stays
    // absent.
    replace(content, "## First Runtime Set", "## Emerging Vector Architecture");
    replace(content, "## Relationship to First Runtime Set",
            "## Relationship to Emerging Runtime Architecture");
    write(path, content);
  }
}

std::map<std::string, std::string>
discovery_additions(const fs::path &baseline) {
  return {
      {"go.work", "go 1.26.5\n"},
      {"knowledge/TIME.md", read(baseline / "knowledge/TIME.md")},
      {"knowledge/FOUNDATIONAL-LIFECYCLE.md",
       read(baseline / "knowledge/FOUNDATIONAL-LIFECYCLE.md")},
      {"knowledge/sacv/INTENT.md", R"(# Symphony API Contract Vector Intent
## Purpose
## Source-Truth Boundary
## Scope
## Non-Authorization Statement
)"},
      {"knowledge/sacv/MANIFEST.md", R"(# Symphony API Contract Vector Manifest
## Identity
## Declared Contract Truth Role
## Installability Considerations
## Non-Authorization Statement
)"},
      {"knowledge/sacv/SKILL.md", R"(# Symphony API Contract Vector Skill
## Purpose
## Caller Authority
## Non-Authorization Statement
)"},
      {"knowledge/sacv/SPEC.md", R"(# Symphony API Contract Vector Specification
## Purpose
## Registry Contract
## SACV Engine Operations
## Non-Authorization Statement
)"},
      {"knowledge/sacv/REGISTRY.md", R"(# Symphony API Contract Registry
## Purpose
## Entry Model
## Canonical Entries
None.
## Prohibited Entries
)"},
      {"knowledge/sodv/RELEASES.md", R"(# Symphony Release Ledger
## Canonical Release Records
None.
)"},
  };
}

std::string index_entry(const std::string &path) {
  return "\n#### " + path + "\n- path: `" + path +
         "`\n- title: Current discovery fixture surface\n"
         "- surface_type: fixture contract\n"
         "- truth_role: fixture source truth\n"
         "- owner: validator fixture\n"
         "- scope: Enables the current discovery boundary for a legacy case.\n"
         "- relationships: none\n"
         "- consumers: symphony-validator tests\n"
         "- deferred_projections: none\n"
         "- status: canonical\n"
         "- notes: Test data only.\n";
}

void prepare(const fs::path &source, const fs::path &destination) {
  if (!fs::is_directory(source) || fs::exists(destination)) {
    throw std::runtime_error(
        "Preparation requires a source directory and absent destination");
  }
  const auto baseline = source.parent_path() / "fixtures_valid";
  const auto name = source.filename().string();
  if (name == "fixtures") {
    // The earliest malformed-index fixture contains only its index. Supply the
    // independent contract scaffold so malformed fields reach the SKVI parser.
    fs::copy(baseline, destination, fs::copy_options::recursive);
    fs::copy_file(source / "knowledge/skvi/INDEX.md",
                  destination / "knowledge/skvi/INDEX.md",
                  fs::copy_options::overwrite_existing);
    const auto index_path = destination / "knowledge/skvi/INDEX.md";
    write(index_path, read(index_path) + R"(
Symphony Knowledge Vector Index
## Purpose
## Entry Model
## Relationship Model
## Projection Doctrine
)");
  } else {
    fs::copy(source, destination, fs::copy_options::recursive);
  }
  modernize_existing_headings(destination);

  auto additions = discovery_additions(baseline);
  std::set<std::string> canonical;
  // This declaration comes from the positive fixture, not from the mutated
  // index or the negative fixture's remaining files. Missing coverage remains
  // observable, as do missing canonical files.
  for (const auto &entry : fs::recursive_directory_iterator(baseline)) {
    if (entry.is_regular_file()) {
      canonical.insert(fs::relative(entry.path(), baseline).generic_string());
    }
  }
  for (const auto &[path, content] : additions) {
    canonical.insert(path);
    if (!fs::exists(destination / path)) {
      write(destination / path, content);
    }
  }
  canonical.insert("knowledge/MANIFEST.md");
  std::string manifest =
      "# Validator Fixture Manifest\n\n## Canonical Surfaces\n\n";
  for (const auto &path : canonical) {
    manifest += "- `" + path + "`\n";
  }
  manifest += "\n## Subordinate Manifests\n";
  write(destination / "knowledge/MANIFEST.md", manifest);

  const auto index_path = destination / "knowledge/skvi/INDEX.md";
  auto index = read(index_path);
  additions.emplace("knowledge/MANIFEST.md", manifest);
  for (const auto &[path, unused] : additions) {
    if (index.find("- path: `" + path + "`") == std::string::npos) {
      index += index_entry(path);
    }
  }
  write(index_path, index);

  if (name == "fixtures_invalid_skvi_status") {
    // Its status payload predates the requirement for these four fields. Supply
    // them without changing either status value, so vocabulary checking runs.
    auto status_index = read(index_path);
    const std::string fields = "- relationships: none\n- consumers: humans\n"
                               "- deferred_projections: none\n- notes: none\n";
    replace(status_index, "- status: some_invalid_status\n",
            "- status: some_invalid_status\n" + fields);
    // Only the original INDEX.md entry lacks these fields; additions have them.
    replace(status_index, "- scope: index\n- status: canonical\n",
            "- scope: index\n- status: canonical\n" + fields);
    write(index_path, status_index);
  }

  if (name == "fixtures_missing_runtime_anchor") {
    // node-troll is now a legacy seed. Exercise the same missing Identity
    // anchor against the currently checked runtime while retaining the original
    // data.
    const auto intent = destination / "modules/hotpath-runtime/INTENT.md";
    auto content = read(intent);
    replace(content, "## Identity", "## Midentity");
    write(intent, content);
  }

  // This fixture had become identical to the general legacy scaffold before
  // this repair. Restore its explicit missing-knowledge-surface stimulus.
  if (name == "fixtures_missing_knowledge_surface") {
    fs::remove(destination / "knowledge/INTENT.md");
  }

  // The old value fixtures predate the parser's required PR011 bootstrap
  // record. Append that independent valid record without rewriting their
  // malformed PR010 record or any original bytes. Other fixture ledgers stay
  // exact.
  const auto ledger = source / "knowledge/sclv/CHANGELOG.md";
  if (fs::exists(ledger)) {
    const auto original = read(ledger);
    const auto prepared_ledger = destination / "knowledge/sclv/CHANGELOG.md";
    if (read(prepared_ledger) != original) {
      throw std::runtime_error(
          "Fixture preparation changed historical SCLV bytes");
    }
    if (name.starts_with("fixtures_invalid_sclv_")) {
      if (original.find("- record_id: `SCLV-PR-011`") == std::string::npos) {
        const auto valid = read(baseline / "knowledge/sclv/CHANGELOG.md");
        const auto begin = valid.find("### SCLV-PR-011\n");
        const auto end = valid.find("### SCLV v3", begin);
        if (begin == std::string::npos || end == std::string::npos) {
          throw std::runtime_error("Missing independent PR011 scaffold record");
        }
        write(prepared_ledger,
              original + "\n" + valid.substr(begin, end - begin));
        if (!read(prepared_ledger).starts_with(original)) {
          throw std::runtime_error(
              "Fixture preparation rewrote original ledger prefix");
        }
      }
    }
  }
  std::cout << "Prepared " << name << " with " << canonical.size()
            << " declared surfaces; original ledger preserved\n";
}

void check_json_inventory(const fs::path &repository,
                          const fs::path &evidence) {
  const auto catalog = symphony::knowledge::engine::discover_canonical_surfaces(
      fs::absolute(repository));
  if (!catalog.valid()) {
    throw std::runtime_error(
        "Cannot compare JSON inventory against invalid manifests");
  }
  std::set<std::string> expected;
  for (const auto &surface : catalog.surfaces) {
    if (surface.path.starts_with("knowledge/") &&
        surface.path.ends_with(".json")) {
      expected.insert(surface.path);
    }
  }
  std::set<std::string> actual;
  constexpr std::string_view prefix =
      "evidence pass artifact.canonical_json_authorized path=";
  std::istringstream lines(read(evidence));
  for (std::string line; std::getline(lines, line);) {
    if (line.starts_with(prefix)) {
      const auto end = line.find(" authority=", prefix.size());
      if (end == std::string::npos ||
          !actual.insert(line.substr(prefix.size(), end - prefix.size()))
               .second) {
        throw std::runtime_error(
            "Malformed or duplicate JSON artifact evidence");
      }
    }
  }
  if (expected.empty() || actual != expected) {
    for (const auto &path : expected) {
      if (!actual.contains(path)) {
        std::cerr << "Missing JSON authorization evidence: " << path << '\n';
      }
    }
    for (const auto &path : actual) {
      if (!expected.contains(path)) {
        std::cerr << "Undeclared JSON authorization evidence: " << path << '\n';
      }
    }
    throw std::runtime_error(
        "JSON evidence differs from exact manifest inventory");
  }
  std::cout << "Exact canonical JSON inventory passed: " << expected.size()
            << " paths\n";
}

} // namespace

int main(int argc, char **argv) {
  try {
    if (argc != 4) {
      throw std::runtime_error(
          "Usage: validator-fixture-prepare prepare <source> <destination> | "
          "json-inventory <repository> <evidence>");
    }
    const std::string command = argv[1];
    if (command == "prepare") {
      prepare(fs::absolute(argv[2]), fs::absolute(argv[3]));
    } else if (command == "json-inventory") {
      check_json_inventory(argv[2], argv[3]);
    } else {
      throw std::runtime_error("Unknown fixture helper command: " + command);
    }
    return 0;
  } catch (const std::exception &error) {
    std::cerr << "error: " << error.what() << '\n';
    return 1;
  }
}
