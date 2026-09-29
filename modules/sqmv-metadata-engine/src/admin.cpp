#include "metadata_json.hpp"
#include "request.hpp"
namespace symphony::sqmv::administration {
using namespace symphony::sqv_admin;
namespace support = administration_support;
Json administer(std::string_view operation, const Json &p, std::int64_t) {
  Manifest manifest;
  std::string projection = "all";
  if (operation == "metadata_validate") {
    fields(p, {"protocol", "description", "limits"});
    accepted(Manifest::create(support::description(p.at("description")),
                              support::limits(p.at("limits")), manifest));
  } else {
    fields(p, {"protocol", "encoded_hex_chunks", "expected_reference", "limits",
               "projection"});
    projection = text(p, "projection", 16);
    if (projection != "all")
      (void)support::role(projection);
    auto raw = support::decode_chunks(p.at("encoded_hex_chunks"));
    accepted(Manifest::resolve(
        ByteView(reinterpret_cast<const std::uint8_t *>(raw.data()),
                 raw.size()),
        text(p, "expected_reference", 128), support::limits(p.at("limits")),
        manifest));
  }
  sqfv::Binding b;
  accepted(manifest.binding(b));
  return {
      {"metadata_reference", manifest.reference()},
      {"encoded_bytes", std::to_string(manifest.encoded().size())},
      {"encoded_hex_chunks", support::chunks(manifest.encoded())},
      {"binding", support::binding(b)},
      {"description", support::describe(manifest.description(), projection)},
      {"projection", projection},
      {"evidence_resolution", "not_performed"}};
}
} // namespace symphony::sqmv::administration
