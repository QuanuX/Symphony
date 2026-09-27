#include <symphony/sqpv/local_store.hpp>

#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
#include <stdexcept>
#include <string>
#include <system_error>
#include <vector>
#include <unistd.h>

namespace sqpv = symphony::sqpv;
namespace sqmv = symphony::sqmv;
namespace sqfv = symphony::sqfv;

namespace {
sqmv::Description metadata_description() {
  return {"fixture-dataset", "revision-7", "schema-1", "row-v1", "fixture-scope",
          "fixture-producer",
          {{sqmv::EvidenceRole::schema, "schema-owner", "schema-evidence-1"},
           {sqmv::EvidenceRole::layout, "layout-owner", "layout-evidence-1"},
           {sqmv::EvidenceRole::access, "access-owner", "scope-evidence-1"}}};
}

constexpr sqmv::Limits metadata_limits{4096, 256, 8};

std::string read_bytes(const std::filesystem::path& path) {
  std::ifstream stream(path, std::ios::binary);
  if (!stream) throw std::runtime_error("cannot read fixture file");
  return {std::istreambuf_iterator<char>(stream), std::istreambuf_iterator<char>()};
}

struct TemporaryRoot {
  std::filesystem::path path;
  TemporaryRoot() {
    std::string name = (std::filesystem::temp_directory_path() /
                        "sqpv-installed-consumer-XXXXXX").string();
    std::vector<char> writable(name.begin(), name.end());
    writable.push_back('\0');
    if (!::mkdtemp(writable.data()))
      throw std::system_error(errno, std::generic_category(), "mkdtemp");
    path = std::filesystem::canonical(writable.data());
  }
  ~TemporaryRoot() {
    std::error_code ignored;
    std::filesystem::remove_all(path, ignored);
  }
};

template <typename Status>
bool expect(Status actual, Status wanted, const char* label) {
  if (actual == wanted) return true;
  std::cerr << label << ": expected " << static_cast<int>(wanted)
            << ", received " << static_cast<int>(actual) << '\n';
  return false;
}

bool installed_store_reopen_and_replay() {
  TemporaryRoot root;
  sqmv::Manifest manifest;
  if (!expect(sqmv::Manifest::create(metadata_description(), metadata_limits, manifest),
              sqmv::Status::ok, "manifest create")) return false;
  sqfv::Binding binding;
  if (!expect(manifest.binding(binding), sqmv::Status::ok, "metadata binding"))
    return false;
  const sqpv::Options options{"partition-a", {1}, {7}, 4, {4096, 1'048'576, 8}};
  sqpv::Store store;
  if (!expect(sqpv::Store::create(root.path.string(), manifest, options, store),
              sqpv::Status::ok, "store create")) return false;
  sqfv::Context context;
  if (!expect(sqfv::Context::create({1024, 4096, 1024, 65'536, 2}, context),
              sqfv::Status::ok, "batch context")) return false;
  sqfv::Descriptor descriptor{binding, "partition-a", "fixture-source", "position-4",
                              {1}, 4, 1};
  const std::array<std::uint8_t, 5> payload{3, 1, 4, 1, 5};
  sqfv::Batch batch;
  if (!expect(context.prepare_copy(descriptor, payload, batch), sqfv::Status::ok,
              "bound batch")) return false;
  sqpv::Receipt committed;
  if (!expect(store.append(context, batch, committed), sqpv::Status::ok,
              "append commit") || committed.batch_sequence != 4 ||
      committed.content_id != batch.content_id()) return false;
  sqpv::Receipt repeated;
  if (!expect(store.append(context, batch, repeated), sqpv::Status::duplicate,
              "duplicate verified receipt") ||
      repeated.commit_sha256 != committed.commit_sha256) return false;
  store.reset();
  if (!expect(sqpv::Store::open(root.path.string(), manifest, options, store),
              sqpv::Status::ok, "store reopen")) return false;
  sqfv::Batch restored;
  sqpv::Receipt replayed;
  if (!expect(store.read(4, context, restored, replayed), sqpv::Status::ok,
              "committed replay") ||
      replayed.commit_sha256 != committed.commit_sha256 ||
      restored.content_id() != batch.content_id()) return false;
  sqfv::Lease lease;
  if (!expect(restored.acquire(binding.access_scope, lease), sqfv::Status::ok,
              "replayed payload lease") ||
      !std::equal(payload.begin(), payload.end(), lease.payload().begin(), lease.payload().end()))
    return false;
  sqpv::Snapshot snapshot;
  if (!expect(store.snapshot(snapshot), sqpv::Status::ok, "store snapshot") ||
      snapshot.committed_batches != 1 || snapshot.next_sequence != 5 || snapshot.staged)
    return false;

  descriptor.binding.dataset_revision = "different-revision";
  sqfv::Batch mismatched;
  if (!expect(context.prepare_copy(descriptor, payload, mismatched), sqfv::Status::ok,
              "mismatched batch fixture")) return false;
  return expect(store.append(context, mismatched, replayed), sqpv::Status::binding_mismatch,
                "store binding refusal") &&
         replayed.commit_sha256 == committed.commit_sha256;
}

bool installed_store_rejects_busy_and_wrong_options() {
  TemporaryRoot root;
  TemporaryRoot sentinel_root;
  sqmv::Manifest manifest;
  if (!expect(sqmv::Manifest::create(metadata_description(), metadata_limits, manifest),
              sqmv::Status::ok, "busy fixture metadata")) return false;
  const sqpv::Options options{"partition-a", {1}, {7}, 4, {4096, 1'048'576, 8}};
  auto sentinel_options = options;
  sentinel_options.first_sequence = 99;
  sqpv::Store live;
  sqpv::Store output;
  if (!expect(sqpv::Store::create(root.path.string(), manifest, options, live),
              sqpv::Status::ok, "busy fixture store") ||
      !expect(sqpv::Store::create(sentinel_root.path.string(), manifest,
                                 sentinel_options, output), sqpv::Status::ok,
              "preserved output store")) return false;
  const auto preserved = [&output] {
    sqpv::Snapshot snapshot;
    return expect(output.snapshot(snapshot), sqpv::Status::ok, "preserved output handle") &&
           snapshot.next_sequence == 99 && snapshot.committed_batches == 0;
  };
  if (!expect(sqpv::Store::open(root.path.string(), manifest, options, output),
              sqpv::Status::busy, "second writer refused") || !preserved()) return false;
  live.reset();
  for (const bool change_store : std::array{true, false}) {
    auto wrong_options = options;
    if (change_store) ++wrong_options.store_generation[0];
    else ++wrong_options.producer_generation[0];
    if (!expect(sqpv::Store::open(root.path.string(), manifest, wrong_options, output),
                sqpv::Status::binding_mismatch, "different generation refused") ||
        !preserved()) return false;
  }
  auto changed = metadata_description();
  changed.dataset_revision = "different-revision";
  sqmv::Manifest wrong_manifest;
  if (!expect(sqmv::Manifest::create(changed, metadata_limits, wrong_manifest),
              sqmv::Status::ok, "different manifest fixture")) return false;
  return expect(sqpv::Store::open(root.path.string(), wrong_manifest, options, output),
                sqpv::Status::binding_mismatch, "different manifest refused") && preserved();
}

bool installed_store_rejects_corrupt_retention() {
  TemporaryRoot root;
  TemporaryRoot sentinel_root;
  sqmv::Manifest manifest;
  if (!expect(sqmv::Manifest::create(metadata_description(), metadata_limits, manifest),
              sqmv::Status::ok, "corruption fixture metadata")) return false;
  sqfv::Binding binding;
  if (!expect(manifest.binding(binding), sqmv::Status::ok, "corruption fixture binding"))
    return false;
  const sqpv::Options options{"partition-a", {1}, {7}, 4, {4096, 1'048'576, 8}};
  sqpv::Store store;
  sqfv::Context context;
  if (!expect(sqpv::Store::create(root.path.string(), manifest, options, store),
              sqpv::Status::ok, "corruption fixture store") ||
      !expect(sqfv::Context::create({1024, 4096, 1024, 65'536, 2}, context),
              sqfv::Status::ok, "corruption fixture context")) return false;
  const sqfv::Descriptor descriptor{binding, "partition-a", "fixture-source", "position-4",
                                    {1}, 4, 1};
  const std::array<std::uint8_t, 5> payload{3, 1, 4, 1, 5};
  sqfv::Batch batch;
  sqpv::Receipt committed;
  if (!expect(context.prepare_copy(descriptor, payload, batch), sqfv::Status::ok,
              "corruption fixture batch") ||
      !expect(store.append(context, batch, committed), sqpv::Status::ok,
              "corruption fixture commit")) return false;
  store.reset();
  std::filesystem::path frame;
  for (const auto& entry : std::filesystem::directory_iterator(root.path)) {
    if (entry.path().filename().string().starts_with("frame-")) frame = entry.path();
  }
  if (frame.empty()) return false;
  const auto head_before = read_bytes(root.path / "head");
  auto changed_frame = read_bytes(frame);
  if (changed_frame.empty()) return false;
  changed_frame.back() ^= 1;
  {
    std::fstream stream(frame, std::ios::binary | std::ios::in | std::ios::out);
    if (!stream) return false;
    stream.seekp(-1, std::ios::end);
    stream.put(changed_frame.back());
    stream.flush();
    if (!stream) return false;
  }
  auto sentinel_options = options;
  sentinel_options.first_sequence = 99;
  sqpv::Store output;
  if (!expect(sqpv::Store::create(sentinel_root.path.string(), manifest,
                                 sentinel_options, output), sqpv::Status::ok,
              "corruption output sentinel")) return false;
  if (!expect(sqpv::Store::open(root.path.string(), manifest, options, output),
              sqpv::Status::corrupt, "corrupt retained frame refused")) return false;
  sqpv::Snapshot snapshot;
  return expect(output.snapshot(snapshot), sqpv::Status::ok, "corruption output preserved") &&
         snapshot.next_sequence == 99 && snapshot.committed_batches == 0 &&
         read_bytes(frame) == changed_frame && read_bytes(root.path / "head") == head_before;
}
} // namespace

int main() {
  try {
    if (!installed_store_reopen_and_replay() ||
        !installed_store_rejects_busy_and_wrong_options() ||
        !installed_store_rejects_corrupt_retention()) return 1;
    std::cout << "installed SQPV C++26 consumer: exact manifest binding, commit receipt, reopen/replay, preserved failure output\n";
  } catch (const std::exception& error) {
    std::cerr << error.what() << '\n';
    return 1;
  }
}
