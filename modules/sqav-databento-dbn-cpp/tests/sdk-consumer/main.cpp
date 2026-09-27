#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <symphony/sqav/databento/dbn.hpp>
#include <vector>
using namespace symphony;
namespace db = symphony::sqav::databento;
constexpr db::Limits limits{65536, 4096, 100};
void check(bool p) {
  if (!p)
    std::abort();
}
std::vector<std::uint8_t> fixture() {
  // Public Databento C++ v0.68.0 test_data.mbo.v3.dbn, unchanged file bytes.
  // https://github.com/databento/databento-cpp/blob/v0.68.0/tests/data/test_data.mbo.v3.dbn
  constexpr std::string_view hex =
      "44424e0360010000474c42582e4d4450330000000000000000000020a0acdbe254160000"
      "8fc4df065516020000000000000001000047000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000100000045534831000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000010000004553483100000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000010000000c3f34010d3f3401353438320000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "0ea001006a15000007afa6acdbe254168945fed29600000080fb30c56203000001000000"
      "800043413cdeaaacdbe25416d1590000b0db11000ea001006a15000031b6a6acdbe25416"
      "3f45fed29600000000ae17d4620300000100000080004341b0faaaacdbe25416a54c0000"
      "b1db1100";
  std::vector<std::uint8_t> b;
  b.reserve(hex.size() / 2);
  auto n = [](char c) { return c <= '9' ? c - '0' : c - 'a' + 10; };
  for (std::size_t i = 0; i < hex.size(); i += 2)
    b.push_back(static_cast<std::uint8_t>((n(hex[i]) << 4) | n(hex[i + 1])));
  return b;
}

void installed_capture_binding() {
  auto b = fixture();
  sqav::Description d{{"databento", "public-sdk-fixture", "0.68.0",
                       "offline-file-inspect", "caller", "caller", "GLBX.MDP3",
                       "fixture-v0.68.0", "fixture:ESH1",
                       std::string(db::native_schema),
                       std::string(db::native_encoding), "private:fixture"},
                      "attempt",
                      "observer",
                      "",
                      sqav::Coverage::unknown,
                      "fixture-range",
                      "",
                      std::nullopt,
                      {{sqav::TimeRole::acquisition, "2026-09-27", "iso-date",
                        "UTC-calendar", "day", "local"}}};
  sqav::Capture c;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, c) ==
        db::Status::ok);
  check(std::ranges::equal(c.original(), b) &&
        c.description().source_record_count == 2);
  db::FileView view;
  check(db::inspect_capture(c, limits, view) == db::Status::ok);
  const auto ref = std::string(c.reference());
  d.source.dataset_id = "WRONG";
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, c) ==
            db::Status::binding_mismatch &&
        c.reference() == ref);
}
void installed_fidelity_and_rejection() {
  auto b = fixture();
  db::FileView v;
  check(db::FileView::inspect(b, limits, v) == db::Status::ok);
  db::Mbo m;
  check(v.record(0, m) == db::Status::ok && m.order_id == 647784973705ULL &&
        m.price == 3722750000000LL && m.ts_recv == 1609160400000704060ULL &&
        m.sequence == 1170352);
  auto malformed = b;
  malformed[360] = 0;
  check(db::FileView::inspect(malformed, limits, v) == db::Status::malformed &&
        v.original().data() == b.data());
  malformed = b;
  malformed[3] = 1;
  check(db::FileView::inspect(malformed, limits, v) == db::Status::unsupported);
}
int main() {
  installed_fidelity_and_rejection();
  installed_capture_binding();
  std::puts("DBN installed consumer: public fixture fidelity and atomic "
            "rejection passed");
}
