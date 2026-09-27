#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <symphony/sqav/capture.hpp>
#include <vector>
using namespace symphony;
void check(bool p) {
  if (!p)
    std::abort();
}
sqav::Description description() {
  return {{"provider", "interface", "v1", "read-original", "adapter", "v2",
           "dataset", "vintage-1", "selection", "native-schema", "binary",
           "private"},
          "attempt-1",
          "observer",
          "native-position-99",
          sqav::Coverage::gap,
          "requested-range",
          "gap-evidence",
          std::nullopt,
          {{sqav::TimeRole::acquisition, "2026-09-26", "iso-date",
            "UTC-calendar", "day", "clock-evidence"}}};
}
sqav::Capture fixture() {
  sqav::Capture c;
  const std::vector<std::uint8_t> raw{0, 255, 127, 128};
  check(sqav::Capture::create(description(), raw, {65536, 16384, 4096}, c) ==
        sqav::Status::ok);
  // Golden values independently encoded from SPEC with struct.pack/hashlib.
  check(c.reference() ==
        "sqac1-sha256-"
        "db710c82fe68387142c177b34b93e815c470b264d4c207d32804ae9fe7d46edc");
  check(c.source_reference() ==
        "sqas1-sha256-"
        "c6b60a5711f7a957dbc5ec24b3393073a5c4c9a99eae6b67b0c7c294dc656293");
  return c;
}
sqmv::Manifest metadata(const sqav::Capture &c) {
  sqmv::Description d{
      "dataset",
      "vintage-1",
      std::string(sqav::capture_schema),
      std::string(sqav::capture_layout),
      "private",
      "observer",
      {{sqmv::EvidenceRole::schema, "format-owner", "schema-evidence"},
       {sqmv::EvidenceRole::layout, "format-owner", "layout-evidence"},
       {sqmv::EvidenceRole::access, "observer", "access-evidence"},
       {sqmv::EvidenceRole::source, "observer",
        std::string(c.source_reference())}}};
  sqmv::Manifest m;
  check(sqmv::Manifest::create(d, {65536, 4096, 128}, m) == sqmv::Status::ok);
  return m;
}
void installed_capture_roundtrip() {
  auto c = fixture();
  auto m = metadata(c);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 4}, flow) ==
        sqfv::Status::ok);
  sqav::Position p{"partition", {}, 41};
  p.producer_generation[0] = 1;
  sqfv::Batch b;
  check(c.prepare(flow, m, p, b) == sqav::Status::ok);
  sqfv::Port port;
  sqfv::PortConfig cfg{
      b.descriptor().binding, p.partition, p.producer_generation, 41, 65536, 2};
  check(flow.add_port(cfg, port) == sqfv::Status::ok);
  check(port.offer(b) == sqfv::Status::ok);
  sqfv::Lease lease;
  check(port.take(lease) == sqfv::Status::ok);
  sqav::Capture out;
  check(sqav::Capture::from_delivery(lease.payload(), lease.descriptor(), m,
                                     {65536, 16384, 4096},
                                     out) == sqav::Status::ok);
  check(std::ranges::equal(c.original(), out.original()));
  check(out.description().coverage == sqav::Coverage::gap &&
        out.description().source_position == "native-position-99" &&
        lease.descriptor().batch_sequence == 41);
  check(!out.description().source_record_count &&
        out.description().times[0].precision_ref == "day");
}
void installed_capture_rejection() {
  auto c = fixture();
  auto out = fixture();
  auto ref = std::string(out.reference());
  auto b = std::vector<std::uint8_t>(c.encoded().begin(), c.encoded().end());
  b[b.size() - 33] ^= 1;
  check(sqav::Capture::resolve(b, c.reference(), {65536, 16384, 4096}, out) ==
        sqav::Status::corrupt_capture);
  check(out.reference() == ref);
  auto d = description();
  d.source.dataset_revision = "vintage-2";
  sqav::Capture changed;
  check(sqav::Capture::create(d, c.original(), {65536, 16384, 4096}, changed) ==
        sqav::Status::ok);
  check(changed.source_reference() != c.source_reference());
  auto m = metadata(c);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 4}, flow) ==
        sqfv::Status::ok);
  sqav::Position p{"partition", {}, 1};
  p.producer_generation[0] = 1;
  sqfv::Batch out_batch;
  check(changed.prepare(flow, m, p, out_batch) ==
        sqav::Status::binding_mismatch);
}
int main(int argc, char **argv) {
  installed_capture_roundtrip();
  installed_capture_rejection();
  if (argc == 2) {
    auto c = fixture();
    auto f = std::fopen(argv[1], "wb");
    check(f);
    check(std::fwrite(c.encoded().data(), 1, c.encoded().size(), f) ==
          c.encoded().size());
    check(std::fclose(f) == 0);
    std::printf("%.*s\n%.*s\n", static_cast<int>(c.reference().size()),
                c.reference().data(),
                static_cast<int>(c.source_reference().size()),
                c.source_reference().data());
  }
  std::puts("sqav installed consumer: 2 groups passed");
}
