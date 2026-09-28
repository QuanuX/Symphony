#include "request.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <symphony/sqav/databento/historical.hpp>
#include <symphony/sqav/databento/reference.hpp>
#include <symphony/sqav/fred.hpp>
#include <fstream>
#include <iostream>
#include <stdexcept>
using namespace symphony::sqav;
using request::Json;
void check(bool v) { if (!v) throw std::runtime_error("assertion failed"); }
Json fixture(const char *name) { std::ifstream f(std::string(SQAV_FIXTURES)+"/"+name+".json"); return Json::parse(f); }
void reject(const Json &j) {
  try { (void)request::validate(j); } catch (const request::engine::Error &e) {
    check(std::string(e.what()).find("private-marker") == std::string::npos); return;
  }
  throw std::runtime_error("invalid request accepted");
}
int main() {
  try {
    for (const auto *name : {"fred", "databento_historical", "databento_reference"}) {
      const auto p=fixture(name); const auto r=request::validate(p);
      check(r.at("request_digest") == request::engine::tagged_sha256(p.dump()));
      auto unsigned_result=r; unsigned_result.erase("result_digest");
      check(r.at("result_digest") == request::engine::tagged_sha256(unsigned_result.dump()));
      check(r.at("provider_observation") == "not_performed");
      check(request::validate(p)==r);
      auto bad=p; bad["api_key"]="private-marker"; reject(bad);
      bad=p; bad["selection"]["extra"]="private-marker"; reject(bad);
      bad=p; bad["adapter"]="private-marker"; reject(bad);
    }
    auto f=fixture("fred");
    fred::Selection fs; fs.operation=fred::Operation::observations; fs.series="GDP";
    fs.observation_start="2020-01-01"; fs.observation_end="2020-12-31";
    fs.realtime_start="2021-01-01"; fs.realtime_end="2021-01-01"; fs.limit=10; fs.offset=0;
    fred::Plan fp; check(fred::Plan::create(fs,fp)==fred::Status::ok);
    check(request::validate(f).at("plan_reference")==fp.reference());
    auto bad=f; bad["selection"]["limit"]=4294967296ULL; reject(bad);
    bad=f; bad["selection"]["limit"]=-1; reject(bad);
    bad=f; bad["selection"]["limit"]=1.0; reject(bad);
    bad=f; bad["selection"]["realtime_start"]="2021-02-30"; reject(bad);
    f["selection"]["operation"]="vintage_dates"; f["selection"]["observation_start"]=""; f["selection"]["observation_end"]=""; (void)request::validate(f);
    auto h=fixture("databento_historical");
    databento::HistoricalSelection hs; hs.dataset="GLBX.MDP3"; hs.symbols={"ESZ0"}; hs.start=1600000000000000000ULL; hs.end=1600000001000000000ULL; hs.record_limit=100;
    databento::HistoricalLimits hl; hl.dbn={1048576,65536,1000}; hl.max_window_ns=1000000000; hl.max_symbols=1; hl.max_attempts=1; hl.max_retry_after_seconds=1;
    databento::HistoricalPlan hp; check(databento::HistoricalPlan::create(hs,hl,hp)==databento::Status::ok);
    check(request::validate(h).at("plan_reference")==hp.reference());
    for (const auto *value : {"01600000000000000000", "18446744073709551616", "-1", "1e9"}) { bad=h; bad["selection"]["start_ns"]=value; reject(bad); }
    bad=h; bad["selection"]["start_ns"]=1600000000000000000ULL; reject(bad);
    bad=h; bad["limits"]["max_attempts"]=257; reject(bad);
    bad=h; bad["limits"]["max_symbols"]=65537; reject(bad);
    bad=h; bad["selection"]["symbols"]={"ESZ0","ESZ0"}; reject(bad);
    auto ref=fixture("databento_reference");
    databento::reference::Selection rs; rs.operation=databento::reference::Operation::security_master_range; rs.symbols={"AAPL"}; rs.start="2020-01-01"; rs.end="2020-01-02";
    databento::reference::Plan rp; check(databento::reference::Plan::create(rs,rp)==databento::reference::Status::ok);
    check(request::validate(ref).at("plan_reference")==rp.reference());
    for (const auto *op : {"corporate_actions","adjustment_factors"}) { auto p=ref; p["selection"]["operation"]=op; (void)request::validate(p); }
    ref["selection"]["operation"]="security_master_last"; ref["selection"]["start"]=""; ref["selection"]["end"]=""; (void)request::validate(ref);
    auto d=request::descriptor(); check(d.at("operations").size()==1);
    std::cout << "PASS native request parity, precision, bounds, refusal and descriptor\n";
  } catch (const std::exception &e) { std::cerr << e.what() << '\n'; return 1; }
}
