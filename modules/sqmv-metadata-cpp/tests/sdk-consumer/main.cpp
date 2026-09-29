#include <symphony/sqmv/metadata.hpp>

#include <algorithm>
#include <array>
#include <iostream>
#include <string>
#include <vector>

namespace sqmv = symphony::sqmv;
namespace sqfv = symphony::sqfv;

namespace {
constexpr sqmv::Limits limits{4096, 256, 8};

sqmv::Description description() {
  return {"fixture-dataset", "revision-7", "schema-1", "row-v1",
          "fixture-scope", "fixture-producer",
          {{sqmv::EvidenceRole::schema, "schema-owner", "schema-evidence-1"},
           {sqmv::EvidenceRole::layout, "layout-owner", "layout-evidence-1"},
           {sqmv::EvidenceRole::access, "access-owner", "scope-evidence-1"}}};
}

template <typename Status>
bool expect(Status actual, Status wanted, const char* label) {
  if (actual == wanted) return true;
  std::cerr << label << ": expected " << static_cast<int>(wanted)
            << ", received " << static_cast<int>(actual) << '\n';
  return false;
}

bool installed_manifest_round_trip_and_binding() {
  auto input = description();
  sqmv::Manifest manifest;
  if (!expect(sqmv::Manifest::create(input, limits, manifest), sqmv::Status::ok,
              "manifest create")) return false;
  std::reverse(input.evidence.begin(), input.evidence.end());
  sqmv::Manifest reordered;
  if (!expect(sqmv::Manifest::create(input, limits, reordered), sqmv::Status::ok,
              "reordered manifest") || manifest.reference() != reordered.reference())
    return false;
  sqmv::Manifest resolved;
  if (!expect(sqmv::Manifest::resolve(manifest.encoded(), manifest.reference(),
                                    limits, resolved), sqmv::Status::ok,
              "exact manifest resolve")) return false;
  sqfv::Binding binding;
  if (!expect(resolved.binding(binding), sqmv::Status::ok, "binding bridge") ||
      binding.metadata_ref != manifest.reference() ||
      binding.dataset_revision != "revision-7") return false;
  sqfv::Context context;
  if (!expect(sqfv::Context::create({1024, 4096, 1024, 65'536, 2}, context),
              sqfv::Status::ok, "batch context")) return false;
  sqfv::Descriptor descriptor{binding, "partition-a", "fixture-source", "position-7",
                              {1}, 7, 1};
  const std::array<std::uint8_t, 4> payload{2, 3, 5, 7};
  sqfv::Batch batch;
  if (!expect(context.prepare_copy(descriptor, payload, batch), sqfv::Status::ok,
              "bound batch") ||
      !expect(resolved.verify_binding(batch.descriptor().binding), sqmv::Status::ok,
              "batch metadata binding")) return false;
  return true;
}

bool installed_manifest_failures_preserve_output() {
  sqmv::Manifest manifest;
  if (!expect(sqmv::Manifest::create(description(), limits, manifest),
              sqmv::Status::ok, "sentinel manifest")) return false;
  const std::string reference(manifest.reference());
  std::vector<std::uint8_t> corrupted(manifest.encoded().begin(), manifest.encoded().end());
  corrupted.back() ^= 1;
  if (!expect(sqmv::Manifest::resolve(corrupted, reference, limits, manifest),
              sqmv::Status::corrupt_manifest, "corrupt manifest refusal") ||
      manifest.reference() != reference) return false;
  auto wrong_reference = reference;
  wrong_reference.back() = wrong_reference.back() == '0' ? '1' : '0';
  if (!expect(sqmv::Manifest::resolve(manifest.encoded(), wrong_reference, limits,
                                    manifest), sqmv::Status::reference_mismatch,
              "wrong exact reference refusal") ||
      manifest.reference() != reference) return false;
  sqfv::Binding changed;
  if (!expect(manifest.binding(changed), sqmv::Status::ok, "negative binding"))
    return false;
  changed.dataset_revision = "different-revision";
  return expect(manifest.verify_binding(changed), sqmv::Status::binding_mismatch,
                "changed binding refusal");
}
} // namespace

int main() {
  if (!installed_manifest_round_trip_and_binding() ||
      !installed_manifest_failures_preserve_output()) return 1;
  std::cout << "installed SQMV C++26 consumer: exact metadata resolution, SQFV binding, preserved failure outputs\n";
}
