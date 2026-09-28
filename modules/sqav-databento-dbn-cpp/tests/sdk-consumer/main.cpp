#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <symphony/sqav/databento/historical.hpp>
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

std::vector<std::uint8_t> fixture_v1() {
  // Unchanged Databento C++ v0.68.0 public test_data.mbo.v1.dbn.
  constexpr std::string_view hex =
      "44424e01c6000000474c42582e4d4450330000000000000000000020a0acdbe2541600"
      "008fc4df06551602000000000000000200000000000000010000000000000000000000"
      "0000000000000000000000000000000000000000000000000000000000000000000000"
      "0000000000000001000000455348310000000000000000000000000000000000000000"
      "0000000000000100000045534831000000000000000000000000000000000000010000"
      "000c3f34010d3f3401353438320000000000000000000000000000000000000ea00100"
      "6a15000007afa6acdbe254168945fed29600000080fb30c56203000001000000800043"
      "413cdeaaacdbe25416d1590000b0db11000ea001006a15000031b6a6acdbe254163f45"
      "fed29600000000ae17d4620300000100000080004341b0faaaacdbe25416a54c0000b1"
      "db1100";
  std::vector<std::uint8_t> b;
  b.reserve(hex.size()/2);
  auto n=[](char c){return c<='9'?c-'0':c-'a'+10;};
  for(std::size_t i=0;i<hex.size();i+=2)
    b.push_back(static_cast<std::uint8_t>((n(hex[i])<<4)|n(hex[i+1])));
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
  const auto v1=fixture_v1();
  auto d1=d;d1.source.native_encoding_ref=db::native_encoding_v1;
  sqav::Capture c1;
  check(db::capture_file(d1,v1,limits,{65536,16384,4096},c1)==db::Status::ok &&
        std::ranges::equal(c1.original(),v1));
  check(db::inspect_capture(c1,limits,view)==db::Status::ok && view.metadata().version==1);
  db::Mbo a,z;
  check(view.record(0,a)==db::Status::ok && db::inspect_capture(c,limits,view)==db::Status::ok &&
        view.record(0,z)==db::Status::ok && a==z);
  check(db::capture_file(d1,b,limits,{65536,16384,4096},c1)==db::Status::binding_mismatch);
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
  malformed[3] = 2;
  check(db::FileView::inspect(malformed, limits, v) == db::Status::unsupported);
}
void installed_historical_admission() {
  auto bytes=fixture();db::FileView file;
  check(db::FileView::inspect(bytes,limits,file)==db::Status::ok);
  const auto &meta=file.metadata();
  db::HistoricalSelection selection{std::string(meta.dataset),{"ESH1"},meta.start,meta.end,meta.limit};
  db::HistoricalLimits policy{limits,86'400'000'000'000ULL,8,2,60};
  db::HistoricalPlan plan;check(db::HistoricalPlan::create(selection,policy,plan)==db::Status::ok);
  db::HistoricalResponse response;
  check(db::HistoricalResponse::begin(plan,1,200,{},response)==db::Status::ok);
  check(response.append(bytes)==db::Status::ok);
  db::HistoricalReport report;
  check(response.finish(db::TransportEnd::complete,report)==db::Status::ok &&
      report.coverage==sqav::Coverage::partial && report.record_limit_reached &&
      report.recovery==db::Recovery::split_window);
  db::HistoricalAttribution observer{"installed-attempt","fixture-observer","public-v0.68.0","private:fixture",
      {sqav::TimeRole::acquisition,"2026-09-28","iso-date","UTC-calendar","day","fixture"}};
  sqav::Capture capture;check(response.capture(observer,{65536,16384,4096},capture)==db::Status::ok &&
      std::ranges::equal(capture.original(),bytes));
  const auto saved=std::string(capture.reference());
  check(db::HistoricalResponse::begin(plan,1,200,{},response)==db::Status::ok && response.append(bytes)==db::Status::ok);
  check(response.finish(db::TransportEnd::interrupted,report)==db::Status::ok && report.coverage==sqav::Coverage::gap);
  check(response.capture(observer,{65536,16384,4096},capture)==db::Status::binding_mismatch && capture.reference()==saved);
  selection.symbols={"AAPL"};check(db::HistoricalPlan::create(selection,policy,plan)==db::Status::ok);
  check(db::HistoricalResponse::begin(plan,1,200,{},response)==db::Status::ok && response.append(bytes)==db::Status::ok);
  check(response.finish(db::TransportEnd::complete,report)==db::Status::ok && report.outcome==db::HistoricalOutcome::binding_mismatch);
  check(response.capture(observer,{65536,16384,4096},capture)==db::Status::binding_mismatch && capture.reference()==saved);
}
int main() {
  installed_fidelity_and_rejection();
  installed_capture_binding();
  installed_historical_admission();
  std::puts("DBN installed consumer: public fixture fidelity and atomic "
            "rejection passed");
}
