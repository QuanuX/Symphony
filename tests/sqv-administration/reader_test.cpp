#include "store_reader.hpp"
#include <private_temp.hpp>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <map>
#include <symphony/sqpv/local_store.hpp>
#include <sys/stat.h>
#include <unistd.h>
using namespace symphony;
namespace reader = sqpv::inspection;
namespace admin = sqv_admin;
void check(bool b) {
  if (!b)
    throw std::runtime_error("read-only store regression failed");
}
struct Root {
  std::string path;
  Root() {
    auto pattern = testing::temporary_pattern("sqv-reader-XXXXXX");
    auto *v = ::mkdtemp(pattern.data());
    check(v != nullptr);
    path = v;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
std::string content(const std::string &path) {
  std::ifstream f(path, std::ios::binary);
  check(f.good());
  return {(std::istreambuf_iterator<char>(f)), {}};
}
void write(const std::string &path, const std::string &bytes) {
  std::ofstream f(path, std::ios::binary | std::ios::trunc);
  f << bytes;
  f.close();
  check(f.good() && ::chmod(path.c_str(), 0600) == 0);
}
auto contents(const std::string &path) {
  std::map<std::string, std::string> result;
  for (const auto &entry : std::filesystem::directory_iterator(path))
    result.emplace(entry.path().filename().string(), content(entry.path().string()));
  return result;
}
void replace(const std::string &path) {
  write(path + ".replacement", content(path));
  check(::rename((path + ".replacement").c_str(), path.c_str()) == 0);
}
int main() try {
  Root root;
  auto parent = root.path + "/parent", path = parent + "/store";
  check(::mkdir(parent.c_str(), 0700) == 0 && ::mkdir(path.c_str(), 0700) == 0);
  sqmv::Description d{"test",
                      "r1",
                      "schema",
                      "layout",
                      "scope",
                      "producer",
                      {{sqmv::EvidenceRole::schema, "test", "schema"},
                       {sqmv::EvidenceRole::layout, "test", "layout"},
                       {sqmv::EvidenceRole::access, "test", "access"}}};
  sqmv::Manifest m;
  check(sqmv::Manifest::create(d, {65536, 4096, 8}, m) == sqmv::Status::ok);
  sqpv::Options options{"partition", {}, {}, 7, {4096, 1048576, 8}};
  options.producer_generation[0] = 1;
  options.store_generation[0] = 2;
  sqpv::Store store;
  check(sqpv::Store::create(path, m, options, store) == sqpv::Status::ok);
  admin::Json selection{
      {"root", path},
      {"expected_metadata_reference", m.reference()},
      {"expected_store_generation", admin::hex(options.store_generation)},
      {"max_read_bytes", "1048576"},
      {"max_batches", "8"}};
  auto inspect = [&](const reader::Visitor &visitor = reader::Visitor{}) {
    return reader::inspect(selection, admin::engine::unix_time_ms() + 4000,
                           visitor);
  };
  auto refused = [&](auto action, const std::string &code = "") {
    bool caught = false;
    try {
      action();
    } catch (const admin::engine::Error &e) {
      caught = true;
      if (!code.empty())
        check(e.code() == code);
    }
    check(caught);
  };
  refused([&] { (void)inspect(); }, "sqv.store.busy");
  sqfv::Context flow;
  check(sqfv::Context::create({2048, 4096, 2048, 1048576, 2}, flow) ==
        sqfv::Status::ok);
  sqfv::Descriptor descriptor;
  check(m.binding(descriptor.binding) == sqmv::Status::ok);
  descriptor.partition = options.partition;
  descriptor.producer_generation = options.producer_generation;
  descriptor.record_count = 1;
  descriptor.source_binding = "test";
  for (std::uint64_t seq = 7; seq < 9; ++seq) {
    descriptor.batch_sequence = seq;
    sqfv::Batch b;
    std::array<std::uint8_t, 3> bytes{0, 1, 255};
    check(flow.prepare_copy(descriptor, bytes, b) == sqfv::Status::ok);
    sqpv::Receipt receipt;
    check(store.append(flow, b, receipt) == sqpv::Status::ok);
  }
  store.reset();
  auto head = content(path + "/head");
  const auto clean_files = contents(path);
  auto clean = inspect();
  check(clean.committed_batches == 2 && !clean.recovery_required &&
        clean.next_sequence == 9 && contents(path) == clean_files);
  selection["max_read_bytes"] = std::to_string(clean.physical_bytes);
  check(inspect().committed_batches == 2);
  selection["max_read_bytes"] = std::to_string(clean.physical_bytes - 1);
  refused([&] { (void)inspect(); }, "sqv.store.limit");
  selection["max_read_bytes"] = "1048576";
  selection["max_batches"] = "1";
  refused([&] { (void)inspect(); }, "sqv.store.limit");
  selection["max_batches"] = "8";
  selection["expected_store_generation"] = std::string(32, '0');
  refused([&] { (void)inspect(); }, "sqv.store.corrupt");
  selection["expected_store_generation"] = admin::hex(options.store_generation);
  check(contents(path) == clean_files);

  // No-follow admission must hold at every component and for every retained
  // file, including the existing lock. A failed observation creates nothing.
  for (const auto &link : {root.path + "/root-link", root.path + "/parent-link"}) {
    const bool ancestor = link.ends_with("parent-link");
    std::filesystem::create_directory_symlink(ancestor ? parent : path, link);
    selection["root"] = ancestor ? link + "/store" : link;
    refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
    selection["root"] = path;
    std::filesystem::remove(link);
  }
  for (const auto &directory : {parent, path}) {
    check(::chmod(directory.c_str(), 0770) == 0);
    refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
    check(::chmod(directory.c_str(), 0700) == 0);
  }
  const std::string first_frame = "frame-00000000000000000007";
  const std::string first_commit = "commit-00000000000000000007";
  for (const auto &name : {std::string("metadata"), std::string("head"),
                         std::string("writer.lock"), first_frame, first_commit}) {
    const auto file = path + "/" + name, saved = parent + "/saved-file";
    check(::rename(file.c_str(), saved.c_str()) == 0);
    std::filesystem::create_symlink(saved, file);
    refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
    std::filesystem::remove(file);
    check(::rename(saved.c_str(), file.c_str()) == 0);
  }
  for (const auto &name : {"writer.lock", "head"}) {
    const auto file = path + "/" + name, alias = parent + "/hardlink";
    std::filesystem::create_hard_link(file, alias);
    refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
    std::filesystem::remove(alias);
    check(::chmod(file.c_str(), 0640) == 0);
    refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
    check(::chmod(file.c_str(), 0600) == 0);
  }
  const auto saved = parent + "/saved-file";
  check(::rename((path + "/" + first_frame).c_str(), saved.c_str()) == 0);
  check(::mkfifo((path + "/" + first_frame).c_str(), 0600) == 0);
  refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
  std::filesystem::remove(path + "/" + first_frame);
  check(::rename(saved.c_str(), (path + "/" + first_frame).c_str()) == 0);
  check(::rename((path + "/writer.lock").c_str(), saved.c_str()) == 0);
  const auto without_lock = contents(path);
  refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
  check(!std::filesystem::exists(path + "/writer.lock") &&
        contents(path) == without_lock);
  check(::rename(saved.c_str(), (path + "/writer.lock").c_str()) == 0);
  write(path + "/writer.lock", "not an empty lock");
  refused([&] { (void)inspect(); }, "sqv.store.corrupt");
  write(path + "/writer.lock", "");
  write(path + "/unexpected", "");
  refused([&] { (void)inspect(); }, "sqv.store.corrupt");
  std::filesystem::remove(path + "/unexpected");
  check(contents(path) == clean_files);

  // Each acknowledged format component is verified, and a missing referenced
  // suffix cannot silently shorten the published range.
  for (const auto &name : {std::string("metadata"), std::string("head"),
                         first_commit, first_frame}) {
    const auto file = path + "/" + name;
    auto damaged = clean_files.at(name);
    damaged[0] ^= 1;
    write(file, damaged);
    refused([&] { (void)inspect(); }, "sqv.store.corrupt");
    check(content(file) == damaged);
    write(file, clean_files.at(name));
  }
  for (const auto &name : {first_frame, first_commit}) {
    const auto file = path + "/" + name;
    check(::rename(file.c_str(), saved.c_str()) == 0);
    refused([&] { (void)inspect(); }, "sqv.store.corrupt");
    check(::rename(saved.c_str(), file.c_str()) == 0);
  }
  unsigned calls = 0;
  (void)inspect([&](const auto &, const auto &, auto) {
    if (calls++ == 0) {
      check(inspect().committed_batches == 2);
      sqpv::Store writer;
      check(sqpv::Store::open(path, m, options, writer) == sqpv::Status::busy);
    }
  });
  check(calls == 2 && head == content(path + "/head"));
  // A cooperating writer cannot acquire ownership; a noncooperating file
  // replacement must invalidate the observation, even when bytes are identical.
  for (const auto &name : {"head", "writer.lock"}) {
    calls = 0;
    refused(
        [&] {
          (void)inspect([&](const auto &, const auto &, auto) {
            if (calls++ == 0)
              replace(path + "/" + name);
          });
        },
        "sqv.store.changed");
  }
  check(inspect().committed_batches == 2);
  calls = 0;
  refused(
      [&] {
        (void)inspect([&](const auto &, const auto &, auto) {
          if (calls++ == 0) {
            check(::rename(path.c_str(), (path + "-moved").c_str()) == 0);
            check(::mkdir(path.c_str(), 0700) == 0);
          }
        });
      },
      "sqv.store.changed");
  check(::rmdir(path.c_str()) == 0 &&
        ::rename((path + "-moved").c_str(), path.c_str()) == 0);
  calls = 0;
  refused([&] {
    (void)inspect([&](const auto &, const auto &, auto) {
      if (calls++ == 0)
        check(::rename(parent.c_str(), (parent + "-moved").c_str()) == 0);
    });
  });
  check(::rename((parent + "-moved").c_str(), parent.c_str()) == 0);
  // Recognized staging and a missing head remain observation-only dirty state.
  for (const auto &stage : {std::string("stage-head"), std::string("stage-commit"),
                           "stage-frame-00000000000000000009-" +
                               std::string(64, '0')}) {
    write(path + "/" + stage, "unvalidated partial tail");
    const auto staged = contents(path);
    const auto dirty = inspect();
    check(dirty.recovery_required && dirty.committed_batches == 2 &&
          reader::summary(dirty)["tail_validated"] == false &&
          contents(path) == staged);
    std::filesystem::remove(path + "/" + stage);
  }
  check(::rename((path + "/head").c_str(), saved.c_str()) == 0);
  const auto no_head = contents(path);
  const auto missing_head = inspect();
  check(missing_head.recovery_required && missing_head.committed_batches == 0 &&
        missing_head.next_sequence == 7 && contents(path) == no_head);
  check(::rename(saved.c_str(), (path + "/head").c_str()) == 0);
  // A fully written commit beyond the published head stays unacknowledged.
  std::string body = head.substr(0, 72);
  body.append(7, '\0');
  body.push_back('\1');
  body += clean_files.at(first_commit).substr(clean_files.at(first_commit).size() - 64);
  auto lagging = body + admin::engine::sha256_hex(body);
  write(path + "/head", lagging);
  const auto unpublished = contents(path);
  calls = 0;
  auto dirty = inspect([&](const auto &, const auto &batch, auto) {
    check(batch.descriptor().batch_sequence == 7);
    ++calls;
  });
  check(dirty.recovery_required && dirty.committed_batches == 1 && calls == 1 &&
        dirty.next_sequence == 8 && contents(path) == unpublished &&
        reader::summary(dirty)["integrity_scope"] == "published_head_prefix_only");
  refused(
      [&] {
        (void)reader::inspect(selection, admin::engine::unix_time_ms() - 1);
      },
      "request.deadline_expired");
  // Interrupted creation lacks an authoritative metadata file; inspection may
  // not create the lock or promote the writer's input-bound metadata stage.
  const auto incomplete = parent + "/incomplete";
  check(::mkdir(incomplete.c_str(), 0700) == 0);
  write(incomplete + "/stage-metadata-" +
            admin::engine::sha256_hex(clean_files.at("metadata")),
        clean_files.at("metadata"));
  selection["root"] = incomplete;
  const auto initial_creation = contents(incomplete);
  refused([&] { (void)inspect(); }, "sqv.store.unsafe_or_missing");
  check(contents(incomplete) == initial_creation);
  write(incomplete + "/writer.lock", "");
  const auto locked_creation = contents(incomplete);
  refused([&] { (void)inspect(); }, "sqv.store.corrupt");
  check(contents(incomplete) == locked_creation);
  std::cout << "Read-only identity, no-follow, locking, integrity, publication "
               "boundary, race and finite bounds passed\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
