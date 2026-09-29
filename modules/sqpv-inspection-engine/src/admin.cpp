#include "request.hpp"
#include "store_reader.hpp"
namespace symphony::sqpv::administration {
using namespace symphony::sqv_admin;
Json administer(std::string_view, const Json &p, std::int64_t end) {
  fields(p, {"protocol", "store"});
  return inspection::summary(inspection::inspect(p.at("store"), end));
}
} // namespace symphony::sqpv::administration
