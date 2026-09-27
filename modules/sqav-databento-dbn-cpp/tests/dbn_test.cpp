#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <symphony/knowledge/engine/digest.hpp>
#include <new>
#include <source_location>
#include <symphony/sqav/databento/dbn.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <unistd.h>
#include <vector>
namespace {
std::atomic<long> fail_at{-1}, allocations{0};
}
void *operator new(std::size_t n) {
  if (fail_at >= 0 && allocations.fetch_add(1) == fail_at)
    throw std::bad_alloc();
  if (auto p = std::malloc(n ? n : 1))
    return p;
  throw std::bad_alloc();
}
void *operator new[](std::size_t n) { return ::operator new(n); }
void operator delete(void *p) noexcept { std::free(p); }
void operator delete[](void *p) noexcept { std::free(p); }
namespace {
using namespace symphony;
namespace db = symphony::sqav::databento;
constexpr db::Limits limits{65536, 4096, 100};
void check(bool p, std::source_location loc = std::source_location::current()) {
  if (!p) {
    std::fprintf(stderr, "DBN test failure at line %u\n", loc.line());
    std::abort();
  }
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

sqav::Description description() {
  return {{"databento", "public-sdk-fixture", "0.68.0", "offline-file-inspect",
           "caller", "caller", "GLBX.MDP3", "fixture-v0.68.0", "fixture:ESH1",
           std::string(db::native_schema), std::string(db::native_encoding),
           "private:fixture"},
          "attempt-1",
          "fixture-observer",
          "",
          sqav::Coverage::unknown,
          "public-fixture-only",
          "",
          std::nullopt,
          {{sqav::TimeRole::acquisition, "2026-09-27", "iso-date",
            "UTC-calendar", "day", "local-read"}}};
}
void public_fixture_fidelity() {
  auto b = fixture();
  db::FileView view;
  check(db::FileView::inspect(b, limits, view) == db::Status::ok);
  const auto &m = view.metadata();
  check(m.dataset == "GLBX.MDP3" && m.start == 1609160400000000000ULL &&
        m.end == 1609200000000000000ULL && m.limit == 2 &&
        m.record_count == 2 && m.symbols == 1 && m.mappings == 1 &&
        m.partial == 0 && m.not_found == 0 && m.symbol_cstr_len == 71 &&
        !m.ts_out);
  db::Mbo a, z;
  check(view.record(0, a) == db::Status::ok &&
        view.record(1, z) == db::Status::ok);
  check(a.publisher_id == 1 && a.instrument_id == 5482 &&
        a.ts_event == 1609160400000429831ULL && a.order_id == 647784973705ULL &&
        a.price == 3722750000000LL && a.size == 1 && a.flags == 128 &&
        a.channel_id == 0 && a.action == 'C' && a.side == 'A' &&
        a.ts_recv == 1609160400000704060ULL && a.ts_in_delta == 22993 &&
        a.sequence == 1170352 && !a.ts_out);
  check(z.ts_event == 1609160400000431665ULL && z.order_id == 647784973631ULL &&
        z.price == 3723000000000LL && z.ts_recv == 1609160400000711344ULL &&
        z.ts_in_delta == 19621 && z.sequence == 1170353);
  check(view.record(2, a) == db::Status::invalid_argument);
  // Unaligned transport bytes do not permit direct reinterpret_cast to provider
  // structs.
  std::vector<std::uint8_t> unaligned{99};
  unaligned.insert(unaligned.end(), b.begin(), b.end());
  check(db::FileView::inspect(sqav::ByteView(unaligned).subspan(1), limits,
                              view) == db::Status::ok);
  check(view.record(0, z) == db::Status::ok && z == a);
}
void raw_fields_and_gateway_timestamp() {
  auto b = fixture();
  std::vector<std::uint8_t> record(b.begin() + 360, b.begin() + 416);
  record[24] = 0;
  for (std::size_t i = 25; i < 31; ++i)
    record[i] = 0;
  record[31] = 128;
  for (std::size_t i = 8; i < 24; ++i)
    record[i] = 255;
  for (std::size_t i = 40; i < 52; ++i)
    record[i] = 255;
  record[36] = 255;
  record[38] = 255;
  record[39] = 255;
  db::Mbo m;
  check(db::decode_mbo(record, false, m) == db::Status::ok &&
        m.price == INT64_MIN && m.order_id == UINT64_MAX &&
        m.ts_event == UINT64_MAX && m.ts_recv == UINT64_MAX &&
        m.ts_in_delta == -1 && m.flags == 255 && m.action == 255 &&
        m.side == 255);
  record[0] = 16;
  record.resize(64, 255);
  check(db::decode_mbo(record, true, m) == db::Status::ok &&
        m.ts_out == UINT64_MAX);
  b.resize(360);
  b[52] = 1;
  b.insert(b.end(), record.begin(), record.end());
  db::FileView v;
  check(db::FileView::inspect(b, limits, v) == db::Status::ok &&
        v.metadata().record_count == 1 && v.record(0, m) == db::Status::ok &&
        m.ts_out == UINT64_MAX);
}
void malformed_and_limits() {
  auto b = fixture();
  db::FileView original;
  check(db::FileView::inspect(b, limits, original) == db::Status::ok);
  std::size_t rejected = 0;
  for (std::size_t n = 0; n < b.size(); ++n) {
    db::FileView out = original;
    auto s = db::FileView::inspect(sqav::ByteView(b).first(n), limits, out);
    if (n == 360 || n == 416)
      check(s == db::Status::ok);
    else {
      check(s != db::Status::ok && out.original().size() == b.size());
      ++rejected;
    }
  }
  std::printf("DBN truncations rejected: %zu; two valid record-boundary "
              "prefixes accepted\n",
              rejected);
  auto bad = b;
  bad[3] = 2;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad[24] = 255;
  bad[25] = 255;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad[52] = 2;
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  std::fill(bad.begin() + 112, bad.begin() + 116, 255);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  std::fill(bad.begin() + 116, bad.begin() + 187, 65);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  bad[360] = 0;
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  bad[417] = 1;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad.push_back(0);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  auto l = limits;
  l.max_records = 1;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  l = limits;
  l.max_metadata_bytes = 128;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  l = limits;
  l.max_file_bytes = 471;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  // Parser and field access allocate no heap state, even with first-allocation
  // failure armed.
  allocations = 0;
  fail_at = 0;
  db::FileView view;
  auto status = db::FileView::inspect(b, limits, view);
  db::Mbo record;
  auto rs = view.record(0, record);
  fail_at = -1;
  check(status == db::Status::ok && rs == db::Status::ok && allocations == 0);
}
void capture_and_retained_delivery() {
  auto b = fixture();
  auto d = description();
  sqav::Capture capture;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
        db::Status::ok);
  check(std::ranges::equal(capture.original(), b) &&
        capture.description().source_record_count == 2 &&
        capture.description().coverage == sqav::Coverage::unknown);
  db::FileView view;
  check(db::inspect_capture(capture, limits, view) == db::Status::ok);
  const auto before = std::string(capture.reference());
  d.source.dataset_id = "WRONG";
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
            db::Status::binding_mismatch &&
        capture.reference() == before);
  d = description();
  d.source_record_count = 3;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
        db::Status::binding_mismatch);
  d = description();
  d.attempt_id.assign(4097, 'x');
  allocations = 0;
  fail_at = 0;
  const auto bounded =
      db::capture_file(d, b, limits, {65536, 16384, 4096}, capture);
  fail_at = -1;
  check(bounded == db::Status::limit && allocations == 0 &&
        capture.reference() == before);
  // Explicit partial-symbol evidence contradicts a complete coverage claim.
  auto partial = b;
  partial[187] = 1;
  partial.insert(partial.begin() + 191, 71, 0);
  partial.erase(partial.begin() + 424, partial.begin() + 431);
  partial[4] = 160;
  partial[5] = 1; // 416 metadata bytes after prefix.
  d = description();
  d.coverage = sqav::Coverage::complete;
  d.coverage_evidence_ref = "assertion";
  check(db::capture_file(d, partial, limits, {65536, 16384, 4096}, capture) ==
        db::Status::binding_mismatch);
  d.coverage = sqav::Coverage::partial;
  sqav::Capture partial_capture;
  check(db::capture_file(d, partial, limits, {65536, 16384, 4096},
                         partial_capture) == db::Status::ok);
  sqmv::Description md{
      "GLBX.MDP3",
      "fixture-v0.68.0",
      std::string(sqav::capture_schema),
      std::string(sqav::capture_layout),
      "private:fixture",
      "fixture-observer",
      {{sqmv::EvidenceRole::schema, "fixture", "capture-schema"},
       {sqmv::EvidenceRole::layout, "fixture", "capture-layout"},
       {sqmv::EvidenceRole::access, "fixture", "private"},
       {sqmv::EvidenceRole::source, "fixture-observer",
        std::string(capture.source_reference())}}};
  sqmv::Manifest manifest;
  check(sqmv::Manifest::create(md, {65536, 4096, 128}, manifest) ==
        sqmv::Status::ok);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 4}, flow) ==
        sqfv::Status::ok);
  sqav::Position pos{"partition", {}, 1};
  pos.producer_generation[0] = 1;
  sqfv::Batch batch;
  check(capture.prepare(flow, manifest, pos, batch) == sqav::Status::ok);
  char temp[] = "/private/tmp/sqv12-retention-XXXXXX";
  auto path = ::mkdtemp(temp);
  check(path);
  {
    sqpv::Options o{
        pos.partition, pos.producer_generation, {}, 1, {73728, 1U << 20, 4}};
    o.store_generation[0] = 1;
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create(path, manifest, o, source) ==
          sqdv::Status::ok);
    sqdv::RetainedBatch proof;
    check(source.commit(flow, batch, proof) == sqdv::Status::ok);
    sqdv::Config cfg{"view",
                     "recipient",
                     "capture",
                     pos.partition,
                     pos.producer_generation,
                     1,
                     sqdv::Profile::retained_before_delivery};
    sqdv::Session session;
    check(sqdv::Session::create(flow, manifest, cfg, {65536, 2}, &source,
                                nullptr, session) == sqdv::Status::ok &&
          session.offer_next(flow) == sqdv::Status::ok);
    sqdv::Delivery delivery;
    check(session.take(delivery) == sqdv::Status::ok);
    sqav::Capture replay;
    check(sqav::Capture::from_delivery(
              delivery.payload(), delivery.descriptor(), manifest,
              {65536, 16384, 4096}, replay) == sqav::Status::ok);
    check(std::ranges::equal(replay.original(), b) &&
          db::inspect_capture(replay, limits, view) == db::Status::ok);
    db::Mbo record;
    check(view.record(1, record) == db::Status::ok &&
          record.sequence == 1170353);
    check(session.acknowledge_processed(delivery) == sqdv::Status::ok);
  }
  std::filesystem::remove_all(path);
}
void version_one_fidelity_and_binding() {
  const auto b=fixture_v1();
  db::FileView v1,v3;
  check(db::FileView::inspect(b,limits,v1)==db::Status::ok);
  auto b3=fixture();
  check(db::FileView::inspect(b3,limits,v3)==db::Status::ok);
  const auto& m=v1.metadata();
  check(m.version==1 && v3.metadata().version==3 && m.dataset=="GLBX.MDP3" &&
        m.symbol_cstr_len==22 && m.stype_in==1 && m.stype_out==0 &&
        m.start==v3.metadata().start && m.end==v3.metadata().end &&
        m.limit==2 && m.record_count==2 && m.symbols==1 && m.mappings==1 &&
        !m.partial && !m.not_found && !m.ts_out && v1.encoded_metadata().size()==206);
  for(std::uint64_t i=0;i<2;++i) {
    db::Mbo a,z;check(v1.record(i,a)==db::Status::ok &&
        v3.record(i,z)==db::Status::ok && a==z);
  }
  std::size_t rejected=0;
  for(std::size_t n=0;n<b.size();++n) {
    auto out=v1;auto status=db::FileView::inspect(sqav::ByteView(b).first(n),limits,out);
    if(n==206 || n==262) check(status==db::Status::ok);
    else {check(status!=db::Status::ok && out.original().size()==b.size());++rejected;}
  }
  auto bad=b;bad[60]=2;
  check(db::FileView::inspect(bad,limits,v1)==db::Status::malformed);
  bad=b;std::fill(bad.begin()+112,bad.begin()+116,255);
  check(db::FileView::inspect(bad,limits,v1)==db::Status::malformed);
  bad=b;std::fill(bad.begin()+116,bad.begin()+138,65);
  check(db::FileView::inspect(bad,limits,v1)==db::Status::malformed);
  bad=b;bad.insert(bad.begin()+206,0);++bad[4];
  check(db::FileView::inspect(bad,limits,v1)==db::Status::malformed);
  // Reserved legacy bytes are preserved, not mistaken for symbol width/count.
  bad=b;std::fill(bad.begin()+50,bad.begin()+58,255);
  check(db::FileView::inspect(bad,limits,v1)==db::Status::ok);
  auto d=description();sqav::Capture c;
  check(db::capture_file(d,b,limits,{65536,16384,4096},c)==db::Status::binding_mismatch);
  d.source.native_encoding_ref=db::native_encoding_v1;
  check(db::capture_file(d,b,limits,{65536,16384,4096},c)==db::Status::ok &&
        std::ranges::equal(c.original(),b) && c.description().source.adapter_version=="0.2.0-dev");
  check(db::inspect_capture(c,limits,v1)==db::Status::ok && v1.metadata().version==1);
  const auto ref=std::string(c.reference());
  check(db::capture_file(d,b3,limits,{65536,16384,4096},c)==db::Status::binding_mismatch && c.reference()==ref);
  d.source.adapter_ref=db::adapter_id;d.source.adapter_version="0.1.0-dev";
  sqav::Capture old;
  check(sqav::Capture::create(d,b,{65536,16384,4096},old)==sqav::Status::ok);
  check(db::inspect_capture(old,limits,v1)==db::Status::binding_mismatch);
  // v1 ts_out starts at offset 60 and the same 64-byte MBO grammar applies.
  auto with_ts=b;with_ts.resize(206);with_ts[60]=1;
  for(std::size_t i=0;i<2;++i) {
    with_ts.insert(with_ts.end(),b.begin()+206+i*56,b.begin()+262+i*56);
    with_ts[206+i*64]=16;with_ts.insert(with_ts.end(),8,255);
  }
  allocations=0;fail_at=0;
  auto status=db::FileView::inspect(with_ts,limits,v1);db::Mbo record;
  auto rs=v1.record(1,record);fail_at=-1;
  check(status==db::Status::ok && rs==db::Status::ok && record.ts_out==UINT64_MAX && allocations==0);
  std::printf("DBNv1 truncations rejected: %zu; two valid prefixes accepted\n",rejected);
}

