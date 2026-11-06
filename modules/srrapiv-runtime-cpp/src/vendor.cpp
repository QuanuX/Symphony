#include "vendor.hpp"
#include <algorithm>
#include <array>
#include <atomic>
#include <charconv>
#include <chrono>
#include <cmath>
#include <cstring>
#include <deque>
#include <iomanip>
#include <limits>
#include <map>
#include <mutex>
#include <random>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string_view>
#include <type_traits>
#include <vector>

#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
#include <RApiPlus.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>
#endif

namespace symphony::srrapiv {
namespace {
#include "vendor_metadata.inc"

Json failure(const std::string& code, const std::string& message) {
    return Json{{"status","error"},{"code",code},{"message",message},{"native_called",false}};
}
#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
void require_object(const Json& j) {
    if (!j.is_object()) throw std::invalid_argument("expected a JSON object");
}
const Json& required(const Json& j, const std::string& name) {
    auto it=j.find(name); if(it==j.end()) throw std::invalid_argument("missing native argument: "+name); return *it;
}
void validate_keys(const Json& j, std::initializer_list<const char*> keys) {
    require_object(j);
    for(auto it=j.begin();it!=j.end();++it) {
        bool found=false; for(auto key:keys) if(it.key()==key) {found=true;break;}
        if(!found) throw std::invalid_argument("unknown native field");
    }
}
Json select_keys(const Json& j, std::initializer_list<const char*> keys) {
    Json selected=Json::object(); for(auto key:keys) if(j.contains(key)) selected[key]=j.at(key); return selected;
}
std::string hex_bytes(const char* data, std::size_t size) {
    static constexpr char digits[]="0123456789abcdef"; std::string out; out.reserve(size*2);
    for(std::size_t i=0;i<size;++i) {auto c=static_cast<unsigned char>(data[i]);out.push_back(digits[c>>4]);out.push_back(digits[c&15]);} return out;
}
std::string unhex(const std::string& text) {
    if(text.size()%2) throw std::invalid_argument("odd hexadecimal byte length");
    auto digit=[](char c)->unsigned {if(c>='0'&&c<='9')return c-'0';if(c>='a'&&c<='f')return c-'a'+10;if(c>='A'&&c<='F')return c-'A'+10;throw std::invalid_argument("invalid hexadecimal byte");};
    std::string out;out.reserve(text.size()/2);for(std::size_t i=0;i<text.size();i+=2)out.push_back(static_cast<char>((digit(text[i])<<4)|digit(text[i+1])));return out;
}
std::string generation_id() {
    std::random_device rd; std::ostringstream out;out<<std::hex<<rd()<<rd()<<rd()<<rd();return out.str();
}

using namespace RApi;
#include "vendor_initializers.inc"
struct State;
struct Encoding {
    State& state; std::size_t elements=0, bytes=0;
    explicit Encoding(State& s):state(s){}
    void consume(std::size_t n,std::size_t b);
};
struct Arena {
    State& state; std::vector<std::shared_ptr<void>> owned; std::size_t elements=0,bytes=0;
    explicit Arena(State& s):state(s){}
    ~Arena()=default;
    void consume(std::size_t n,std::size_t b);
    template<class T> T* make() {
        consume(1,sizeof(T)); auto* p=new T{};
        if constexpr(requires { initialize_public(*p); })initialize_public(*p);
        owned.emplace_back(p,[](void* q){try {delete static_cast<T*>(q);}catch(...) {}});return p;
    }
    template<class T> T* array(std::size_t n) {
        if(n>std::numeric_limits<std::size_t>::max()/sizeof(T))throw std::invalid_argument("array allocation overflow");
        consume(n,n*sizeof(T));if(n==0)return nullptr;auto* p=new T[n]{};
        if constexpr(requires { initialize_public(*p); })for(std::size_t i=0;i<n;++i)initialize_public(p[i]);
        owned.emplace_back(p,[](void* q){try{delete[] static_cast<T*>(q);}catch(...) {}});return p;
    }
    tsNCharcb string(const Json& j);
    tsNCharcb secret_string(const Json& j);
    char** environment(const Json& j);
};
struct State {
    Json config; Vendor::Emit emit;
    REngine* engine=nullptr;
    std::unique_ptr<Arena> engine_inputs;
    std::unique_ptr<RCallbacks> callbacks;
    std::unique_ptr<AdmCallbacks> admin;
    std::unique_ptr<OmneEngineSpace::OmneTimerCB> timer;
    std::string generation=generation_id();
    std::atomic<std::uint64_t> event_sequence{0},encoding_failures{0},delivery_failures{0};
    std::recursive_mutex invocation_mutex;
    std::mutex handles_mutex;
    struct Handle {void* value;std::string kind;};
    struct Context {std::string label;};
    std::map<std::string,Handle> handles;
    std::map<std::string,std::unique_ptr<Context>> contexts;
    std::uint64_t next_handle=0;
    std::size_t max_array=65536,max_string=1024*1024,max_payload=16*1024*1024,max_contexts=4096;
    int lock_depth=0; bool native_call_started=false;
    explicit State(Json c,Vendor::Emit e);
    ~State();
    Json handle(void* ptr,const std::string& kind);
    void* context(const Json& j);
    void* callback_handle(const std::string& kind,const Json& j);
    void deliver(const std::string& event,Json payload) noexcept;
    void encoding_failure(const std::string& event,const std::string& error) noexcept;
    void stop();
};
void Encoding::consume(std::size_t n,std::size_t b) {
    if(n>state.max_array-elements||b>state.max_payload-bytes)throw std::length_error("callback serialization budget exceeded");elements+=n;bytes+=b;
}
void Arena::consume(std::size_t n,std::size_t b) {
    if(n>state.max_array-elements||b>state.max_payload-bytes)throw std::length_error("native request budget exceeded");elements+=n;bytes+=b;
}
bool sensitive_name(const tsNCharcb& v) {
    if(v.iDataLen<=0||!v.pData)return false;
    std::string name(v.pData,static_cast<std::size_t>(v.iDataLen));
    std::transform(name.begin(),name.end(),name.begin(),[](unsigned char c){return static_cast<char>(std::tolower(c));});
    return name.find("password")!=std::string::npos||name.find("passwd")!=std::string::npos||name.find("secret")!=std::string::npos||name.find("token")!=std::string::npos||name.find("key")!=std::string::npos||name.find("authorization")!=std::string::npos||name.find("credential")!=std::string::npos;
}
Json encode(const tsNCharcb& value,Encoding& e) {
    if(value.iDataLen<0) return Json{{"length",value.iDataLen},{"bytes_hex",nullptr},{"pointer_present",value.pData!=nullptr},{"state","vendor-sentinel"}};
    auto n=static_cast<std::size_t>(value.iDataLen);
    if(n>e.state.max_string)throw std::length_error("native string exceeds configured bound");
    if(n&&!value.pData)throw std::invalid_argument("native string has length without data");
    e.consume(1,n*3);
    std::string s(value.pData?value.pData:"",n);
    Json result{{"bytes_hex",hex_bytes(s.data(),s.size())},{"length",value.iDataLen},{"pointer_present",value.pData!=nullptr}};
    try {Json text=s;(void)text.dump();result["text"]=std::move(text);}catch(const Json::type_error&){}
    return result;
}
void write_secret_output(State& state,const std::string& reference,const tsNCharcb& secret);
Json encode_environment_value(const tsNCharcb& name,const tsNCharcb& value,Encoding& e) {
    if(name.iDataLen<0||!name.pData||static_cast<std::size_t>(name.iDataLen)>e.state.max_string)throw std::invalid_argument("invalid environment variable name");
    std::string key(name.pData,static_cast<std::size_t>(name.iDataLen));
    const auto destinations=e.state.config.find("protected_environment_outputs");
    if(destinations!=e.state.config.end()&&destinations->contains(key)) {
        if(value.iDataLen<0||static_cast<std::size_t>(value.iDataLen)>e.state.max_string)throw std::length_error("environment value exceeds protected output bound");
        auto reference=destinations->at(key).get<std::string>();write_secret_output(e.state,reference,value);return Json{{"redacted",true},{"output_ref",reference},{"written",true}};
    }
    const auto allowed=e.state.config.find("safe_environment_keys");
    if(!sensitive_name(name)&&allowed!=e.state.config.end()&&allowed->is_array()&&std::find(allowed->begin(),allowed->end(),key)!=allowed->end())return encode(value,e);
    return Json{{"redacted",true}};
}
Json encode(bool value,Encoding& e){e.consume(1,1);return value;}
template<class T, std::enable_if_t<std::is_integral_v<T>&&!std::is_same_v<T,bool>,int> =0>
Json encode(T value,Encoding& e) {e.consume(1,sizeof(T));if constexpr(sizeof(T)>=8)return std::to_string(value);else return value;}
template<class T,std::enable_if_t<std::is_enum_v<T>,int> =0>
Json encode(T value,Encoding& e){return encode(static_cast<std::underlying_type_t<T>>(value),e);}
Json encode(double value,Encoding& e) {
    e.consume(1,8);std::uint64_t bits=0;std::memcpy(&bits,&value,8);std::ostringstream out;out<<std::hex<<std::setfill('0')<<std::setw(16)<<bits;
    return Json{{"f64_bits",out.str()}};
}
void decode(const Json& j, bool& value,Arena&){if(!j.is_boolean())throw std::invalid_argument("expected boolean");value=j.get<bool>();}
template<class T,std::enable_if_t<std::is_integral_v<T>&&!std::is_same_v<T,bool>,int> =0>
void decode(const Json& j,T& value,Arena&) {
    std::string text;
    if(j.is_string())text=j.get<std::string>();
    else if(j.is_number_integer()||j.is_number_unsigned()) {
        if(j.is_number_unsigned()&&j.get<std::uint64_t>()>9007199254740991ULL)throw std::invalid_argument("wide integers require decimal strings");
        if(j.is_number_integer()&&!j.is_number_unsigned()&&(j.get<std::int64_t>()>9007199254740991LL||j.get<std::int64_t>()< -9007199254740991LL))throw std::invalid_argument("wide integers require decimal strings");
        text=j.dump();
    } else throw std::invalid_argument("expected integer or decimal string");
    if(text.empty()||text.front()=='+'||(text.size()>1&&text.front()=='0')||(text.size()>2&&text[0]=='-'&&text[1]=='0'))throw std::invalid_argument("noncanonical integer");
    T parsed{};auto result=std::from_chars(text.data(),text.data()+text.size(),parsed);
    if(result.ec!=std::errc{}||result.ptr!=text.data()+text.size())throw std::invalid_argument("integer out of range or malformed");value=parsed;
}
template<class T,std::enable_if_t<std::is_enum_v<T>,int> =0>
void decode(const Json& j,T& value,Arena& a){std::underlying_type_t<T> parsed{};decode(j,parsed,a);value=static_cast<T>(parsed);}
void decode(const Json& j,double& value,Arena&) {
    if(j.is_object()&&j.contains("f64_bits")) {
        validate_keys(j,{"f64_bits"});auto text=j.at("f64_bits").get<std::string>();if(text.size()!=16)throw std::invalid_argument("f64_bits requires 16 hexadecimal digits");
        std::uint64_t bits{};auto r=std::from_chars(text.data(),text.data()+text.size(),bits,16);if(r.ec!=std::errc{}||r.ptr!=text.data()+text.size())throw std::invalid_argument("invalid f64_bits");std::memcpy(&value,&bits,8);
    }else if(j.is_number())value=j.get<double>();else throw std::invalid_argument("expected f64_bits or numeric value");
}
tsNCharcb Arena::string(const Json& j) {
    if(j.is_null())return {nullptr,0};
    std::string value;
    if(j.is_string())value=j.get<std::string>();
    else if(j.is_object()&&j.contains("bytes_hex")) {
        validate_keys(j,{"bytes_hex","length","text","pointer_present"});value=unhex(j.at("bytes_hex").get<std::string>());
        if(j.contains("length")&&j.at("length")!=value.size())throw std::invalid_argument("string length does not match bytes");
        if(j.contains("text")&&j.at("text")!=value)throw std::invalid_argument("string text does not match bytes");
        if(j.contains("pointer_present")){if(!j.at("pointer_present").is_boolean())throw std::invalid_argument("pointer_present must be boolean");if(!j.at("pointer_present").get<bool>()){if(!value.empty())throw std::invalid_argument("absent string pointer cannot contain bytes");return {nullptr,0};}}
    }else throw std::invalid_argument("expected text or bytes_hex string object");
    if(value.size()>state.max_string||value.size()>static_cast<std::size_t>(std::numeric_limits<int>::max()))throw std::length_error("string exceeds configured bound");
    consume(1,value.size()+1);auto keep=std::make_shared<std::string>(std::move(value));tsNCharcb out{keep->data(),static_cast<int>(keep->size())};owned.push_back(keep);return out;
}
tsNCharcb Arena::secret_string(const Json& j) {
    validate_keys(j,{"secret_ref"});auto name=required(j,"secret_ref").get<std::string>();
    if(!state.config.contains("secrets")||!state.config.at("secrets").contains(name))throw std::invalid_argument("unknown protected secret reference");
    return string(state.config.at("secrets").at(name));
}
void decode(const Json& j,tsNCharcb& value,Arena& a){value=a.string(j);}
void decode(const Json& j,void*& value,Arena& a){value=a.state.context(j);}
Json encode_pointer(void* value,Encoding& e){e.consume(1,0);return e.state.handle(value,"context");}
Json encode_environment(char* const* values,Encoding& e) {
    if(!values)return nullptr;Json result=Json::array();
    for(std::size_t i=0;i<e.state.max_array;++i) {
        if(!values[i])return result;auto len=strnlen(values[i],e.state.max_string+1);if(len>e.state.max_string)throw std::length_error("environment item exceeds limit");
        auto eq=std::string_view(values[i],len).find('=');tsNCharcb key{values[i],static_cast<int>(eq==std::string_view::npos?len:eq)};
        tsNCharcb value{values[i]+(eq==std::string_view::npos?len:eq+1),static_cast<int>(eq==std::string_view::npos?0:len-eq-1)};
        result.push_back(Json{{"name",encode(key,e)},{"value",encode_environment_value(key,value,e)}});
    }throw std::length_error("unterminated native environment array");
}
char** Arena::environment(const Json& j) {
    if(!j.is_array())throw std::invalid_argument("envp must be an array");auto** out=array<char*>(j.size()+1);
    for(std::size_t i=0;i<j.size();++i) {auto text=string(j.at(i));if(std::memchr(text.pData,'\0',static_cast<std::size_t>(text.iDataLen)))throw std::invalid_argument("environment value contains null byte");out[i]=text.pData;}out[j.size()]=nullptr;return out;
}
#include "vendor_declarations.inc"
template<class T> Json encode_pointer(const T* value,Encoding& e){if(!value)return nullptr;return encode(*value,e);}
template<class T> Json encode_array(const T* values,int count,Encoding& e) {
    if(count<0)throw std::invalid_argument("negative native array length");if(!values)return nullptr;
    e.consume(static_cast<std::size_t>(count),0);Json out=Json::array();for(int i=0;i<count;++i)out.push_back(encode(values[i],e));return out;
}
template<class T> T* decode_pointer(const Json& j,Arena& a){if(j.is_null())return nullptr;auto* out=a.make<T>();decode(j,*out,a);return out;}
template<class T> T* decode_array(const Json& j,Arena& a) {
    if(j.is_null())return nullptr;if(!j.is_array())throw std::invalid_argument("expected native array");auto* out=a.array<T>(j.size());for(std::size_t i=0;i<j.size();++i)decode(j.at(i),out[i],a);return out;
}
void require_array_count(const Json& array,int count){if(count<0||!array.is_array()||array.size()!=static_cast<std::size_t>(count))throw std::invalid_argument("native array/count mismatch");}
int validate_array_group(const Json& object,const char* count_name,std::initializer_list<const char*> fields,int fallback,Arena& arena) {
    int count=fallback;bool known=false;
    if(object.contains(count_name)){decode(object.at(count_name),count,arena);if(count<0)throw std::invalid_argument("negative native array length");known=true;}
    for(const auto* name:fields) {
        if(!object.contains(name)||object.at(name).is_null())continue;
        const auto& array=object.at(name);if(!array.is_array())throw std::invalid_argument("expected native array");
        if(array.size()>static_cast<std::size_t>(std::numeric_limits<int>::max()))throw std::length_error("native array too long");
        auto size=static_cast<int>(array.size());if(known&&size!=count)throw std::invalid_argument("native shared array/count mismatch");count=size;known=true;
    }
    return count;
}
#include "vendor_codecs.inc"
#include "vendor_callbacks.inc"
class TimerCallback final:public OmneEngineSpace::OmneTimerCB {
    State& state;
public:
    explicit TimerCallback(State& s):state(s){}
    int call(tsNCharcb* name,void* context,int* code) override {
        try{Encoding e(state);state.deliver("OmneTimerCB.call#1",Json{{"pTimerName",encode_pointer(name,e)},{"pContext",encode_pointer(context,e)}});}catch(const std::exception&){state.encoding_failure("OmneTimerCB.call#1","callback payload failed bounded encoding");}catch(...){state.encoding_failure("OmneTimerCB.call#1","timer encoding failed");}if(code)*code=API_OK;return OK;
    }
};
State::State(Json c,Vendor::Emit e):config(std::move(c)),emit(std::move(e)) {
    require_object(config);
    auto limit=[&](const char* name,std::size_t fallback,std::size_t ceiling){auto v=config.value(name,fallback);if(v==0||v>ceiling)throw std::invalid_argument(std::string("invalid limit: ")+name);return v;};
    max_array=limit("max_array_elements",65536,1048576);max_string=limit("max_string_bytes",1024*1024,16*1024*1024);max_payload=limit("max_payload_bytes",16*1024*1024,64*1024*1024);max_contexts=limit("max_contexts",4096,65536);
    callbacks=std::make_unique<Callbacks>(*this);admin=std::make_unique<AdminCallbacks>(*this);timer=std::make_unique<TimerCallback>(*this);
}
State::~State(){try{stop();}catch(...) {}}
Json State::handle(void* ptr,const std::string& kind) {
    if(!ptr)return nullptr;std::lock_guard<std::mutex> lock(handles_mutex);
    for(const auto& pair:handles)if(pair.second.value==ptr&&pair.second.kind==kind)return Json{{"handle",pair.first},{"kind",kind},{"generation",generation}};
    if(handles.size()>=max_contexts)throw std::length_error("native handle bound exceeded");auto id=kind+":"+generation+":"+std::to_string(++next_handle);handles[id]={ptr,kind};return Json{{"handle",id},{"kind",kind},{"generation",generation}};
}
void* State::context(const Json& j) {
    if(j.is_null())return nullptr;
    if(j.is_object()&&j.contains("handle")) {
        validate_keys(j,{"handle","kind","generation"});std::lock_guard<std::mutex> lock(handles_mutex);auto it=handles.find(j.at("handle").get<std::string>());if(it==handles.end()||it->second.kind!="context")throw std::invalid_argument("unknown or stale context handle");if(j.contains("generation")&&j.at("generation")!=generation)throw std::invalid_argument("stale context generation");return it->second.value;
    }
    std::string label;if(j.is_string())label=j.get<std::string>();else {validate_keys(j,{"context_id"});label=required(j,"context_id").get<std::string>();}
    if(label.empty()||label.size()>1024)throw std::invalid_argument("invalid context identifier");std::lock_guard<std::mutex> lock(handles_mutex);auto it=contexts.find(label);if(it!=contexts.end())return it->second.get();if(contexts.size()>=max_contexts)throw std::length_error("context bound exceeded");auto p=std::make_unique<Context>();p->label=label;auto* out=p.get();contexts[label]=std::move(p);return out;
}
void* State::callback_handle(const std::string& kind,const Json& j) {
    void* expected=kind=="RCallbacks"?static_cast<void*>(callbacks.get()):static_cast<void*>(admin.get());
    if(j.is_null())return expected;
    validate_keys(j,{"handle","kind","generation"});std::lock_guard<std::mutex> lock(handles_mutex);auto it=handles.find(required(j,"handle").get<std::string>());if(it==handles.end()||it->second.kind!=kind||it->second.value!=expected)throw std::invalid_argument("unknown callback object handle");return expected;
}
void State::deliver(const std::string& event,Json payload) noexcept {
    try {Json envelope{{"schema","srrapiv.vendor-event.v1"},{"sdk_version","13.7.0.0"},{"event",event},{"generation",generation},{"sequence",std::to_string(++event_sequence)},{"payload",std::move(payload)}};if(envelope.dump().size()>max_payload){encoding_failure(event,"serialized callback exceeds payload limit");return;}if(emit)emit(std::move(envelope));}catch(...){++delivery_failures;}
}
void State::encoding_failure(const std::string& event,const std::string& error) noexcept {
    ++encoding_failures;try{if(emit)emit(Json{{"schema","srrapiv.vendor-event.v1"},{"event","runtime.callback_gap"},{"source_event",event},{"generation",generation},{"sequence",std::to_string(++event_sequence)},{"reason",error},{"state_uncertain",true}});}catch(...){++delivery_failures;}
}
void State::stop() {
    std::lock_guard<std::recursive_mutex> guard(invocation_mutex);
    if(lock_depth)throw std::logic_error("cannot destroy engine inside a native lock scope");
    if(engine){auto* current=engine;engine=nullptr;delete current;}
    engine_inputs.reset();
    std::lock_guard<std::mutex> lock(handles_mutex);handles.clear();contexts.clear();generation=generation_id();
}
Json native_result(const std::string& id,int rc,int code,Json outputs) {
    return Json{{"status",rc==OK?"sdk_returned_ok":"sdk_returned_error"},{"operation",id},{"native_called",true},{"native_return",rc},{"native_code",code},{"outputs",std::move(outputs)},{"business_outcome","not_implied_by_native_return"}};
}
void write_secret_output(State& state,const std::string& reference,const tsNCharcb& secret) {
    if(secret.iDataLen<0||(!secret.pData&&secret.iDataLen))throw std::invalid_argument("invalid native secret result");
    if(!state.config.contains("secret_outputs")||!state.config.at("secret_outputs").contains(reference))throw std::invalid_argument("unknown protected output reference");
    auto path=state.config.at("secret_outputs").at(reference).get<std::string>();if(path.empty()||path.front()!='/')throw std::invalid_argument("protected output must be an absolute configured destination");
    struct Close {int fd;~Close(){if(fd>=0)close(fd);}} directory{open("/",O_RDONLY|O_DIRECTORY|O_CLOEXEC)};
    if(directory.fd<0)throw std::runtime_error("protected output root cannot be opened");
    std::size_t start=1;int fd=-1;
    while(start<path.size()) {
        auto end=path.find('/',start);auto part=path.substr(start,end==std::string::npos?std::string::npos:end-start);
        if(part.empty()||part=="."||part=="..")throw std::invalid_argument("protected output path must have canonical components");
        if(end==std::string::npos){fd=openat(directory.fd,part.c_str(),O_WRONLY|O_NOFOLLOW|O_CLOEXEC);break;}
        int next=openat(directory.fd,part.c_str(),O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);
        if(next<0)throw std::runtime_error("protected output parent cannot be opened safely");
        close(directory.fd);directory.fd=next;start=end+1;
    }
    if(fd<0)throw std::runtime_error("protected output cannot be opened");
    Close closer{fd};struct stat st{};
    if(fstat(fd,&st)||!S_ISREG(st.st_mode)||st.st_uid!=geteuid()||(st.st_mode&0777)!=0600||st.st_nlink!=1)throw std::runtime_error("protected output must be an owned 0600 regular file with one link");
    if(ftruncate(fd,0))throw std::runtime_error("protected output truncation failed");
    std::size_t offset=0,total=static_cast<std::size_t>(secret.iDataLen);
    while(offset<total){auto n=write(fd,secret.pData+offset,total-offset);if(n<0&&errno==EINTR)continue;if(n<=0)throw std::runtime_error("protected output write failed");offset+=static_cast<std::size_t>(n);}if(fsync(fd))throw std::runtime_error("protected output synchronization failed");
}
Json invoke_native(State& state,const std::string& id,const Json& p);
Json scoped_lock(State& state,const std::string& id,const Json& p) {
    validate_keys(p,{"operations"});Json operations=p.value("operations",Json::array());if(!operations.is_array()||operations.size()>1024)throw std::invalid_argument("bounded native operations array required");
    int code=API_OK;state.native_call_started=true;int rc=id=="REngine.tryLock#1"?state.engine->tryLock(&code):state.engine->lock(&code);if(rc!=OK)return native_result(id,rc,code,Json::object());
    int initial=state.lock_depth;++state.lock_depth;Json results=Json::array();
    try {for(const auto& operation:operations){validate_keys(operation,{"operation","params"});auto nested=required(operation,"operation").get<std::string>();if(nested=="runtime.initialize"||nested=="runtime.shutdown"||nested.rfind("REngine.login",0)==0||nested.rfind("REngine.logout",0)==0)throw std::invalid_argument("lifecycle operation is not allowed within a native lock scope");results.push_back(invoke_native(state,nested,operation.value("params",Json::object())));}}catch(...) {while(state.lock_depth>initial){int cleanup=API_OK;state.engine->unlock(&cleanup);--state.lock_depth;}throw;}
    int unlock_code=API_OK,unlock_rc=OK;while(state.lock_depth>initial){unlock_rc=state.engine->unlock(&unlock_code);--state.lock_depth;}
    auto result=native_result(id,rc,code,Json{{"operations",results},{"unlock_return",unlock_rc},{"unlock_code",unlock_code},{"scope","native-call-batch"}});return result;
}
Json invoke_native(State& state,const std::string& id,const Json& p) {
    require_object(p);
    if(state.config.contains("allowed_methods")) {
        const auto& allowed=state.config.at("allowed_methods");
        if(!allowed.is_array())throw std::invalid_argument("allowed_methods must be an array");
        if(std::find(allowed.begin(),allowed.end(),id)==allowed.end())return failure("METHOD_NOT_ALLOWED","operation is not permitted by this runtime's exact allowlist");
    }
    const bool trading=id.rfind("REngine.send",0)==0||id.rfind("REngine.modify",0)==0||id.rfind("REngine.cancel",0)==0||id=="REngine.exitPosition#1"||id=="REngine.linkOrders#1"||id=="REngine.createUserDefinedSpread#1"||id=="REngine.submitQuoteList#1";
    if(trading&&!state.config.value("allow_trading",false))return failure("TRADING_DISABLED","private runtime configuration does not authorize trading operations");
    if(state.lock_depth>0&&(id=="REngine.lock#1"||id=="REngine.tryLock#1"))return failure("NESTED_LOCK_SCOPE","nested native lock scopes are not admitted");
    if(id=="runtime.initialize") {
        validate_keys(p,{});if(state.engine)return failure("ALREADY_INITIALIZED","runtime already owns an SDK engine");
        if(!state.config.value("allow_provider_calls",false))return failure("PROVIDER_INITIALIZATION_DISABLED","explicit private configuration must enable provider initialization");
        auto arena=std::make_unique<Arena>(state);auto* params=arena->make<REngineParams>();decode(state.config.value("engine",Json::object()),*params,*arena);if(!params->envp)throw std::invalid_argument("explicit engine.envp is required; ambient environment is not selected");
        state.native_call_started=true;state.engine=new REngine(params);state.engine_inputs=std::move(arena);return Json{{"status","initialized"},{"generation",state.generation},{"login_status","not_requested"},{"native_called",true}};
    }
    if(id=="runtime.shutdown"){validate_keys(p,{});state.stop();return Json{{"status","stopped"},{"native_called",true}};}
    if(id=="runtime.status"){validate_keys(p,{});return Json{{"status","ok"},{"engine_initialized",state.engine!=nullptr},{"generation",state.generation},{"events",std::to_string(state.event_sequence.load())},{"callback_encoding_failures",std::to_string(state.encoding_failures.load())},{"event_delivery_failures",std::to_string(state.delivery_failures.load())}};}
    if(id=="object.construct") {
        validate_keys(p,{"type","iTicksIn","llQtyIn","llQtyReleasedIn"});if(required(p,"type")!="BracketTier")throw std::invalid_argument("use object.initialize or object.convert for this constructor");Arena arena(state);Encoding encoding(state);int ticks{};long long qty{},released{};decode(required(p,"iTicksIn"),ticks,arena);decode(required(p,"llQtyIn"),qty,arena);decode(required(p,"llQtyReleasedIn"),released,arena);BracketTier value(ticks,qty,released);return Json{{"status","ok"},{"type","BracketTier"},{"value",encode(value,encoding)}};
    }
    if(id=="object.convert") {
        validate_keys(p,{"type","from_type","value"});auto type=required(p,"type").get<std::string>();auto from=required(p,"from_type").get<std::string>();Arena arena(state);Encoding encoding(state);
        if(type=="OrderParams" && from=="LimitOrderParams") {auto* input=decode_pointer<LimitOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); OrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="OrderParams" && from=="MarketOrderParams") {auto* input=decode_pointer<MarketOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); OrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="OrderParams" && from=="StopLimitOrderParams") {auto* input=decode_pointer<StopLimitOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); OrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="OrderParams" && from=="StopMarketOrderParams") {auto* input=decode_pointer<StopMarketOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); OrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="ModifyOrderParams" && from=="ModifyLimitOrderParams") {auto* input=decode_pointer<ModifyLimitOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); ModifyOrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="ModifyOrderParams" && from=="ModifyStopLimitOrderParams") {auto* input=decode_pointer<ModifyStopLimitOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); ModifyOrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        if(type=="ModifyOrderParams" && from=="ModifyStopMarketOrderParams") {auto* input=decode_pointer<ModifyStopMarketOrderParams>(required(p,"value"),arena);if(!input)throw std::invalid_argument("null conversion source"); ModifyOrderParams result(input);return Json{{"status","ok"},{"type",type},{"from_type",from},{"value",encode(result,encoding)}};}
        throw std::invalid_argument("no matching native conversion constructor");
    }
    if(id.rfind("object.",0)==0) {
        validate_keys(p,{"type","value"});std::string action=id.substr(7);if(action!="initialize"&&action!="inspect"&&action!="clear"&&action!="dump")throw std::invalid_argument("unknown object operation");auto type=required(p,"type").get<std::string>();Arena arena(state);
#include "vendor_objects.inc"
        throw std::invalid_argument("unknown native data type");
    }
    if(id=="runtime.error.describe") {validate_keys(p,{"code"});Arena arena(state);int code=0;decode(required(p,"code"),code,arena);OmneException error(code);return Json{{"status","ok"},{"code",error.getErrorCode()},{"description",error.getErrorString()?error.getErrorString():""}};}
    if(!state.engine&&id!="REngine.getVersion#1")return failure("ENGINE_NOT_INITIALIZED","explicit runtime.initialize is required before native engine operations");
    if(id=="REngine.lock#1"||id=="REngine.tryLock#1")return scoped_lock(state,id,p);
    if(id=="REngine.unlock#1") {validate_keys(p,{});if(state.lock_depth<=0)return failure("NO_NATIVE_LOCK_SCOPE","unlock is only valid inside the current bounded native scope");int code=API_OK;state.native_call_started=true;int rc=state.engine->unlock(&code);if(rc==OK)--state.lock_depth;return native_result(id,rc,code,Json::object());}
    if(id=="REngine.getPassword#1") {
        validate_keys(p,{"output_ref"});auto reference=required(p,"output_ref").get<std::string>();
        // Validate destination before calling the getter; the native secret never enters JSON.
        if(!state.config.contains("secret_outputs")||!state.config.at("secret_outputs").contains(reference))throw std::invalid_argument("unknown protected output reference");
        tsNCharcb password{};int code=API_OK;state.native_call_started=true;int rc=state.engine->getPassword(&password,&code);if(rc==OK)write_secret_output(state,reference,password);return native_result(id,rc,code,Json{{"output_ref",reference},{"written",rc==OK}});
    }
    if(id=="REngine.addTimer#1") {validate_keys(p,{"pTimerName","lFreq","pContext","pTimerCB"});Arena arena(state);auto name=arena.string(required(p,"pTimerName"));long frequency=0;decode(required(p,"lFreq"),frequency,arena);if(p.contains("pTimerCB")&&!p.at("pTimerCB").is_null())throw std::invalid_argument("timer callback is the runtime-owned observable timer adapter");int code=API_OK;state.native_call_started=true;int rc=state.engine->addTimer(&name,state.timer.get(),frequency,state.context(p.value("pContext",Json())),&code);Encoding e(state);return native_result(id,rc,code,Json{{"pTimerName",encode(name,e)},{"callback","OmneTimerCB.call#1"}});}
