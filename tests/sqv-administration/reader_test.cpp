#include "store_reader.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
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
    char p[] = "/private/tmp/sqv-reader-XXXXXX";
    auto *v = ::mkdtemp(p);
    check(v != nullptr);
    path = v;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
std::string content(const std::string &path) {
  std::ifstream f(path, std::ios::binary);
  return {(std::istreambuf_iterator<char>(f)), {}};
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
  check(inspect().committed_batches == 2);
  selection["max_batches"] = "1";
  refused([&] { (void)inspect(); }, "sqv.store.limit");
  selection["max_batches"] = "8";
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
  calls = 0;
  refused(
      [&] {
        (void)inspect([&](const auto &, const auto &, auto) {
          if (calls++ == 0) {
            auto data = content(path + "/head");
            std::ofstream f(path + "/replacement", std::ios::binary);
            f << data;
            f.close();
            check(::chmod((path + "/replacement").c_str(), 0600) == 0);
            check(::rename((path + "/replacement").c_str(),
                           (path + "/head").c_str()) == 0);
          }
        });
      },
      "sqv.store.changed");
  check(inspect().committed_batches == 2);
  calls = 0;
  refused([&] {
    (void)inspect([&](const auto &, const auto &, auto) {
      if (calls++ == 0)
        check(::rename(parent.c_str(), (parent + "-moved").c_str()) == 0);
    });
  });
  check(::rename((parent + "-moved").c_str(), parent.c_str()) == 0);
  // A fully written commit beyond the published head stays unacknowledged.
  std::string body = head.substr(0, 72);
  body.append(8, '\0');
  body.append(64, '0');
  auto lagging = body + admin::engine::sha256_hex(body);
  {
    std::ofstream f(path + "/head", std::ios::binary);
    f << lagging;
  }
  auto dirty = inspect();
  check(dirty.recovery_required && dirty.committed_batches == 0 &&
        dirty.next_sequence == 7 && content(path + "/head") == lagging);
  refused(
      [&] {
        (void)reader::inspect(selection, admin::engine::unix_time_ms() - 1);
      },
      "request.deadline_expired");
  std::cout << "Read-only locking, publication boundary, race and finite "
               "bounds passed\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
