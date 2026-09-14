#include "pdf.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include <dlfcn.h>
#include <filesystem>
#include <regex>
#include <set>
#include <sstream>
namespace symphony::knowledge::shv_pdf {
namespace {
[[noreturn]] void bad() {
  throw engine::Error("shv-pdf.invalid", "invalid bounded PDF extraction", 4);
}
void fields(const Json &j, std::initializer_list<const char *> keys) {
  if (!j.is_object() || j.size() != keys.size())
    bad();
  for (auto k : keys)
    if (!j.contains(k))
      bad();
}
std::string str(const Json &j) {
  if (!j.is_string())
    bad();
  auto s = j.get<std::string>();
  if (s.empty() || s.size() > 4096 || s.find('\0') != s.npos)
    bad();
  return s;
}
Json seal(Json j) {
  j["digest"] = engine::tagged_sha256(j.dump());
  return j;
}
struct Decoder {
  bool initialized = false;
  void *handle = nullptr;
  void *doc = nullptr;
  void *page = nullptr;
  void *text = nullptr;
  void (*destroy)() = nullptr;
  void (*close_doc)(void *) = nullptr;
  void (*close_page)(void *) = nullptr;
  void (*close_text)(void *) = nullptr;
  template <class T> T sym(const char *n) {
    auto p = dlsym(handle, n);
    if (!p)
      bad();
    return reinterpret_cast<T>(p);
  }
  ~Decoder() {
    if (text && close_text)
      close_text(text);
    if (page && close_page)
      close_page(page);
    if (doc && close_doc)
      close_doc(doc);
    if (initialized && destroy)
      destroy();
    if (handle)
      dlclose(handle);
  }
};
std::string ascii_lines(const std::vector<unsigned short> &w, int n) {
  std::string s;
  for (int i = 0; i < n - 1; ++i) {
    unsigned c = w.at(static_cast<std::size_t>(i));
    if (c == 13)
      continue;
    // The selected profile needs exact ASCII identifiers and headings; retain
    // other Unicode scalars as UTF-8, rejecting malformed surrogate sequences.
    if (c >= 0xd800 && c <= 0xdbff) {
      if (++i >= n - 1)
        bad();
      auto low = w.at(static_cast<std::size_t>(i));
      if (low < 0xdc00 || low > 0xdfff)
        bad();
      c = 0x10000 + ((c - 0xd800) << 10) + (low - 0xdc00);
    } else if (c >= 0xdc00 && c <= 0xdfff)
      bad();
    if (c == 0)
      bad();
    if (c < 0x80)
      s += static_cast<char>(c);
    else if (c < 0x800) {
      s += static_cast<char>(0xc0 | (c >> 6));
      s += static_cast<char>(0x80 | (c & 63));
    } else if (c < 0x10000) {
      s += static_cast<char>(0xe0 | (c >> 12));
      s += static_cast<char>(0x80 | ((c >> 6) & 63));
      s += static_cast<char>(0x80 | (c & 63));
    } else {
      s += static_cast<char>(0xf0 | (c >> 18));
      s += static_cast<char>(0x80 | ((c >> 12) & 63));
      s += static_cast<char>(0x80 | ((c >> 6) & 63));
      s += static_cast<char>(0x80 | (c & 63));
    }
  }
  return s;
}
Json extract(const engine::Request &r) {
  auto p = r.payload;
  fields(p, {"source_root", "source", "decoder_root", "decoder", "profile"});
  fields(p["source"], {"id", "path", "bytes", "digest"});
  fields(p["decoder"], {"path", "digest"});
  if (p["profile"] != "amd-69290-table8.v1")
    bad();
  auto source = p["source"];
  auto id = str(source["id"]);
  if (!std::regex_match(id, std::regex("[A-Za-z0-9._-]{1,128}")))
    bad();
  auto bytes = engine::read_regular_file_no_follow(
      str(p["source_root"]), str(source["path"]), 1048576, r.deadline_unix_ms);
  if (!source["bytes"].is_number_integer() || source["bytes"] != bytes.size() ||
      source["digest"] != engine::tagged_sha256(bytes))
    bad();
  auto dr = std::filesystem::path(str(p["decoder_root"]));
  if (!dr.is_absolute())
    bad();
  auto dp = str(p["decoder"]["path"]);
  auto lib =
      engine::read_regular_file_no_follow(dr, dp, 67108864, r.deadline_unix_ms);
  if (p["decoder"]["digest"] != engine::tagged_sha256(lib))
    bad();
  Decoder d;
  d.handle = dlopen((dr / dp).c_str(), RTLD_NOW | RTLD_LOCAL);
  if (!d.handle)
    bad();
  auto init = d.sym<void (*)()>("FPDF_InitLibrary");
  d.destroy = d.sym<void (*)()>("FPDF_DestroyLibrary");
  d.close_doc = d.sym<void (*)(void *)>("FPDF_CloseDocument");
  d.close_page = d.sym<void (*)(void *)>("FPDF_ClosePage");
  d.close_text = d.sym<void (*)(void *)>("FPDFText_ClosePage");
  init();
  d.initialized = true;
  d.doc = d.sym<void *(*)(const void *, size_t, const char *)>(
      "FPDF_LoadMemDocument64")(bytes.data(), bytes.size(), nullptr);
  if (!d.doc)
    bad();
  int pages = d.sym<int (*)(void *)>("FPDF_GetPageCount")(d.doc);
  if (pages != 16)
    bad();
  d.page = d.sym<void *(*)(void *, int)>("FPDF_LoadPage")(d.doc, 12);
  if (!d.page)
    bad();
  d.text = d.sym<void *(*)(void *)>("FPDFText_LoadPage")(d.page);
  if (!d.text)
    bad();
  int count = d.sym<int (*)(void *)>("FPDFText_CountChars")(d.text);
  if (count < 1 || count > 32768)
    bad();
  std::vector<unsigned short> buffer(static_cast<size_t>(count) + 1);
  int copied = d.sym<int (*)(void *, int, int, unsigned short *)>(
      "FPDFText_GetText")(d.text, 0, count, buffer.data());
  if (copied < 1 || copied > count + 1 ||
      buffer[static_cast<size_t>(copied - 1)] != 0)
    bad();
  auto text = ascii_lines(buffer, copied);
  auto rows = parse_table(text);
  auto lib_after =
      engine::read_regular_file_no_follow(dr, dp, 67108864, r.deadline_unix_ms);
  if (lib_after != lib)
    bad();
  return seal(Json{{"protocol", "symphony.shv.pdf-extraction.v1"},
                   {"profile", p["profile"]},
                   {"source", source},
                   {"decoder", p["decoder"]},
                   {"page_index", 12},
                   {"page_count", pages},
                   {"text", text},
                   {"text_digest", engine::tagged_sha256(text)},
                   {"rows", rows},
                   {"namespace", "AMD.OPN"},
                   {"namespace_equivalence", "not_asserted"},
                   {"documentary_lineages", 1}});
}
} // namespace
Json parse_table(const std::string &text) {
  if (text.find("69290 Rev. 1.00 April 2026") == text.npos ||
      text.find("Table 8. EPYC 7003 Series Portfolio APP Values") == text.npos)
    bad();
  const std::string header = "OPN Model\nNumber\nAPP \n(Weighted TFLOPS)\n";
  auto start = text.find(header);
  if (start == text.npos || text.find(header, start + 1) != text.npos)
    bad();
  start += header.size();
  auto end = text.find("[Public]", start);
  if (end == text.npos || text.find_first_not_of(" \n\t", end + 8) != text.npos)
    bad();
  Json rows = Json::array();
  std::set<std::string> ids, models;
  std::istringstream lines(text.substr(start, end - start));
  std::string line;
  std::regex row("(100-[0-9]{9}) ([0-9A-Z]{4,8}) ([0-9]+\\.[0-9]+)");
  while (std::getline(lines, line)) {
    if (line.empty())
      continue;
    std::smatch m;
    if (!std::regex_match(line, m, row) || rows.size() >= 32 ||
        !ids.insert(m[1]).second || !models.insert(m[2]).second)
      bad();
    rows.push_back(Json{{"opn", m[1].str()},
                        {"model", m[2].str()},
                        {"source_metric_text", m[3].str()}});
  }
  if (rows.size() != 28)
    bad();
  return rows;
}
Json handle_request(const engine::Request &r) {
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    bad();
  Json result;
  if (r.operation == "inspect") {
    fields(r.payload, {});
    result = descriptor();
  } else if (r.operation == "extract")
    result = extract(r);
  else
    bad();
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    bad();
  return result;
}
} // namespace symphony::knowledge::shv_pdf
