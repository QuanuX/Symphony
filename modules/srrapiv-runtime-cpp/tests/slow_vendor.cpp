#include "vendor.hpp"
#include <chrono>
#include <thread>

// Test-only link replacement. No licensed SDK, network, credential, or order
// path exists in this executable; delays exercise supervisor responsiveness.
namespace symphony::srrapiv {
struct Vendor::Impl { Emit emit;explicit Impl(Emit value):emit(std::move(value)){} };
Vendor::Vendor(Json,Emit emit):impl_(std::make_unique<Impl>(std::move(emit))) {}
Vendor::~Vendor()=default;
Json Vendor::describe() const {
  return {{"sdk_version","13.7.0.0"},
    {"header_sha256","b996e76fbfe20c7f4d7ba2d645c45cb13d6b19df37d7522125e94e61e74df5b4"},
    {"sdk_available",false},{"test_fixture",true}};
}
Json Vendor::invoke(const std::string &operation,const Json &) {
  if(operation!="REngine.getVersion#1")return {{"status","error"},{"code","FIXTURE_OPERATION_UNAVAILABLE"},{"native_called",false}};
  std::this_thread::sleep_for(std::chrono::milliseconds(1000));
  Json values=Json::array();for(unsigned i=0;i<33000;++i)values.push_back(0);
  impl_->emit({{"test_fixture",true},{"values",values}});
  impl_->emit({{"test_fixture",true},{"values",values}});
  auto excessive=values;for(unsigned i=0;i<33000;++i)excessive.push_back(0);
  impl_->emit({{"test_fixture",true},{"values",std::move(excessive)}});
  return {{"status","sdk_returned_ok"},{"native_called",false},{"test_fixture",true},{"outputs",std::move(values)}};
}
void Vendor::shutdown() { std::this_thread::sleep_for(std::chrono::milliseconds(500)); }
}