void allocation_rollback() {
  auto b = fixture();
  auto d = description();
  sqav::Capture out;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, out) ==
        db::Status::ok);
  auto ref = std::string(out.reference());
  long failures = 0;
  for (long n = 0; n < 256; ++n) {
    allocations = 0;
    fail_at = n;
    auto status = db::capture_file(d, b, limits, {65536, 16384, 4096}, out);
    fail_at = -1;
    if (status == db::Status::ok) {
      failures = n;
      break;
    }
    check(status == db::Status::no_memory && out.reference() == ref);
  }
  check(failures > 0);
  std::printf("DBN capture allocation failures: %ld\n", failures);
}
void verify_private_sample(const char* path, const char* dataset) {
  constexpr std::uint64_t maximum=8U<<20;
  const auto size=std::filesystem::file_size(path);
  check(size<=maximum);
  std::ifstream input(path,std::ios::binary);
  std::vector<std::uint8_t> b(static_cast<std::size_t>(size));
  check(static_cast<bool>(input.read(reinterpret_cast<char*>(b.data()),static_cast<std::streamsize>(b.size()))));
  check(input.peek()==std::char_traits<char>::eof());
  db::Limits dl{maximum,1U<<20,100000};
  db::FileView v;
  check(db::FileView::inspect(b,dl,v)==db::Status::ok);
  const auto& m=v.metadata();
  check(m.dataset==dataset && m.record_count==100000 && m.limit==100000 &&
        !m.partial && !m.not_found && m.start==1790344800000000000ULL &&
        m.end==1790345400000000000ULL);
  std::uint64_t first_recv=UINT64_MAX,last_recv=0,first_event=UINT64_MAX,last_event=0;
  std::string fields;fields.reserve(24U<<20);
  std::uint32_t instrument=0;
  for(std::uint64_t i=0;i<m.record_count;++i) {
    db::Mbo record;check(v.record(i,record)==db::Status::ok);
    check(record.ts_recv>=m.start && record.ts_recv<m.end &&
          record.ts_event>=m.start && record.ts_event<m.end);
    if(i==0) instrument=record.instrument_id;
    check(record.instrument_id==instrument);
    first_recv=std::min(first_recv,record.ts_recv);last_recv=std::max(last_recv,record.ts_recv);
    first_event=std::min(first_event,record.ts_event);last_event=std::max(last_event,record.ts_event);
    auto field=[&](auto value){fields+=std::to_string(value);fields+='|';};
    field(record.publisher_id);field(record.instrument_id);field(record.ts_event);
    field(record.order_id);field(record.price);field(record.size);field(record.flags);
    field(record.channel_id);field(record.action);field(record.side);field(record.ts_recv);
    field(record.ts_in_delta);field(record.sequence);fields+='\n';
  }
  sqav::Description d{{"databento","https://hist.databento.com/v0/","0; observed 2026-09-27",
       "timeseries.get_range","caller","caller",dataset,"provider-revision-unspecified",
       "2026-09-25T14:00:00Z/14:10:00Z;raw_symbol;limit=100000",std::string(db::native_schema),
       std::string(db::encoding_for_version(m.version)),"private:databento-user-research"},
       "sqv13-native-historical-test","user-authorized-bounded-probe","",sqav::Coverage::partial,
       "requested ten-minute regular-session interval","request-record-limit-reached",m.record_count,
       {{sqav::TimeRole::acquisition,"2026-09-27T23:06:50.517513Z","iso8601","local-UTC","microsecond",
          "sqv13-history-result;credential-origin=user-provided-private-pipe;ssiag-unbound"}}};
  sqav::Limits cl{maximum,16384,4096};sqav::Capture capture;
  check(db::capture_file(d,b,dl,cl,capture)==db::Status::ok);
  sqmv::Description md{dataset,"provider-revision-unspecified",std::string(sqav::capture_schema),
       std::string(sqav::capture_layout),d.source.access_scope,d.attribution_ref,
       {{sqmv::EvidenceRole::schema,"sqv13","capture-schema"},
        {sqmv::EvidenceRole::layout,"sqv13","capture-layout"},
        {sqmv::EvidenceRole::access,"user-request","private"},
        {sqmv::EvidenceRole::source,d.attribution_ref,std::string(capture.source_reference())}}};
  sqmv::Manifest manifest;check(sqmv::Manifest::create(md,{65536,4096,128},manifest)==sqmv::Status::ok);
  sqfv::Context flow;check(sqfv::Context::create({maximum,maximum+8192,4096,32U<<20,4},flow)==sqfv::Status::ok);
  sqav::Position pos{"sample",{},1};pos.producer_generation[0]=1;
  sqfv::Batch batch;check(capture.prepare(flow,manifest,pos,batch)==sqav::Status::ok);
  char temp[]="/private/tmp/sqv13-retained-XXXXXX";auto root=::mkdtemp(temp);check(root);
  {
    sqpv::Options options{pos.partition,pos.producer_generation,{},1,{maximum+8192,32U<<20,4}};
    options.store_generation[0]=1;sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create(root,manifest,options,source)==sqdv::Status::ok);
    sqdv::RetainedBatch proof;check(source.commit(flow,batch,proof)==sqdv::Status::ok);
    sqdv::Config config{"view","recipient","capture",pos.partition,pos.producer_generation,1,sqdv::Profile::retained_before_delivery};
    sqdv::Session session;check(sqdv::Session::create(flow,manifest,config,{maximum,2},&source,nullptr,session)==sqdv::Status::ok);
    check(session.offer_next(flow)==sqdv::Status::ok);sqdv::Delivery delivery;
    check(session.take(delivery)==sqdv::Status::ok);sqav::Capture replay;
    check(sqav::Capture::from_delivery(delivery.payload(),delivery.descriptor(),manifest,cl,replay)==sqav::Status::ok);
    check(std::ranges::equal(replay.original(),b) && replay.reference()==capture.reference() &&
          db::inspect_capture(replay,dl,v)==db::Status::ok && v.metadata().version==m.version);
    check(session.acknowledge_processed(delivery)==sqdv::Status::ok);
  }
  std::filesystem::remove_all(root);
  std::printf("{\"dataset\":\"%s\",\"version\":%u,\"records\":%llu,\"instrument_id\":%u,\"first_recv\":%llu,\"last_recv\":%llu,\"first_event\":%llu,\"last_event\":%llu,\"field_sha256\":\"%s\",\"original_sha256\":\"%s\",\"retained_replay_exact\":true,\"coverage\":\"partial-record-limit\"}\n",dataset,m.version,
       static_cast<unsigned long long>(m.record_count),instrument,static_cast<unsigned long long>(first_recv),static_cast<unsigned long long>(last_recv),
       static_cast<unsigned long long>(first_event),static_cast<unsigned long long>(last_event),
       knowledge::engine::sha256_hex(fields).c_str(),knowledge::engine::sha256_hex(b).c_str());
}

} // namespace
int main(int argc,char** argv) {
  if(argc==4 && std::string_view(argv[1])=="--sample") {
    verify_private_sample(argv[2],argv[3]);return 0;
  }
  check(argc==1);
  public_fixture_fidelity();
  version_one_fidelity_and_binding();
  raw_fields_and_gateway_timestamp();
  malformed_and_limits();
  capture_and_retained_delivery();
  allocation_rollback();
  std::puts("DBN native acceptance: 6 groups passed");
}