#include "vendor_dispatch.inc"
    return failure("UNKNOWN_OPERATION","operation is not in the selected SDK surface");
}
#endif
}

struct Vendor::Impl {
#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
    State state;
    Impl(Json config,Emit emit):state(std::move(config),std::move(emit)){}
#else
    Impl(Json,Emit){}
#endif
};
Vendor::Vendor(Json config,Emit emit):impl_(std::make_unique<Impl>(std::move(config),std::move(emit))){}
Vendor::~Vendor(){try{shutdown();}catch(...) {}}
Json Vendor::describe() const {
    Json result=Json::parse(vendor_metadata);
    result["supplemental_operations"]={"runtime.initialize","runtime.shutdown","runtime.status","runtime.error.describe","object.initialize","object.inspect","object.clear","object.dump","object.convert","object.construct"};
#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
    result["sdk_available"]=true;tsNCharcb version{};int code=API_OK;int rc=REngine::getVersion(&version,&code);result["sdk_native_version_return"]=rc;result["sdk_native_version_code"]=code;if(rc==OK&&version.pData&&version.iDataLen>=0)result["sdk_native_version"]=std::string(version.pData,static_cast<std::size_t>(version.iDataLen));
#else
    result["sdk_available"]=false;result["availability_reason"]="built without the user-supplied licensed SDK";
#endif
#ifdef SYMPHONY_SRRAPIV_LIBRARY_SHA256
    result["selected_library_sha256"]=SYMPHONY_SRRAPIV_LIBRARY_SHA256;
#endif
#ifdef SYMPHONY_SRRAPIV_PLATFORM
    result["selected_platform"]=SYMPHONY_SRRAPIV_PLATFORM;
#endif
    result["qualification"]="source and local checks only; provider behavior requires environment qualification";return result;
}
Json Vendor::invoke(const std::string& id,const Json& p) {
#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
    std::lock_guard<std::recursive_mutex> guard(impl_->state.invocation_mutex);
    impl_->state.native_call_started=false;
    auto error_result=[&](const std::string& code,const std::string& message){auto result=failure(code,message);result["native_called"]=impl_->state.native_call_started;if(impl_->state.native_call_started)result["business_outcome"]="unknown_after_native_failure";return result;};
    try{return invoke_native(impl_->state,id,p);}catch(const OmneException& error){return error_result("SDK_EXCEPTION",std::to_string(const_cast<OmneException&>(error).getErrorCode()));}catch(const Json::exception&){return error_result("INVALID_JSON_CONTRACT","JSON value has invalid type, structure or encoding");}catch(const std::invalid_argument& error){return error_result("INVALID_NATIVE_ARGUMENT",error.what());}catch(const std::length_error& error){return error_result("NATIVE_BOUNDS_EXCEEDED",error.what());}catch(const std::exception&){return error_result("INVALID_OR_FAILED_OPERATION","native operation failed; inspect its safe status and recovery state");}catch(...){return error_result("NATIVE_EXCEPTION","unclassified native exception");}
#else
    (void)id;(void)p;return failure("SDK_UNAVAILABLE","this build has no licensed R API+ SDK linked");
#endif
}
void Vendor::shutdown(){
#if defined(SYMPHONY_SRRAPIV_WITH_RAPI)
    impl_->state.stop();
#endif
}
}
