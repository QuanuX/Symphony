#include <fstream>
#include <iostream>
#include <map>
#include <stdexcept>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/sniv.hpp>
#include <symphony/snv/snrv.hpp>
#include <symphony/snv/sciv.hpp>
#include <symphony/snv/scnv.hpp>
#include <symphony/snv/snv.hpp>
#include <symphony/snv/local_observer.hpp>
namespace e = symphony::knowledge::engine;
namespace s = symphony::snv;
namespace o = s::local_observer;
using Json = e::Json;
void require(bool v, const char* message) { if (!v) throw std::runtime_error(message); }
struct Reader final : o::SourceReader {
  bool refreshed = false;
  o::Platform platform() const override { return {true,"synthetic","synthetic"}; }
  o::ReadResult read(o::Source source,std::size_t,std::int64_t) override {
    switch(source) {
    case o::Source::boot: return {o::ReadStatus::ok,"00000000-1111-2222-3333-444444444444\n"};
    case o::Source::cpu_present: return {o::ReadStatus::ok,refreshed?"0-7\n":"0-3\n"};
    case o::Source::cpu_online: return {o::ReadStatus::permission_denied,{}};
    case o::Source::meminfo: return {o::ReadStatus::ok,refreshed?"MemAvailable: 256 kB\n":"MemAvailable: 512 kB\n"};
    }
    throw std::runtime_error("unselected source");
  }
};
int main(int argc,char** argv) {
 try {
  require(argc==2,"exact supplied bundle required");
  std::ifstream file(argv[1]);const auto input=Json::parse(file);
  using Handler = Json(*)(std::string_view,const Json&,std::int64_t);
  const std::map<std::string,Handler> owners{{"sniv",s::sniv::handle},{"snrv",s::snrv::handle},{"sciv",s::sciv::handle},{"scnv",s::scnv::handle}};
  std::size_t replays=0;
  for(const auto& artifact:input.at("bundle").at("artifacts")) {
    const auto owner=artifact.at("owner").get<std::string>();
    const auto payload=Json::parse(artifact.at("source_utf8").get<std::string>());
    const auto result=owners.at(owner)(artifact.at("operation").get<std::string>(),payload,e::unix_time_ms()+5000);
    require(result.at("owner")==owner && result.at("owner_version")=="0.1.0-dev","installed child owner identity");
    ++replays;
  }
  require(replays==4,"all four original child records required");
  const auto parent=s::snv::handle("snv_inspect",input,e::unix_time_ms()+5000);
  require(parent.at("owner")=="symphony-snv" && parent.at("owner_version")=="0.1.0-dev","installed parent replay");
  Reader reader;
  Json request{{"protocol","symphony.snv.local-observe-input.v1"},{"observation_id","sdk-before"},{"node_ref",nullptr},{"profile","linux-proc-sysfs-v1"},{"fields",Json::array({"cpu_present","cpu_online","memory_available"})}};
  const auto before=o::collect(request,reader,e::unix_time_ms()+5000);
  const auto original=before.dump();reader.refreshed=true;request["observation_id"]="sdk-after";
  const auto after=o::collect(request,reader,e::unix_time_ms()+5000);
  for(const auto* result:{&before,&after}) {
    require(result->at("status")=="partial" && result->at("acquisition_route")=="sdk_supplied_reader","partial injected scope");
    require(result->at("subject_ids").empty() && result->at("physical_inventory_complete")==false && result->at("allocation_verified")==false,"no invented identity or entitlement");
    bool unseen=false;
    for(const auto& field:result->at("fields")) if(field.at("field")=="cpu_online") {
      require(field.at("status")=="permission_denied" && field.at("value").is_null(),"unseen field must not become zero or absent");unseen=true;
    }
    require(unseen,"unseen field retained");
  }
  require(before.dump()==original && before.at("fields")!=after.at("fields"),"refresh preserves prior observation");
  std::cout<<Json{{"scope","installed SDK supplied replay; observer injection separate from native host"},{"semantic_engines_replayed",5},{"synthetic_before",before},{"synthetic_after",after}}.dump()<<'\n';
 } catch(const std::exception& ex) {std::cerr<<ex.what()<<'\n';return 1;}
}
