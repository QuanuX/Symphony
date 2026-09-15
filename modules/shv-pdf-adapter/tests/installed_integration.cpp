#include "installed_support.hpp"
using namespace native_test;
void test_installed_pdf_source_replay(const Arguments &args) {
  Evidence evidence(args);
  auto prefix = fs::canonical(args.require("prefix"));
  auto receipt = prefix / "share/symphony/receipts/shv-pdf-adapter" /
                 args.require("version") / "install-receipt.json";
  auto installation = evidence.installation(
      "shv-pdf-adapter", "symphony-shv-pdf", {"pdf"}, receipt);
  auto call = [&](const std::string &op, const Json &payload,
                  const std::string &error = "") {
    return evidence.call(installation, op, payload, error);
  };
  auto request = read_json(args.require("request"));
  TempDir temporary("shv-pdf-installed-");
  auto root = temporary.path;
  auto original = fs::path(request["source_root"].get<std::string>()) /
                  request["source"]["path"].get<std::string>();
  auto local = root / "original.pdf";
  fs::copy_file(original, local);
  request["source_root"] = root.string();
  request["source"]["path"] = local.filename().string();
  auto extracted = call("extract", request);
  NT_REQUIRE(extracted["rows"].size() == 28 &&
             extracted["documentary_lineages"] == 1);
  NT_REQUIRE(extracted["namespace"] == "AMD.OPN" &&
             extracted["namespace_equivalence"] == "not_asserted");
  auto graph = call("graph_project", request);
  NT_REQUIRE(graph["nodes"].size() == 30 && graph["edges"].size() == 57);
  call("graph_validate", {{"request", request}, {"graph", graph}});
  auto forged = graph;
  forged["edges"][0]["properties"]["qualifier"] =
      "issuer=AMD;namespace=product-id-tray;profile=1";
  call("graph_validate", {{"request", request}, {"graph", seal(forged)}},
       "shv-pdf.invalid");
  auto wrong_decoder = request;
  wrong_decoder["decoder"]["digest"] = "sha256:" + std::string(64, '0');
  call("extract", wrong_decoder, "shv-pdf.invalid");
  write(local, "changed retained original");
  call("graph_validate", {{"request", request}, {"graph", graph}},
       "shv-pdf.invalid");
  evidence.finish("test_installed_pdf_source_replay");
}
int main(int argc, char **argv) {
  return test_main(
      [&] { test_installed_pdf_source_replay(Arguments(argc, argv)); });
}
