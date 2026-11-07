#include "runtime.hpp"
#include "catalog.hpp"
#include "private_io.hpp"
#include "vendor.hpp"
#include <algorithm>
#include <csignal>
#include <cstring>
#include <deque>
#include <fstream>
#include <future>
#include <mutex>
#include <poll.h>
#include <random>
#include <sys/file.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <thread>
#ifdef __APPLE__
#include <mach-o/dyld.h>
#endif

namespace symphony::srrapiv {
namespace {
namespace io = private_io;
namespace engine = symphony::knowledge::engine;
using io::require;
constexpr std::size_t frame_limit = 1048576;
constexpr std::size_t retained_value_limit = 65000;
// Match engine parser accounting: container starts, object keys and scalar
// values count; container ends do not. Saturation bounds work and recursion.
std::size_t json_value_count(const Json &value,std::size_t limit=retained_value_limit,unsigned depth=0) {
  if(limit==0||depth>64)return limit+1;
  std::size_t count=1;
  if(value.is_object()) {
    for(auto entry=value.begin();entry!=value.end();++entry) {
      if(++count>limit)return limit+1;
      auto child=json_value_count(entry.value(),limit-count,depth+1);
      if(child>limit-count)return limit+1;
      count+=child;
    }
  } else if(value.is_array()) {
    for(const auto &entry:value) {
      auto child=json_value_count(entry,limit-count,depth+1);
      if(child>limit-count)return limit+1;
      count+=child;
    }
  }
  return count;
}
volatile std::sig_atomic_t interrupted = 0;
void interrupt(int) { interrupted = 1; }
std::string random_id() {
  std::random_device random; std::string out;
  const char *hex = "0123456789abcdef";
  for (int i=0; i<32; ++i) out += hex[random() & 15];
  return out;
}
std::string executable() {
#ifdef __APPLE__
  char buf[4096]; std::uint32_t size=sizeof buf;
  require(_NSGetExecutablePath(buf, &size)==0);
  return std::filesystem::canonical(buf).string();
#else
  char buf[4096]; auto n=::readlink("/proc/self/exe",buf,sizeof(buf)-1);
  require(n>0 && n<static_cast<ssize_t>(sizeof(buf)-1)); return std::string(buf,static_cast<std::size_t>(n));
#endif
}
sockaddr_un address(const std::string &directory) {
  sockaddr_un addr{}; addr.sun_family=AF_UNIX;
  auto p=directory+"/control.sock";
  require(p.size()<sizeof(addr.sun_path), "srrapiv.path.unsafe");
  std::memcpy(addr.sun_path,p.c_str(),p.size()+1); return addr;
}
void deadline_socket(int fd) {
  timeval value{}; value.tv_sec=3;
  require(::setsockopt(fd,SOL_SOCKET,SO_RCVTIMEO,&value,sizeof(value))==0 &&
          ::setsockopt(fd,SOL_SOCKET,SO_SNDTIMEO,&value,sizeof(value))==0, "srrapiv.transport.failed");
#ifdef SO_NOSIGPIPE
  int one=1; (void)::setsockopt(fd,SOL_SOCKET,SO_NOSIGPIPE,&one,sizeof(one));
#endif
}
bool same_uid(int fd) {
#ifdef __APPLE__
  uid_t uid{}; gid_t gid{}; return ::getpeereid(fd,&uid,&gid)==0 && uid==::getuid();
#else
  ucred credential{}; socklen_t length=sizeof(credential);
  return ::getsockopt(fd,SOL_SOCKET,SO_PEERCRED,&credential,&length)==0 && credential.uid==::getuid();
#endif
}
void send_frame(int fd,const Json &j) {
  const auto raw=j.dump()+"\n"; require(raw.size()<=frame_limit,"srrapiv.response.limit");
  std::size_t pos=0;
  while(pos<raw.size()) {
#ifdef MSG_NOSIGNAL
    auto n=::send(fd,raw.data()+pos,raw.size()-pos,MSG_NOSIGNAL);
#else
    auto n=::send(fd,raw.data()+pos,raw.size()-pos,0);
#endif
    if(n<0 && errno==EINTR) continue;
    require(n>0,"srrapiv.transport.uncertain"); pos+=static_cast<std::size_t>(n);
  }
}
Json receive_frame(int fd) {
  std::string raw; char buffer[8192];
  const auto deadline=std::chrono::steady_clock::now()+std::chrono::seconds(3);
  while(raw.size()<frame_limit) {
    auto remaining=std::chrono::duration_cast<std::chrono::milliseconds>(deadline-std::chrono::steady_clock::now()).count();
    require(remaining>0,"srrapiv.transport.uncertain");pollfd pfd{fd,POLLIN,0};
    auto ready=::poll(&pfd,1,static_cast<int>(remaining));if(ready<0&&errno==EINTR)continue;require(ready>0,"srrapiv.transport.uncertain");
    auto n=::recv(fd,buffer,std::min(sizeof(buffer),frame_limit-raw.size()),MSG_DONTWAIT);
    if(n<0&&(errno==EAGAIN||errno==EWOULDBLOCK))continue;
    if(n<0 && errno==EINTR) continue;
    require(n>0,"srrapiv.transport.uncertain"); raw.append(buffer,static_cast<std::size_t>(n));
    auto end=raw.find('\n');
    if(end!=std::string::npos) { require(end==raw.size()-1); raw.pop_back(); return engine::parse_bounded_json(raw,frame_limit,65536); }
  }
  throw engine::Error("srrapiv.request.limit","Control request exceeds its bound",3);
}
Json rpc(const std::string &dir,const Json &request) {
  io::directory(dir); const auto addr=address(dir);
  io::Fd fd(::socket(AF_UNIX,SOCK_STREAM,0)); require(fd.value>=0,"srrapiv.transport.failed");
  deadline_socket(fd.value);
  require(::connect(fd.value,reinterpret_cast<const sockaddr*>(&addr),sizeof(addr))==0,"srrapiv.runtime.unavailable");
  require(same_uid(fd.value),"srrapiv.authority.denied");
  auto wire=request;wire["expected_adapter_digest"]=runtime_build_identity().at("adapter_digest");
  wire["protocol"]="symphony.srrapiv.local-control.v1";
  send_frame(fd.value,wire); auto response=receive_frame(fd.value);
  require(response.is_object()&&response.contains("ok")&&response.at("ok").is_boolean(),"srrapiv.transport.failed");
  require(response.value("protocol","")=="symphony.srrapiv.local-control.v1"&&
    response.value("engine_version","")=="0.1.0-dev"&&response.at("adapter_digest")==wire.at("expected_adapter_digest"),"srrapiv.binding.build_mismatch");
  if(!response.at("ok").get<bool>()) throw engine::Error(response.value("code","srrapiv.runtime.failed"),"Native runtime refused request",3);
  return response.at("result");
}
struct Events {
  std::mutex mutex;
  std::deque<Json> queue;
  std::uint64_t sequence=0,lost=0;
  std::size_t bytes=0;
  std::string generation;
  void emit(Json event) noexcept {
    try {
      std::lock_guard lock(mutex); ++sequence;
      Json item={{"sequence",std::to_string(sequence)},{"generation",generation},
        {"observed_unix_ms",std::to_string(engine::unix_time_ms())},{"event",std::move(event)}};
      const auto raw=item.dump();auto size=raw.size();
      if(size>262144||json_value_count(item)>retained_value_limit) { ++lost; return; }
      // Keep retained pages serializable under the exact control profile.
      (void)engine::parse_bounded_json(raw,262144,retained_value_limit);
      while(!queue.empty()&&(queue.size()>=4096||bytes+size>4194304)) { bytes-=queue.front().dump().size(); queue.pop_front(); ++lost; }
      bytes+=size; queue.push_back(std::move(item));
    } catch(...) { std::lock_guard lock(mutex); ++lost; }
  }
  Json read(const Json &p) {
    auto after=p.contains("after_sequence")?io::decimal(p.at("after_sequence")):0;
    std::size_t limit=128;
    if(p.contains("limit")) { require(p.at("limit").is_number_unsigned()||p.at("limit").is_number_integer()); auto value=p.at("limit").get<std::int64_t>(); require(value>0&&value<=256); limit=static_cast<std::size_t>(value); }
    std::lock_guard lock(mutex); require(after<=sequence);
    Json values=Json::array(); std::size_t output_bytes=0,output_values=0;
    std::uint64_t next=after;
    bool gap=lost>0;
    for(const auto &event:queue) {
      auto n=io::decimal(event.at("sequence")); if(n<=after)continue;
      auto size=event.dump().size();auto count=json_value_count(event);
      if(values.size()>=limit||output_bytes+size>524288||count>retained_value_limit-output_values)break;
      if(n!=next+1)gap=true; next=n; output_bytes+=size;output_values+=count;values.push_back(event);
    }
    return {{"events",values},{"next_sequence",std::to_string(next)},{"latest_sequence",std::to_string(sequence)},
      {"lost_events",std::to_string(lost)},{"gap",gap},{"generation",generation},{"retention","bounded_memory_not_durable"}};
  }
};
Json attempt_records(const std::string &dir,const Json &p) {
  io::directory(dir); auto attempts=dir+"/attempts"; io::directory(attempts);
  if(p.contains("expected_generation"))require(io::read_json(dir+"/runtime.json").at("generation")==p.at("expected_generation"),"srrapiv.generation.stale");
  if(p.contains("attempt_id")) {
    auto id=io::token(p,"attempt_id",128);
    auto record=io::read_json(attempts+"/"+engine::sha256_hex(id)+".json");
    require(json_value_count(record)<=retained_value_limit,"srrapiv.response.limit");return record;
  }
  std::size_t limit=128; if(p.contains("limit")) { auto n=p.at("limit").get<int>(); require(n>0&&n<=256);limit=static_cast<std::size_t>(n); }
  const auto after=p.value("after_key",std::string{}); require(after.size()<=69);
  std::vector<std::string> names;std::size_t scanned=0;
  for(const auto &e:std::filesystem::directory_iterator(attempts)) {
    require(++scanned<=100128,"srrapiv.request.limit");
    const auto name=e.path().filename().string();
    if(name.starts_with(".write-"))continue;
    require(name.size()==69&&name.ends_with(".json"),"srrapiv.storage.corrupt");
    if(name>after) names.push_back(name);
    require(names.size()<=100000,"srrapiv.request.limit");
  }
  std::sort(names.begin(),names.end()); Json rows=Json::array();std::size_t bytes=0,values=0;std::string next=after;
  for(const auto &name:names) {
    auto row=io::read_json(attempts+"/"+name);auto size=row.dump().size();
    auto count=json_value_count(row);require(count<=retained_value_limit,"srrapiv.response.limit");
    if(rows.size()==limit||bytes+size>524288||count>retained_value_limit-values)break;
    bytes+=size;values+=count;rows.push_back(std::move(row));next=name;
  }
  return {{"attempts",rows},{"next_key",next},{"has_more",rows.size()<names.size()},{"provider_reconciliation","not_inferred"}};
}
bool trading_method(const std::string &method) {
  for(const auto *prefix:{"REngine.send","REngine.modify","REngine.cancel","REngine.exitPosition","REngine.submitQuote","REngine.linkOrders","REngine.createUserDefinedSpread"})
    if(method.starts_with(prefix))return true;
  return false;
}
void authorize(const Json &config,const Json &authority,const std::string &method,const Json &params,unsigned depth=0) {
  require(depth<=8,"srrapiv.request.limit");
  require(authority.value("allow_provider",false)&&config.value("allow_provider_calls",false),"srrapiv.authority.denied");
  if(trading_method(method))require(authority.value("allow_trading",false)&&config.value("allow_trading",false),"srrapiv.authority.denied");
  if(config.contains("allowed_methods")) {
    const auto &allowed=config.at("allowed_methods");require(allowed.is_array());
    require(std::find(allowed.begin(),allowed.end(),method)!=allowed.end(),"srrapiv.authority.denied");
  }
  if(params.contains("operations")) {
    require(params.at("operations").is_array()&&params.at("operations").size()<=64);
    for(const auto &nested:params.at("operations")) {
      require(nested.is_object()&&nested.contains("params")&&nested.at("params").is_object());
      authorize(config,authority,io::token(nested,"operation"),nested.at("params"),depth+1);
    }
  }
}
std::string public_config_digest(Json config) {
  config.erase("secrets");return engine::tagged_sha256(config.dump());
}
void check_build(const Json &config) {
  const auto build=runtime_build_identity();
  const auto id=config.at("binding").at("artifact_id");const auto &artifacts=config.at("catalog").at("artifacts");
  auto found=std::find_if(artifacts.begin(),artifacts.end(),[&](const Json &a){return a.at("artifact_id")==id;});
  require(found!=artifacts.end(),"srrapiv.binding.invalid");
  require(found->at("adapter_version")=="0.1.0-dev"&&found->at("adapter_digest")==build.at("adapter_digest"),"srrapiv.binding.build_mismatch");
  require(found->at("platform").at("os")==build.at("os")&&found->at("platform").at("architecture")==build.at("architecture"),"srrapiv.binding.build_mismatch");
#ifdef SYMPHONY_SRRAPIV_WITH_RAPI
  require(found->at("sdk_version")=="13.7.0.0"&&found->at("header_digest")=="sha256:b996e76fbfe20c7f4d7ba2d645c45cb13d6b19df37d7522125e94e61e74df5b4"&&
    found->at("library_digest")==std::string("sha256:")+SYMPHONY_SRRAPIV_LIBRARY_SHA256,"srrapiv.binding.build_mismatch");
#endif
}
}

Json runtime_build_identity() {
  static const Json identity=[] {
    const auto path=executable();std::ifstream stream(path,std::ios::binary|std::ios::ate);
    require(stream.good(),"srrapiv.binding.build_mismatch");auto size=stream.tellg();
    require(size>0&&size<=536870912,"srrapiv.binding.build_mismatch");
    std::string bytes(static_cast<std::size_t>(size),'\0');stream.seekg(0);stream.read(bytes.data(),size);
    require(stream.good(),"srrapiv.binding.build_mismatch");
    Json result={{"adapter_digest",engine::tagged_sha256(bytes)},{"adapter_version","0.1.0-dev"}};
#ifdef __APPLE__
    result["os"]="darwin";
#else
    result["os"]="linux";
#endif
#if defined(__aarch64__) || defined(__arm64__)
    result["architecture"]="arm64";
#else
    result["architecture"]="x86_64";
#endif
    return result;
  }();return identity;
}

Json runtime_operation(const std::string &operation,const Json &p,std::int64_t deadline_ms) {
  const auto dir=io::token(p,"runtime_dir",4096);
  if(operation=="runtime_attempts") {
    io::fields(p,{"runtime_dir","expected_generation","attempt_id","after_key","limit"});
    return attempt_records(dir,p);
  }
  if(operation=="runtime_start") {
    io::fields(p,{"runtime_dir","config_file"}); const auto config_file=io::token(p,"config_file",4096);
    auto config=io::read_json(config_file);
    require(config.contains("catalog")&&config.contains("binding"));
    auto validation=catalog_operation("binding_validate",{{"catalog",config.at("catalog")},{"binding",config.at("binding")},{"compatibility_evidence",config.value("compatibility_evidence",Json::array())}});
    require(validation.value("ok",false),"srrapiv.binding.invalid");
    check_build(config);
    io::directory(dir,true); io::directory(dir+"/attempts",true);
    // A running owner is never displaced by start. A stale socket may only be
    // replaced by the new server after acquiring the directory's exclusive lock.
    io::Fd lock(::open((dir+"/owner.lock").c_str(),O_CREAT|O_RDWR|O_NOFOLLOW|O_CLOEXEC,0600));
    require(lock.value>=0 && ::flock(lock.value,LOCK_EX|LOCK_NB)==0,"srrapiv.runtime.already_running");
    require(::flock(lock.value,LOCK_UN)==0);
    const auto generation=random_id(); const auto exe=executable();const auto config_digest=public_config_digest(config);
    auto child=::fork(); require(child>=0,"srrapiv.runtime.start_failed");
    if(child==0) {
      if(::setsid()<0)_exit(126);
      int null_fd=::open("/dev/null",O_RDWR);
      if(null_fd<0)_exit(126);
      for(int fd=0;fd<3;++fd)if(::dup2(null_fd,fd)<0)_exit(126);
      if(null_fd>2)::close(null_fd);
      ::execl(exe.c_str(),exe.c_str(),"--serve",dir.c_str(),config_file.c_str(),generation.c_str(),config_digest.c_str(),static_cast<char*>(nullptr)); _exit(127);
    }
    while(engine::unix_time_ms()<deadline_ms-100) {
      int status=0; auto result=::waitpid(child,&status,WNOHANG);
      if(result==child)throw engine::Error("srrapiv.runtime.start_failed","Native host failed before readiness",3);
      try {
        auto status_result=rpc(dir,{{"operation","runtime_status"},{"expected_generation",generation}});
        return {{"ok",true},{"state","running"},{"generation",generation},{"pid",std::to_string(child)},{"runtime",status_result}};
      } catch(const engine::Error &) {}
      std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }
    return {{"ok",false},{"state","start_outcome_unknown"},{"generation",generation},{"pid",std::to_string(child)},{"recovery","runtime_status_then_runtime_attempts_no_automatic_restart"}};
  }
  Json request=p; request.erase("runtime_dir"); request["operation"]=operation;
  require(operation=="runtime_status"||request.contains("expected_generation"));
  try { return rpc(dir,request); }
  catch(const engine::Error &error) {
    if(operation!="runtime_status"||error.code()!="srrapiv.runtime.unavailable")throw;
    Json retained,failure;
    if(std::filesystem::exists(dir+"/runtime.json"))retained=io::read_json(dir+"/runtime.json");
    if(std::filesystem::exists(dir+"/runtime-failure.json")) {
      failure=io::read_json(dir+"/runtime-failure.json");
      require(failure.value("protocol","")=="symphony.srrapiv.runtime-failure.v1","srrapiv.storage.corrupt");
    }
    const bool selected_failure=!failure.is_null()&&(retained.is_null()||
      (p.contains("expected_generation")&&failure.at("generation")==p.at("expected_generation")&&
        retained.at("generation")!=p.at("expected_generation")));
    if(selected_failure)retained={{"ok",true},{"protocol","symphony.srrapiv.runtime.v1"},
      {"generation",failure.at("generation")},{"state","startup_failed"}};
    require(!retained.is_null(),"srrapiv.runtime.unavailable");
    if(p.contains("expected_generation"))require(retained.at("generation")==p.at("expected_generation"),"srrapiv.generation.stale");
    if(!failure.is_null()&&failure.at("generation")==retained.at("generation"))retained["failure"]=failure;
    if(!selected_failure)retained["state"]="unreachable";
    retained["provider_readiness"]="unknown";
    retained["recovery"]="inspect_attempts_no_automatic_takeover";return retained;
  }
}

int serve(const std::string &dir,const std::string &config_file,const std::string &generation,const std::string &config_digest) {
  io::Fd lock(-1);bool owner_locked=false;const char *phase="acquiring_owner";
  auto record_failure=[&](const std::string &code) {
    // Retain the exclusive owner lock while publishing failure evidence. A
    // failed contender must never overwrite the active owner's diagnostics.
    if(!owner_locked)return;
    try { io::write(dir+"/runtime-failure.json",{{"protocol","symphony.srrapiv.runtime-failure.v1"},
      {"generation",generation},{"phase",phase},{"code",code},{"observed_unix_ms",std::to_string(engine::unix_time_ms())}}); }
    catch(...) {}
  };
  try {
    io::directory(dir); io::directory(dir+"/attempts");
    lock=io::Fd(::open((dir+"/owner.lock").c_str(),O_RDWR|O_NOFOLLOW|O_CLOEXEC));
    require(lock.value>=0 && ::flock(lock.value,LOCK_EX|LOCK_NB)==0,"srrapiv.runtime.already_running");
    owner_locked=true;phase="validating_binding";
    auto config=io::read_json(config_file);
    require(public_config_digest(config)==config_digest,"srrapiv.binding.config_changed");
    auto validation=catalog_operation("binding_validate",{{"catalog",config.at("catalog")},{"binding",config.at("binding")},{"compatibility_evidence",config.value("compatibility_evidence",Json::array())}});
    require(validation.value("ok",false),"srrapiv.binding.invalid");
    check_build(config);
    Events events; events.generation=generation;
    Vendor vendor(config,[&](Json event){events.emit(std::move(event));});
    // Retain exact private binding identity; never write credentials or config.
    Json state={{"ok",true},{"protocol","symphony.srrapiv.runtime.v1"},{"generation",generation},{"config_digest",config_digest},
      {"pid",std::to_string(::getpid())},{"build",runtime_build_identity()},{"binding_id",config.at("binding").at("binding_id")},
      {"artifact_id",config.at("binding").at("artifact_id")},{"state","running"},
      {"provider_readiness","not_established"},{"qualification",validation},
      {"event_retention","bounded_memory_not_durable"}};
    io::write(dir+"/runtime.json",state);
    std::size_t attempt_count=0;
    for(const auto &entry:std::filesystem::directory_iterator(dir+"/attempts")) {
      if(entry.path().filename().string().starts_with(".write-"))continue;
      require(++attempt_count<=100000,"srrapiv.attempt.capacity");
    }
    auto addr=address(dir); struct stat old{};
    phase="binding_socket";
    if(::lstat(addr.sun_path,&old)==0) { require(S_ISSOCK(old.st_mode)&&old.st_uid==::getuid(),"srrapiv.path.unsafe");require(::unlink(addr.sun_path)==0); }
    io::Fd listener(::socket(AF_UNIX,SOCK_STREAM,0)); require(listener.value>=0,"srrapiv.runtime.socket_failed");
    require(::bind(listener.value,reinterpret_cast<const sockaddr*>(&addr),sizeof(addr))==0,"srrapiv.runtime.bind_failed");
    require(::chmod(addr.sun_path,0600)==0 && ::listen(listener.value,16)==0,"srrapiv.runtime.listen_failed");
    phase="serving";
    std::signal(SIGTERM,interrupt);std::signal(SIGINT,interrupt);std::signal(SIGPIPE,SIG_IGN);
    // The control thread never enters the SDK. One supervised worker retains
    // the Vendor lifetime while status/event/recovery observations stay usable.
    // A timed-out caller does not cancel or retry an external operation.
    bool stop=false,pending_stop=false,persistence_failed=false;
    std::future<Json> worker;
    std::string worker_kind,active_attempt;
    Json last_completion;
    auto persist_state=[&] {
      try { io::write(dir+"/runtime.json",state); }
      catch(...) {
        persistence_failed=true;
        state["background_persistence_error"]="runtime_state_write_failed";
        state["recovery"]="inspect_attempts_no_automatic_retry";
      }
    };
    auto collect_worker=[&] {
      if(!worker.valid()||worker.wait_for(std::chrono::milliseconds(0))!=std::future_status::ready)return;
      try { last_completion=worker.get(); }
      catch(...) { last_completion={{"ok",false},{"state","outcome_unknown"},{"error","worker_completion_failed"}}; }
      state["worker_active"]=false;
      state.erase("active_attempt_id");
      state.erase("worker_kind");
      if(worker_kind=="stop") {
        stop=last_completion.value("ok",false);
        state["state"]=stop?"stopped":"stop_failed";
        state["stop_result"]=last_completion;
        pending_stop=false;
      } else {
        state["last_attempt_id"]=active_attempt;
        state["last_attempt_state"]=last_completion.value("state","outcome_unknown");
        if(!last_completion.value("completion_persisted",false)) {
          persistence_failed=true;
          state["background_persistence_error"]="attempt_completion_write_failed";
          state["recovery"]="inspect_retained_uncertainty_and_reconcile_provider";
        }
      }
      active_attempt.clear();worker_kind.clear();persist_state();
    };
    auto begin_stop=[&] {
      require(!worker.valid(),"srrapiv.runtime.busy");
      worker_kind="stop";state["state"]="stopping";state["worker_active"]=true;
      state["worker_kind"]="stop";persist_state();
      try {
        worker=std::async(std::launch::async,[&vendor,generation] {
          try {
            vendor.shutdown();
            return Json{{"ok",true},{"state","stopped"},{"generation",generation},{"external_orders","not_cancelled_or_reversed"}};
          } catch(...) {
            return Json{{"ok",false},{"state","stop_failed"},{"generation",generation},{"recovery","inspect_status_no_automatic_retry"}};
          }
        });
      } catch(...) {
        state["worker_active"]=false;state["state"]="stop_failed";
        state["stop_result"]={{"ok",false},{"state","stop_failed"},{"error","worker_start_failed"}};
        worker_kind.clear();state.erase("worker_kind");persist_state();throw;
      }
    };
    state["worker_active"]=false;
    while(!stop) {
      if(interrupted) {
        interrupted=0;pending_stop=true;state["shutdown_requested"]=true;
      }
      collect_worker();
      if(stop)break;
      if(pending_stop&&!worker.valid()) { pending_stop=false;begin_stop(); }
      pollfd pfd{listener.value,POLLIN,0}; auto ready=::poll(&pfd,1,250);
      if(ready<0 && errno==EINTR)continue;
      require(ready>=0,"srrapiv.transport.failed");if(!ready)continue;
      io::Fd client(::accept(listener.value,nullptr,nullptr)); if(client.value<0)continue;
      try {
        require(same_uid(client.value),"srrapiv.authority.denied");deadline_socket(client.value);
        const auto request=receive_frame(client.value);
        require(request.value("protocol","")=="symphony.srrapiv.local-control.v1"&&
          request.at("expected_adapter_digest")==runtime_build_identity().at("adapter_digest"),"srrapiv.binding.build_mismatch");
        const auto operation=io::token(request,"operation");Json result;
        collect_worker();
        if(operation!="runtime_status"||request.contains("expected_generation"))
          require(io::token(request,"expected_generation")==generation,"srrapiv.generation.stale");
        if(operation=="runtime_status") {
          io::fields(request,{"protocol","expected_adapter_digest","operation","expected_generation"}); result=state;
          result["api"]=Json{{"inspection","use_api_describe"}}; result["events"]=events.read(Json::object());
          result["events"].erase("events");
        } else if(operation=="runtime_events") {
          io::fields(request,{"protocol","expected_adapter_digest","operation","expected_generation","after_sequence","limit"}); result=events.read(request);
        } else if(operation=="runtime_invoke") {
          io::fields(request,{"protocol","expected_adapter_digest","operation","expected_generation","method","params","attempt_id","authority"});
          auto method=io::token(request,"method"); auto attempt=io::token(request,"attempt_id",128);
          require(request.contains("params")&&request.at("params").is_object());
          require(request.contains("authority")&&request.at("authority").is_object(),"srrapiv.authority.denied");
          io::fields(request.at("authority"),{"allow_provider","allow_trading"});
          authorize(config,request.at("authority"),method,request.at("params"));
          const auto digest=engine::tagged_sha256(Json{{"method",method},{"params",request.at("params")},{"binding_id",state.at("binding_id")}}.dump());
          const auto file=dir+"/attempts/"+engine::sha256_hex(attempt)+".json";
          if(std::filesystem::exists(file)) {
            result=io::read_json(file);require(result.at("request_digest")==digest,"srrapiv.attempt.conflict");result["replayed_locally"]=true;
          } else {
            require(!worker.valid()&&!pending_stop&&state.at("state")=="running","srrapiv.runtime.busy");
            require(!persistence_failed,"srrapiv.storage.failed");
            require(attempt_count<100000,"srrapiv.attempt.capacity");
            Json record={{"ok",false},{"attempt_id",attempt},{"request_digest",digest},{"generation",generation},{"binding_id",state.at("binding_id")},
              {"method",method},{"state","outcome_unknown"},{"provider_completion","not_established"},
              {"admitted_unix_ms",std::to_string(engine::unix_time_ms())}};
            io::write(file,record); ++attempt_count; // Durable uncertainty precedes any vendor effect.
            worker_kind="invoke";active_attempt=attempt;
            worker=std::async(std::launch::async,[&vendor,file,method,params=request.at("params"),record]() mutable {
              try {
                record["sdk_result"]=vendor.invoke(method,params);record["state"]="sdk_returned";
                auto status=record["sdk_result"].value("status","");
                record["ok"]=status!="error"&&status!="sdk_returned_error";
                if(status=="error"&&record["sdk_result"].value("native_called",false))record["state"]="outcome_unknown";
                record["completion_persisted"]=true;
                require(record.dump().size()<=524000&&json_value_count(record)<=retained_value_limit,"srrapiv.response.limit");
                (void)engine::parse_bounded_json(record.dump(),524288,retained_value_limit);
              } catch(...) {
                record.erase("sdk_result");record["ok"]=false;record["state"]="outcome_unknown";
                record["error"]="native_invocation_failed";
              }
              record["completion_persisted"]=true;
              try { io::write(file,record); }
              catch(...) { record["completion_persisted"]=false;record["ok"]=false;record["recovery"]="retained_attempt_remains_unknown"; }
              return record;
            });
            state["worker_active"]=true;state["worker_kind"]="invoke";state["active_attempt_id"]=attempt;persist_state();
            (void)worker.wait_for(std::chrono::milliseconds(25));collect_worker();
            if(worker.valid()) {
              result=record;result["ok"]=true;result["state"]="in_progress";
              result["recovery"]="runtime_attempts";
            } else result=last_completion;
          }
        } else if(operation=="runtime_stop") {
          io::fields(request,{"protocol","expected_adapter_digest","operation","expected_generation"});
          if(worker.valid()&&worker_kind!="stop") {
            result={{"ok",false},{"state","stop_blocked_by_active_invocation"},{"generation",generation},
              {"active_attempt_id",active_attempt},{"recovery","inspect_attempts_no_automatic_retry"}};
          } else if(stop)result=last_completion;
          else {
            if(!worker.valid())begin_stop();
            (void)worker.wait_for(std::chrono::milliseconds(25));collect_worker();
            if(worker.valid())result={{"ok",true},{"state","stopping"},{"generation",generation}};
            else result=last_completion;
          }
        } else throw engine::Error("operation.unsupported","Unsupported runtime operation",3);
        send_frame(client.value,{{"protocol","symphony.srrapiv.local-control.v1"},{"engine_version","0.1.0-dev"},{"adapter_digest",runtime_build_identity().at("adapter_digest")},{"ok",true},{"result",result}});
      } catch(const engine::Error &error) {
        try { send_frame(client.value,{{"protocol","symphony.srrapiv.local-control.v1"},{"engine_version","0.1.0-dev"},{"adapter_digest",runtime_build_identity().at("adapter_digest")},{"ok",false},{"code",error.code()}}); }catch(...) {}
      } catch(...) {
        try { send_frame(client.value,{{"protocol","symphony.srrapiv.local-control.v1"},{"engine_version","0.1.0-dev"},{"adapter_digest",runtime_build_identity().at("adapter_digest")},{"ok",false},{"code","srrapiv.runtime.failed"}}); }catch(...) {}
      }
    }
    ::unlink(addr.sun_path);return 0;
  } catch(const engine::Error &error) { record_failure(error.code());return 5; }
    catch(...) { record_failure("srrapiv.runtime.failed");return 5; }
}
}
