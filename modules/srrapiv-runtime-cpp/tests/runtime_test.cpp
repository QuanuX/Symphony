#include "catalog_fixture.hpp"
#include "private_io.hpp"
#include <filesystem>
#include <iostream>
#include <sys/wait.h>
#include <thread>

namespace engine=symphony::knowledge::engine;
namespace owner=symphony::srrapiv;
namespace io=owner::private_io;
using Json=owner::Json;
namespace {
unsigned assertions=0;
void check(bool ok,const char *what) { ++assertions;if(!ok)throw std::runtime_error(what); }
Json call(const std::string &exe,const std::string &operation,Json payload) {
  payload["protocol"]="symphony.srrapiv.request.v1";
  Json request={{"protocol",engine::process_protocol_v1},{"request_id","test-request"},{"correlation_id","test-correlation"},
    {"operation",operation},{"target_engine","symphony-srrapiv"},{"deadline_unix_ms",engine::unix_time_ms()+10000},{"payload",payload}};
  int input[2],output[2];check(::pipe(input)==0&&::pipe(output)==0,"pipes");
  auto pid=::fork();check(pid>=0,"fork");
  if(pid==0) {
    ::dup2(input[0],0);::dup2(output[1],1);::close(input[0]);::close(input[1]);::close(output[0]);::close(output[1]);
    ::execl(exe.c_str(),exe.c_str(),static_cast<char*>(nullptr));_exit(127);
  }
  ::close(input[0]);::close(output[1]);const auto raw=request.dump();
  std::size_t at=0;
  while(at<raw.size()) {auto n=::write(input[1],raw.data()+at,raw.size()-at);check(n>0,"write request");at+=static_cast<std::size_t>(n);}
  ::close(input[1]);std::string response;char b[8192];
  for(;;) {auto n=::read(output[0],b,sizeof b);if(n<0&&errno==EINTR)continue;check(n>=0,"read response");if(!n)break;response.append(b,static_cast<std::size_t>(n));check(response.size()<4194304,"response bound");}
  ::close(output[0]);int status=0;check(::waitpid(pid,&status,0)==pid,"wait");
  check(!response.empty(),"response absent");auto value=engine::parse_bounded_json(response,4194304,65536);
  auto digest=value.at("response_digest");auto copy=value;copy.erase("response_digest");check(digest==engine::tagged_sha256(copy.dump()),"response digest");
  check(value.at("request_id")=="test-request","correlation");return value;
}
Json good(const Json &response) {
  if(response.at("outcome")!="ok")throw std::runtime_error("native operation "+response.value("operation","")+" failed: "+response.at("error").value("code","unknown"));
  check(true,"native operation accepted");return response.at("result").at("result");
}
void refused(const Json &response,const char *code) {check(response.at("outcome")=="error","expected native refusal");check(response.at("error").at("code")==code,"wrong refusal");}
}
int main(int argc,char **argv) {
  std::string base,exe;std::vector<std::pair<std::string,std::string>> active;
  try {
    check(argc==2||(argc==3&&std::string(argv[2])=="--slow"),"executable argument");
    const bool slow=argc==3;exe=std::filesystem::canonical(argv[1]).string();
    std::string pattern="/private/tmp/srrapiv-test-XXXXXX";
#ifndef __APPLE__
    pattern="/tmp/srrapiv-test-XXXXXX";
#endif
    auto made=::mkdtemp(pattern.data());check(made!=nullptr,"private test directory");base=made;
    auto api=good(call(exe,"api_describe",Json::object()));
    check(api.at("sdk_version")=="13.7.0.0","SDK identity");
    auto artifact=owner::test_fixture::artifact();
    artifact["adapter_version"]="0.1.0-dev";artifact["adapter_digest"]=api.at("runtime_build").at("adapter_digest");
    artifact["platform"]["os"]=api.at("runtime_build").at("os");artifact["platform"]["architecture"]=api.at("runtime_build").at("architecture");
    artifact["header_digest"]="sha256:"+api.at("header_sha256").get<std::string>();
    if(api.contains("selected_library_sha256"))artifact["library_digest"]="sha256:"+api.at("selected_library_sha256").get<std::string>();
    owner::test_fixture::seal(artifact,"artifact_id");
    auto config=owner::test_fixture::input(artifact);
    config["allow_provider_calls"]=true;config["allow_trading"]=false;
    config["engine"]=Json::object();config["secrets"]=Json::object();
    auto configfile=base+"/config.json";io::write(configfile,config);
    auto a=base+"/runtime-a",b=base+"/runtime-b";
    auto started=good(call(exe,"runtime_start",{{"runtime_dir",a},{"config_file",configfile}}));
    check(started.at("state")=="running","detached runtime started");
    auto generation=started.at("generation").get<std::string>();active.push_back({a,generation});
    auto status=good(call(exe,"runtime_status",{{"runtime_dir",a}}));
    check(status.at("generation")==generation,"recover generation without prior response");
    check(status.at("binding_id")==config.at("binding").at("binding_id"),"exact binding selection");
    refused(call(exe,"runtime_start",{{"runtime_dir",a},{"config_file",configfile}}),"srrapiv.runtime.already_running");
    check(!std::filesystem::exists(a+"/runtime-failure.json"),"failed contender does not overwrite owner diagnostics");
    auto second=good(call(exe,"runtime_start",{{"runtime_dir",b},{"config_file",configfile}}));
    auto second_generation=second.at("generation").get<std::string>();active.push_back({b,second_generation});
    check(second_generation!=generation,"concurrent generation isolation");
    refused(call(exe,"runtime_status",{{"runtime_dir",a},{"expected_generation",second_generation}}),"srrapiv.generation.stale");
    Json invocation={{"runtime_dir",a},{"expected_generation",generation},{"method","REngine.getVersion#1"},
      {"params",Json::object()},{"attempt_id","attempt-one"},{"authority",{{"allow_provider",false}}}};
    refused(call(exe,"runtime_invoke",invocation),"srrapiv.authority.denied");
    invocation["authority"]["allow_provider"]=true;
    auto first=good(call(exe,"runtime_invoke",invocation));
    if(slow) {
      check(first.at("state")=="in_progress","slow dispatch returns bounded in-progress receipt");
      const auto before=std::chrono::steady_clock::now();
      auto responsive=good(call(exe,"runtime_status",{{"runtime_dir",a},{"expected_generation",generation}}));
      check(std::chrono::steady_clock::now()-before<std::chrono::milliseconds(300),"control status blocked behind SDK worker");
      check(responsive.at("worker_active")==true,"status identifies active worker");
      auto distinct=invocation;distinct["attempt_id"]="busy-distinct-attempt";
      refused(call(exe,"runtime_invoke",distinct),"srrapiv.runtime.busy");
      auto blocked=good(call(exe,"runtime_stop",{{"runtime_dir",a},{"expected_generation",generation}}));
      check(blocked.at("state")=="stop_blocked_by_active_invocation"&&blocked.at("ok")==false,"stop does not race SDK worker");
      auto duplicate=good(call(exe,"runtime_invoke",invocation));
      check(duplicate.at("replayed_locally")==true,"active attempt is not dispatched twice");
    }
    auto completion_deadline=std::chrono::steady_clock::now()+std::chrono::seconds(3);
    while(first.at("state")!="sdk_returned"&&std::chrono::steady_clock::now()<completion_deadline) {
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
      first=good(call(exe,"runtime_attempts",{{"runtime_dir",a},{"attempt_id","attempt-one"}}));
    }
    check(first.at("state")=="sdk_returned","SDK result observed without provider init");
    auto repeat=good(call(exe,"runtime_invoke",invocation));check(repeat.at("replayed_locally")==true,"local retry returns retained result");
    check(repeat.at("request_digest")==first.at("request_digest"),"retry identity");
    invocation["params"]["changed"]=true;
    refused(call(exe,"runtime_invoke",invocation),"srrapiv.attempt.conflict");
    invocation["params"]=Json::object();invocation["method"]="REngine.sendOrder#1";invocation["attempt_id"]="forbidden-order";
    refused(call(exe,"runtime_invoke",invocation),"srrapiv.authority.denied");
    invocation["method"]="REngine.lock#1";invocation["params"]={{"operations",Json::array({{{"operation","REngine.sendOrder#1"},{"params",Json::object()}}})}};
    refused(call(exe,"runtime_invoke",invocation),"srrapiv.authority.denied");
    auto events=good(call(exe,"runtime_events",{{"runtime_dir",a},{"expected_generation",generation},{"after_sequence","0"},{"limit",1}}));
    if(slow) {
      check(events.at("events").size()==1&&events.at("lost_events")=="1","oversized callback dropped with explicit gap");
      auto page=good(call(exe,"runtime_events",{{"runtime_dir",a},{"expected_generation",generation},{"after_sequence","0"},{"limit",128}}));
      check(page.at("events").size()==1&&page.at("next_sequence")=="1","callback page respects aggregate value budget");
      auto next=good(call(exe,"runtime_events",{{"runtime_dir",a},{"expected_generation",generation},{"after_sequence",page.at("next_sequence")},{"limit",128}}));
      check(next.at("events").size()==1&&next.at("next_sequence")=="2"&&next.at("gap")==true,"callback cursor preserves next valid event and gap");
    } else check(events.at("events").empty(),"offline host has no fabricated callbacks");
    auto attempts=good(call(exe,"runtime_attempts",{{"runtime_dir",a}}));check(attempts.at("attempts").size()==1,"only admitted request recorded");
    refused(call(exe,"runtime_attempts",{{"runtime_dir",a},{"expected_generation",second_generation}}),"srrapiv.generation.stale");
    if(slow) {
      auto large=invocation;large["method"]="REngine.getVersion#1";large["params"]=Json::object();large["attempt_id"]="large-second-attempt";
      auto second_result=good(call(exe,"runtime_invoke",large));
      const auto deadline=std::chrono::steady_clock::now()+std::chrono::seconds(3);
      while(second_result.at("state")!="sdk_returned"&&std::chrono::steady_clock::now()<deadline) {
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
        second_result=good(call(exe,"runtime_attempts",{{"runtime_dir",a},{"attempt_id","large-second-attempt"}}));
      }
      check(second_result.at("state")=="sdk_returned","second large result completed");
      auto page=good(call(exe,"runtime_attempts",{{"runtime_dir",a},{"limit",128}}));
      check(page.at("attempts").size()==1&&page.at("has_more")==true,"attempt page respects aggregate value budget");
      auto next=good(call(exe,"runtime_attempts",{{"runtime_dir",a},{"after_key",page.at("next_key")},{"limit",128}}));
      check(next.at("attempts").size()==1&&next.at("has_more")==false,"attempt value pagination makes progress");
    }
    // Model crash immediately after durable dispatch admission; an identical
    // attempt must stay unknown even though the running SDK could now execute it.
    auto uncertain=first;uncertain["state"]="outcome_unknown";uncertain["ok"]=false;uncertain.erase("sdk_result");
    io::write(a+"/attempts/"+engine::sha256_hex("attempt-one")+".json",uncertain);
    invocation["method"]="REngine.getVersion#1";invocation["params"]=Json::object();invocation["attempt_id"]="attempt-one";
    auto recovered=good(call(exe,"runtime_invoke",invocation));check(recovered.at("state")=="outcome_unknown"&&recovered.at("replayed_locally")==true,"unknown never reissued");
    auto bad=config;bad["binding"]["environment"]="altered";io::write(base+"/invalid.json",bad);
    refused(call(exe,"runtime_start",{{"runtime_dir",base+"/invalid"},{"config_file",base+"/invalid.json"}}),"srrapiv.binding.invalid");
    check(::chmod(configfile.c_str(),0644)==0,"make unsafe mode");
    refused(call(exe,"runtime_start",{{"runtime_dir",base+"/unsafe"},{"config_file",configfile}}),"srrapiv.path.unsafe");check(::chmod(configfile.c_str(),0600)==0,"restore mode");
    auto failed=base+"/failed-start";io::directory(failed,true);
    io::write(failed+"/control.sock",{{"private-marker","must-not-be-diagnostic"}});
    refused(call(exe,"runtime_start",{{"runtime_dir",failed},{"config_file",configfile}}),"srrapiv.runtime.start_failed");
    auto diagnostic=io::read_json(failed+"/runtime-failure.json");
    check(diagnostic.at("phase")=="binding_socket"&&diagnostic.at("code")=="srrapiv.path.unsafe","owner retains bounded startup failure");
    check(diagnostic.dump().find("private-marker")==std::string::npos,"startup diagnostic never echoes arbitrary input");
    auto observed_failure=good(call(exe,"runtime_status",{{"runtime_dir",failed}}));
    check(observed_failure.at("failure")==diagnostic,"status exposes matching-generation failure");
    auto early=base+"/early-failure";io::directory(early,true);io::directory(early+"/attempts",true);
    io::write(early+"/owner.lock",Json::object());
    auto child=::fork();check(child>=0,"early failure fork");
    if(child==0) {::execl(exe.c_str(),exe.c_str(),"--serve",early.c_str(),configfile.c_str(),"early-test-generation","invalid-config-digest",static_cast<char*>(nullptr));_exit(127);}
    int early_status=0;check(::waitpid(child,&early_status,0)==child&&WIFEXITED(early_status)&&WEXITSTATUS(early_status)==5,"early serve failure");
    check(!std::filesystem::exists(early+"/runtime.json"),"early failure precedes runtime record");
    auto early_observed=good(call(exe,"runtime_status",{{"runtime_dir",early},{"expected_generation","early-test-generation"}}));
    check(early_observed.at("state")=="startup_failed"&&early_observed.at("failure").at("code")=="srrapiv.binding.config_changed","status exposes failure before runtime record");
    for(const auto &[directory,gen]:active) {
      auto stopped=good(call(exe,"runtime_stop",{{"runtime_dir",directory},{"expected_generation",gen}}));
      check(stopped.at("state")=="stopped"||stopped.at("state")=="stopping","stop");
      if(slow)check(stopped.at("state")=="stopping","slow shutdown returns bounded stopping receipt");
      if(stopped.at("state")=="stopping") {
        bool terminal=false;const auto deadline=std::chrono::steady_clock::now()+std::chrono::seconds(3);
        while(std::chrono::steady_clock::now()<deadline) {
          auto observed=good(call(exe,"runtime_status",{{"runtime_dir",directory},{"expected_generation",gen}}));
          if(observed.contains("stop_result")&&observed.at("stop_result").value("state","")=="stopped") {terminal=true;break;}
          std::this_thread::sleep_for(std::chrono::milliseconds(10));
        }
        check(terminal,"stop worker completed before fixture removal");
      }
    }
    active.clear();
    auto retained=good(call(exe,"runtime_attempts",{{"runtime_dir",a},{"attempt_id","attempt-one"}}));check(retained.at("state")=="outcome_unknown","offline recovery survives stop");
    std::filesystem::remove_all(base);std::cout<<assertions<<" runtime assertions passed; no provider initialization or login\n";return 0;
  }catch(const std::exception &error) {
    for(const auto &[directory,gen]:active)try{(void)call(exe,"runtime_stop",{{"runtime_dir",directory},{"expected_generation",gen}});}catch(...){}
    std::cerr<<"runtime test failed: "<<error.what()<<"; fixture "<<base<<'\n';return 1;
  }
}
