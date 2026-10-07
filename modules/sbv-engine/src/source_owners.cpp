#include "source_owners.hpp"
#include <algorithm>
#include <array>
#include <fcntl.h>
#include <filesystem>
#include <limits>
#include <symphony/knowledge/engine/path.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>

namespace symphony::sbv::detail {
namespace {
namespace db = sqav::databento;
constexpr std::string_view profile = "sqv-local-retained-capture.v1";
constexpr std::string_view retained_protocol =
    "symphony.sbv.retained-source.v1";
std::string_view name(db::Status x) {
  constexpr std::array a{
      "ok",    "invalid_argument", "unsupported", "malformed",
      "limit", "binding_mismatch", "no_memory",   "internal_error"};
  return a.at(static_cast<std::size_t>(x));
}
std::string_view name(sqav::Status x) {
  constexpr std::array a{"ok",
                         "invalid_argument",
                         "limit",
                         "no_memory",
                         "corrupt_capture",
                         "unsupported_capture",
                         "reference_mismatch",
                         "binding_mismatch",
                         "stale",
                         "closed",
                         "internal_error"};
  return a.at(static_cast<std::size_t>(x));
}
std::string_view name(sqmv::Status x) {
  constexpr std::array a{"ok",
                         "invalid_argument",
                         "limit",
                         "no_memory",
                         "unsupported_manifest",
                         "corrupt_manifest",
                         "reference_mismatch",
                         "binding_mismatch",
                         "internal_error"};
  return a.at(static_cast<std::size_t>(x));
}
std::string_view name(sqfv::Status x) {
  constexpr std::array a{"ok",
                         "invalid_argument",
                         "limit",
                         "no_memory",
                         "blocked",
                         "duplicate",
                         "conflict",
                         "gap",
                         "stale",
                         "scope_mismatch",
                         "binding_mismatch",
                         "closed",
                         "empty",
                         "corrupt_frame",
                         "unsupported_frame",
                         "internal_error"};
  return a.at(static_cast<std::size_t>(x));
}
std::string_view name(sqpv::Status x) {
  constexpr std::array a{"ok",
                         "duplicate",
                         "invalid_argument",
                         "binding_mismatch",
                         "limit",
                         "busy",
                         "conflict",
                         "gap",
                         "stale",
                         "missing",
                         "corrupt",
                         "unsafe_path",
                         "io_error",
                         "outcome_uncertain",
                         "closed",
                         "no_memory",
                         "unsupported"};
  return a.at(static_cast<std::size_t>(x));
}
std::string_view name(sqdv::Status x) {
  constexpr std::array a{
      "ok",          "invalid_argument",  "limit",    "no_memory",
      "blocked",     "duplicate",         "conflict", "gap",
      "stale",       "missing",           "corrupt",  "unsafe_path",
      "io_error",    "outcome_uncertain", "closed",   "empty",
      "unsupported", "binding_mismatch",  "busy",     "internal_error"};
  return a.at(static_cast<std::size_t>(x));
}
template <class S> void owner_ok(S status, const char *operation) {
  if (status != S::ok) {
    const bool uncertain = name(status) == "outcome_uncertain";
    throw e::Error(
        "sbv.source_owner." + std::string(name(status)),
        std::string(operation) + ": " + std::string(name(status)) +
            (uncertain
                 ? "; reopen and reconcile exact owner identity before retry"
                 : ""),
        uncertain ? 5 : 2);
  }
}
template <class N> N number(const Json &j) {
  const auto n = u64(j);
  need(n <= std::numeric_limits<N>::max(),
       "owner integer representation exceeded");
  return static_cast<N>(n);
}
std::string hash_string(const Json &j) {
  auto s = str(j);
  need(s.size() == 64 &&
           s.find_first_not_of("0123456789abcdef") == std::string::npos,
       "lowercase SHA-256 required");
  return s;
}
std::string local_path(const Json &j) {
  auto s = str(j);
  need(s.size() > 1 && s.front() == '/' &&
           e::is_safe_relative_path(s.substr(1)),
       "absolute non-symlink local path required");
  return s;
}
struct Directory {
  int fd = -1;
  Directory() : fd(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC)) {
    need(fd >= 0, "filesystem root unavailable");
  }
  ~Directory() {
    if (fd >= 0)
      ::close(fd);
  }
  Directory(const Directory &) = delete;
  void descend(const std::filesystem::path &part) {
    const auto next = ::openat(fd, part.c_str(),
                               O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    need(next >= 0, "directory ancestor unavailable or symlinked");
    ::close(fd);
    fd = next;
  }
  struct stat identity() const {
    struct stat s{};
    need(::fstat(fd, &s) == 0, "directory identity unavailable");
    return s;
  }
};
void outside_store(const std::string &output, const std::string &root) {
  Directory store;
  for (const auto &part : std::filesystem::path(root).relative_path())
    store.descend(part);
  const auto owner = store.identity();
  Directory parent;
  auto check = [&] {
    const auto actual = parent.identity();
    need(actual.st_dev != owner.st_dev || actual.st_ino != owner.st_ino,
         "SBV publication must be outside the exact SQPV store directory");
  };
  check();
  for (const auto &part :
       std::filesystem::path(output).parent_path().relative_path()) {
    parent.descend(part);
    check();
  }
  // require_new_file separately rejects an existing store root used as the
  // output leaf. Identity comparisons also cover filesystem case/Unicode
  // aliases.
}
template <class T> std::string hex(const T &bytes) {
  constexpr char digits[] = "0123456789abcdef";
  std::string s;
  for (auto b : bytes) {
    s += digits[b >> 4];
    s += digits[b & 15];
  }
  return s;
}
sqfv::Generation generation(const Json &j) {
  const auto s = str(j);
  need(s.size() == 32 &&
           s.find_first_not_of("0123456789abcdef") == std::string::npos,
       "exact lowercase 16-byte generation required");
  sqfv::Generation g{};
  for (std::size_t i = 0; i < g.size(); ++i) {
    unsigned v = 0;
    const auto [at, error] =
        std::from_chars(s.data() + 2 * i, s.data() + 2 * i + 2, v, 16);
    need(error == std::errc{} && at == s.data() + 2 * i + 2,
         "invalid generation");
    g[i] = static_cast<std::uint8_t>(v);
  }
  return g;
}
sqav::ByteView bytes(const std::string &s) {
  return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}
Json versions() {
  return {{"sqav-databento-dbn-cpp", "0.5.0-dev"},
          {"sqav-capture-cpp", "0.2.0-dev"},
          {"sqmv-metadata-cpp", "0.2.0-dev"},
          {"sqfv-batch-cpp", "0.3.0-dev"},
          {"sqpv-local-store-cpp", "0.2.0-dev"},
          {"sqdv-delivery-cpp", "0.3.0-dev"},
          {"knowledge-vector-engine-cpp", "0.2.0-dev"}};
}
struct Limits {
  db::Limits dbn;
  sqav::Limits capture;
  sqmv::Limits metadata;
  sqfv::Limits flow;
  sqpv::Limits store;
  explicit Limits(const Json &j) {
    keys(j, {"dbn", "capture", "metadata", "flow", "store"});
    const auto &d = j.at("dbn"), &c = j.at("capture"), &m = j.at("metadata"),
               &f = j.at("flow"), &s = j.at("store");
    keys(d, {"max_file_bytes", "max_metadata_bytes", "max_records"});
    keys(c, {"max_capture_bytes", "max_metadata_bytes", "max_field_bytes"});
    keys(m, {"max_manifest_bytes", "max_field_bytes", "max_evidence_refs"});
    keys(f, {"max_payload_bytes", "max_frame_bytes", "max_descriptor_bytes",
             "global_allocation_bytes", "max_ports"});
    keys(s, {"max_frame_bytes", "max_store_bytes", "max_batches"});
    dbn = {u64(d.at("max_file_bytes")),
           number<std::uint32_t>(d.at("max_metadata_bytes")),
           u64(d.at("max_records"))};
    capture = {u64(c.at("max_capture_bytes")),
               number<std::uint32_t>(c.at("max_metadata_bytes")),
               number<std::uint16_t>(c.at("max_field_bytes"))};
    metadata = {number<std::uint32_t>(m.at("max_manifest_bytes")),
                number<std::uint32_t>(m.at("max_field_bytes")),
                number<std::uint16_t>(m.at("max_evidence_refs"))};
    flow = {u64(f.at("max_payload_bytes")), u64(f.at("max_frame_bytes")),
            u64(f.at("max_descriptor_bytes")),
            u64(f.at("global_allocation_bytes")),
            number<std::uint32_t>(f.at("max_ports"))};
    store = {u64(s.at("max_frame_bytes")), u64(s.at("max_store_bytes")),
             number<std::uint32_t>(s.at("max_batches"))};
    // Validate this selected release before reading/allocating source bytes or
    // creating a store. These are not constraints on standalone SBV datasets.
    need(dbn.max_file_bytes > 0 && dbn.max_file_bytes <= (64U << 20) &&
             dbn.max_metadata_bytes > 0 &&
             dbn.max_metadata_bytes <= (1U << 20) && dbn.max_records > 0 &&
             dbn.max_records <= 1048576,
         "selected DBN capture profile limits unsupported");
    need(capture.max_capture_bytes > 0 &&
             capture.max_capture_bytes <= (64U << 20) &&
             capture.max_metadata_bytes > 0 &&
             capture.max_metadata_bytes <= 65536 &&
             capture.max_field_bytes > 0 && capture.max_field_bytes <= 4096,
         "selected SQAV profile limits unsupported");
    need(metadata.max_manifest_bytes > 0 &&
             metadata.max_manifest_bytes <= 65536 &&
             metadata.max_field_bytes > 0 && metadata.max_field_bytes <= 4096 &&
             metadata.max_evidence_refs > 0 &&
             metadata.max_evidence_refs <= 128,
         "selected SQMV profile limits unsupported");
    need(flow.max_payload_bytes > 0 && flow.max_payload_bytes <= (64U << 20) &&
             flow.max_frame_bytes > 0 && flow.max_frame_bytes <= (128U << 20) &&
             flow.max_descriptor_bytes > 0 &&
             flow.max_descriptor_bytes <= 65536 &&
             flow.global_allocation_bytes > 0 && flow.max_ports > 0 &&
             flow.max_ports <= 1024,
         "selected SQFV profile limits unsupported");
    need(store.max_frame_bytes > 0 && store.max_frame_bytes <= (64U << 20) &&
             store.max_store_bytes > 0 &&
             store.max_store_bytes <= (1ULL << 40) && store.max_batches > 0 &&
             store.max_batches <= 65536,
         "selected SQPV profile limits unsupported");
  }
};
sqav::Description capture_description(const Json &j) {
  keys(j, {"source", "attempt_id", "attribution_ref", "source_position",
           "coverage", "coverage_scope", "coverage_evidence_ref",
           "source_record_count", "times"});
  const auto &s = j.at("source");
  keys(s, {"provider_ref", "interface_ref", "interface_version", "operation",
           "adapter_ref", "adapter_version", "dataset_id", "dataset_revision",
           "selection_ref", "native_schema_ref", "native_encoding_ref",
           "access_scope"});
  sqav::Description d;
  d.source = {str(s.at("provider_ref")),        str(s.at("interface_ref")),
              str(s.at("interface_version")),   str(s.at("operation")),
              str(s.at("adapter_ref")),         str(s.at("adapter_version")),
              str(s.at("dataset_id")),          str(s.at("dataset_revision")),
              str(s.at("selection_ref")),       str(s.at("native_schema_ref")),
              str(s.at("native_encoding_ref")), str(s.at("access_scope"))};
  d.attempt_id = str(j.at("attempt_id"));
  d.attribution_ref = str(j.at("attribution_ref"));
  d.source_position = str(j.at("source_position"));
  d.coverage_scope = str(j.at("coverage_scope"));
  d.coverage_evidence_ref = str(j.at("coverage_evidence_ref"));
  const auto coverage = str(j.at("coverage"));
  need(coverage == "unknown" || coverage == "complete" ||
           coverage == "partial" || coverage == "gap",
       "unknown capture coverage");
  d.coverage = coverage == "unknown"    ? sqav::Coverage::unknown
               : coverage == "complete" ? sqav::Coverage::complete
               : coverage == "partial"  ? sqav::Coverage::partial
                                        : sqav::Coverage::gap;
  if (!j.at("source_record_count").is_null())
    d.source_record_count = u64(j.at("source_record_count"));
  need(j.at("times").is_array(), "capture times array required");
  for (const auto &t : j.at("times")) {
    keys(t, {"role", "value", "format_ref", "clock_ref", "precision_ref",
             "evidence_ref"});
    const auto role = str(t.at("role"));
    constexpr std::array roles{"acquisition", "event", "publication",
                               "revision", "receipt"};
    const auto found = std::find(roles.begin(), roles.end(), role);
    need(found != roles.end(), "unknown capture time role");
    d.times.push_back({static_cast<sqav::TimeRole>(found - roles.begin() + 1),
                       str(t.at("value")), str(t.at("format_ref")),
                       str(t.at("clock_ref")), str(t.at("precision_ref")),
                       str(t.at("evidence_ref"))});
  }
  return d;
}
Json capture_description(const sqav::Description &d) {
  const auto &s = d.source;
  Json times = Json::array();
  constexpr std::array roles{"acquisition", "event", "publication", "revision",
                             "receipt"};
  constexpr std::array coverage{"unknown", "complete", "partial", "gap"};
  for (const auto &t : d.times)
    times.push_back({{"role", roles.at(static_cast<std::size_t>(t.role) - 1)},
                     {"value", t.value},
                     {"format_ref", t.format_ref},
                     {"clock_ref", t.clock_ref},
                     {"precision_ref", t.precision_ref},
                     {"evidence_ref", t.evidence_ref}});
  return {{"source",
           {{"provider_ref", s.provider_ref},
            {"interface_ref", s.interface_ref},
            {"interface_version", s.interface_version},
            {"operation", s.operation},
            {"adapter_ref", s.adapter_ref},
            {"adapter_version", s.adapter_version},
            {"dataset_id", s.dataset_id},
            {"dataset_revision", s.dataset_revision},
            {"selection_ref", s.selection_ref},
            {"native_schema_ref", s.native_schema_ref},
            {"native_encoding_ref", s.native_encoding_ref},
            {"access_scope", s.access_scope}}},
          {"attempt_id", d.attempt_id},
          {"attribution_ref", d.attribution_ref},
          {"source_position", d.source_position},
          {"coverage", coverage.at(static_cast<std::size_t>(d.coverage))},
          {"coverage_scope", d.coverage_scope},
          {"coverage_evidence_ref", d.coverage_evidence_ref},
          {"source_record_count", d.source_record_count
                                      ? Json(dec(*d.source_record_count))
                                      : Json(nullptr)},
          {"times", times}};
}
constexpr std::array evidence_roles{"schema", "layout",   "access",  "source",
                                    "time",   "coverage", "lineage", "units"};
Json manifest_description(const sqmv::Manifest &m) {
  const auto &d = m.description();
  Json evidence = Json::array();
  for (const auto &r : d.evidence)
    evidence.push_back(
        {{"role", evidence_roles.at(static_cast<std::size_t>(r.role) - 1)},
         {"producer_ref", r.producer_ref},
         {"evidence_ref", r.evidence_ref}});
  return {{"dataset_id", d.dataset_id},
          {"dataset_revision", d.dataset_revision},
          {"schema_version", d.schema_version},
          {"layout_version", d.layout_version},
          {"access_scope", d.access_scope},
          {"producer_ref", d.producer_ref},
          {"evidence", evidence}};
}
sqmv::Description manifest_description(const Json &j) {
  keys(j, {"dataset_id", "dataset_revision", "schema_version", "layout_version",
           "access_scope", "producer_ref", "evidence"});
  sqmv::Description d{str(j.at("dataset_id")),
                      str(j.at("dataset_revision")),
                      str(j.at("schema_version")),
                      str(j.at("layout_version")),
                      str(j.at("access_scope")),
                      str(j.at("producer_ref")),
                      {}};
  need(j.at("evidence").is_array(), "manifest evidence array required");
  for (const auto &r : j.at("evidence")) {
    keys(r, {"role", "producer_ref", "evidence_ref"});
    auto found = std::find(evidence_roles.begin(), evidence_roles.end(),
                           str(r.at("role")));
    need(found != evidence_roles.end(), "unknown manifest evidence role");
    d.evidence.push_back(
        {static_cast<sqmv::EvidenceRole>(found - evidence_roles.begin() + 1),
         str(r.at("producer_ref")), str(r.at("evidence_ref"))});
  }
  return d;
}
Json receipt(const sqpv::Receipt &r) {
  return {{"store_generation", hex(r.store_generation)},
          {"producer_generation", hex(r.producer_generation)},
          {"batch_sequence", dec(r.batch_sequence)},
          {"frame_bytes", dec(r.frame_bytes)},
          {"content_id", hex(r.content_id)},
          {"frame_sha256", r.frame_sha256},
          {"commit_sha256", r.commit_sha256}};
}
Json descriptor(const sqfv::Descriptor &d) {
  return {{"binding",
           {{"metadata_ref", d.binding.metadata_ref},
            {"dataset_revision", d.binding.dataset_revision},
            {"schema_version", d.binding.schema_version},
            {"layout_version", d.binding.layout_version},
            {"access_scope", d.binding.access_scope}}},
          {"partition", d.partition},
          {"source_binding", d.source_binding},
          {"source_position", d.source_position},
          {"producer_generation", hex(d.producer_generation)},
          {"batch_sequence", dec(d.batch_sequence)},
          {"record_count", dec(d.record_count)}};
}
Json snapshot(sqpv::Store &store) {
  sqpv::Snapshot s;
  owner_ok(store.snapshot(s), "SQPV snapshot");
  return {{"committed_batches", dec(s.committed_batches)},
          {"next_sequence", dec(s.next_sequence)},
          {"store_bytes", dec(s.store_bytes)},
          {"sequence_exhausted", s.sequence_exhausted},
          {"staged", s.staged}};
}
Json snapshot(const sqpv::Snapshot &s) {
  return {{"committed_batches", dec(s.committed_batches)},
          {"next_sequence", dec(s.next_sequence)},
          {"store_bytes", dec(s.store_bytes)},
          {"sequence_exhausted", s.sequence_exhausted},
          {"staged", s.staged}};
}
Json stats(const sqdv::Session &session) {
  sqdv::SessionStats s;
  owner_ok(session.stats(s), "SQDV stats");
  return {{"next_offer_sequence", dec(s.next_offer_sequence)},
          {"next_processed_sequence", dec(s.next_processed_sequence)},
          {"outstanding_bytes", dec(s.outstanding_bytes)},
          {"unacknowledged_batches", dec(s.unacknowledged_batches)},
          {"pending_deliveries", dec(s.pending_deliveries)},
          {"offer_exhausted", s.offer_exhausted},
          {"processed_exhausted", s.processed_exhausted}};
}
Json checkpoint(const sqdv::Session &session) {
  sqdv::Checkpoint c;
  owner_ok(session.checkpoint(c), "SQDV checkpoint");
  return {{"view_reference", c.view_reference},
          {"next_sequence", dec(c.next_sequence)},
          {"sequence_exhausted", c.sequence_exhausted}};
}
sqmv::EvidenceReference access(const Json &j) {
  keys(j, {"producer_ref", "evidence_ref"});
  return {sqmv::EvidenceRole::access, str(j.at("producer_ref")),
          str(j.at("evidence_ref"))};
}
Json options(const sqpv::Options &o) {
  return {{"partition", o.partition},
          {"producer_generation", hex(o.producer_generation)},
          {"store_generation", hex(o.store_generation)},
          {"first_sequence", dec(o.first_sequence)},
          {"limits",
           {{"max_frame_bytes", dec(o.limits.max_frame_bytes)},
            {"max_store_bytes", dec(o.limits.max_store_bytes)},
            {"max_batches", dec(o.limits.max_batches)}}}};
}
sqpv::Options options(const Json &j, const Limits &l) {
  keys(j, {"partition", "producer_generation", "store_generation",
           "first_sequence", "limits"});
  sqpv::Options o{str(j.at("partition")),
                  generation(j.at("producer_generation")),
                  generation(j.at("store_generation")),
                  u64(j.at("first_sequence")), l.store};
  need(options(o) == j, "retained store options and limits disagree");
  return o;
}
Json original(const std::string &path, const db::FileView &v) {
  const auto &m = v.metadata();
  return {{"source_path", path},
          {"source_sha256", e::sha256_hex(v.original())},
          {"dataset", m.dataset},
          {"bytes", dec(v.original().size())},
          {"record_count", dec(m.record_count)},
          {"dbn_version", dec(m.version)},
          {"native_encoding", db::encoding_for_version(m.version)},
          {"encoded_metadata_sha256", e::sha256_hex(v.encoded_metadata())}};
}
Json capture_evidence(const sqav::Capture &c) {
  return {{"reference", c.reference()},
          {"source_reference", c.source_reference()},
          {"encoded_sha256", e::sha256_hex(c.encoded())},
          {"bytes", dec(c.encoded().size())},
          {"description", capture_description(c.description())}};
}
Json manifest_evidence(const sqmv::Manifest &m) {
  return {{"reference", m.reference()},
          {"encoded_sha256", e::sha256_hex(m.encoded())},
          {"bytes", dec(m.encoded().size())},
          {"description", manifest_description(m)}};
}
} // namespace

Json source_owner_profile() {
  return {{"protocol", "symphony.sbv.source-owner-profile.v1"},
          {"id", profile},
          {"scope", "optional composed SQAV/SQMV/SQFV/SQPV/SQDV route; not "
                    "standalone SBV or general SQPV limits"},
          {"platform", "macos_local_apfs"},
          {"owner_versions", versions()},
          {"maximum_root_bytes", "4096"},
          {"maximum_partition_bytes", "4096"},
          {"maximum_delivery_field_bytes", "4096"},
          {"maximum_delivery_credit_bytes", "67108864"},
          {"maximum_unacknowledged_batches", "65536"},
          {"checkpoint", "null_only"},
          {"deadline_policy", "caller_optional"},
          {"limit_maxima",
           {{"dbn",
             {{"max_file_bytes", "67108864"},
              {"max_metadata_bytes", "1048576"},
              {"max_records", "1048576"}}},
            {"capture",
             {{"max_capture_bytes", "67108864"},
              {"max_metadata_bytes", "65536"},
              {"max_field_bytes", "4096"}}},
            {"metadata",
             {{"max_manifest_bytes", "65536"},
              {"max_field_bytes", "4096"},
              {"max_evidence_refs", "128"}}},
            {"flow",
             {{"max_payload_bytes", "67108864"},
              {"max_frame_bytes", "134217728"},
              {"max_descriptor_bytes", "65536"},
              {"global_allocation_bytes", dec(UINT64_MAX)},
              {"max_ports", "1024"}}},
            {"store",
             {{"max_frame_bytes", "67108864"},
              {"max_store_bytes", "1099511627776"},
              {"max_batches", "65536"}}}}}};
}
Json source_retain(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "output_path", "source", "capture_description",
           "access_evidence", "owner_profile", "limits", "retention",
           "extensions"});
  need(p.at("protocol") == "symphony.sbv.source-retain-input.v1",
       "source-retain protocol required");
  need(p.at("owner_profile") == profile && p.at("extensions").is_object(),
       "unsupported owner profile or extensions");
  keys(p.at("source"), {"path", "expected_sha256", "dataset"});
  const auto path = local_path(p.at("source").at("path"));
  const auto expected = hash_string(p.at("source").at("expected_sha256"));
  const auto dataset = str(p.at("source").at("dataset"));
  require_new_file(str(p.at("output_path")));
  deadline(end);
  const Limits limits(p.at("limits"));
  const auto description = capture_description(p.at("capture_description"));
  need(description.source.dataset_id == dataset,
       "source and capture dataset disagree");
  const auto access_evidence = access(p.at("access_evidence"));
  const auto &retention = p.at("retention");
  keys(retention, {"mode", "root", "partition", "producer_generation",
                   "store_generation", "first_sequence", "batch_sequence"});
  const auto root = local_path(retention.at("root"));
  const auto output = local_path(p.at("output_path"));
  outside_store(output, root);
  const auto mode = str(retention.at("mode"));
  need(mode == "create" || mode == "open",
       "explicit create/open retention mode required");
  sqpv::Options selected{str(retention.at("partition")),
                         generation(retention.at("producer_generation")),
                         generation(retention.at("store_generation")),
                         u64(retention.at("first_sequence")), limits.store};
  need(root.size() <= 4096 && !selected.partition.empty() &&
           selected.partition.size() <= 4096,
       "selected composed owner route requires root/partition within SQDV "
       "4096-byte bounds");
  const auto sequence = u64(retention.at("batch_sequence"));
  need(sequence >= selected.first_sequence,
       "batch precedes selected store baseline");
  need(mode != "create" || sequence == selected.first_sequence,
       "new store first append must equal its selected first sequence");
  auto result = base("source_retain");
  result["sections"]["choices"] = section(p);
  validate_result(seal_result(
      result)); // Reject unrepresentable choices before persistent work.
  sqfv::Context flow;
  owner_ok(sqfv::Context::create(limits.flow, flow), "SQFV create");
  const auto raw = e::read_regular_file_no_follow(
      "/", path.substr(1), static_cast<std::size_t>(limits.dbn.max_file_bytes),
      end);
  need(e::sha256_hex(raw) == expected, "source original SHA-256 mismatch");
  sqav::Capture capture;
  owner_ok(db::capture_file(description, bytes(raw), limits.dbn, limits.capture,
                            capture),
           "Databento capture_file");
  db::FileView view;
  owner_ok(db::inspect_capture(capture, limits.dbn, view),
           "Databento inspect_capture");
  sqmv::Manifest manifest;
  owner_ok(capture.metadata(false, access_evidence, limits.metadata, manifest),
           "SQAV metadata factory");
  sqfv::Batch batch;
  owner_ok(capture.prepare(
               flow, manifest,
               {selected.partition, selected.producer_generation, sequence},
               batch),
           "SQAV prepare capture batch");
  Json source{{"protocol", retained_protocol},
              {"owner_profile", profile},
              {"owner_versions", versions()},
              {"profile_contract", source_owner_profile()},
              {"original", original(path, view)},
              {"capture", capture_evidence(capture)},
              {"metadata", manifest_evidence(manifest)},
              {"batch",
               {{"descriptor", descriptor(batch.descriptor())},
                {"content_id", hex(batch.content_id())}}},
              {"limits", p.at("limits")},
              {"access_evidence", p.at("access_evidence")},
              {"retention", {{"root", root}, {"options", options(selected)}}},
              {"delivery", {{"status", "not_selected"}}}};
  auto recovery = [&](const char *code, const char *stage, bool confirmed,
                      Json actual_receipt) {
    return Json{{"protocol", "symphony.sbv.source-retain.v1"},
                {"status", "recovery_required"},
                {"code", code},
                {"stage", stage},
                {"request_sha256", e::sha256_hex(p.dump())},
                {"output_path", p.at("output_path")},
                {"automatic_retry", false},
                {"rollback_performed", false},
                {"recovery",
                 {{"retention_confirmed", confirmed},
                  {"root", root},
                  {"options", options(selected)},
                  {"metadata", source.at("metadata")},
                  {"batch", source.at("batch")},
                  {"receipt", std::move(actual_receipt)}}}};
  };
  deadline(end);
  sqpv::Store store;
  const auto opening =
      mode == "create" ? sqpv::Store::create(root, manifest, selected, store)
                       : sqpv::Store::open(root, manifest, selected, store);
  if (opening == sqpv::Status::outcome_uncertain)
    return recovery("sbv.source_owner.outcome_uncertain",
                    mode == "create" ? "store_create" : "store_open", false,
                    nullptr);
  if (opening != sqpv::Status::ok)
    throw e::Error(
        "sbv.source_owner." + std::string(name(opening)),
        "SQPV " + mode + ": " + std::string(name(opening)) +
            "; reopen and reconcile exact identity before retry; root=" + root +
            "; exact requested options=" + options(selected).dump(),
        opening == sqpv::Status::outcome_uncertain ? 5 : 2);
  sqpv::Receipt retained_receipt;
  const auto appended = store.append(flow, batch, retained_receipt);
  if (appended == sqpv::Status::outcome_uncertain)
    return recovery("sbv.source_owner.outcome_uncertain", "store_append", false,
                    nullptr);
  if (appended != sqpv::Status::ok && appended != sqpv::Status::duplicate)
    throw e::Error("sbv.source_owner." + std::string(name(appended)),
                   "SQPV append: " + std::string(name(appended)) +
                       "; store exists; reopen and reconcile exact identity "
                       "before retry; root=" +
                       root +
                       "; exact requested options=" + options(selected).dump() +
                       "; batch=" + source.at("batch").dump(),
                   appended == sqpv::Status::outcome_uncertain ? 5 : 2);
  // After this point the retained batch exists even if result publication
  // fails. Preserve the actual receipt in any error; retry must use the exact
  // same source metadata/options, not a newly generated identity or sampled
  // clock.
  try {
    source["retention"]["receipt"] = receipt(retained_receipt);
    source["retention"]["snapshot"] = snapshot(store);
    source["retention"]["append_status"] =
        appended == sqpv::Status::duplicate ? "duplicate" : "committed";
    store.reset();
    auto &sections = result["sections"];
    sections["source"] = section(source);
    sections["summary"] =
        section({{"source_sha256", expected},
                 {"dataset", dataset},
                 {"source_bytes", dec(view.original().size())},
                 {"source_records", dec(view.metadata().record_count)},
                 {"capture_reference", capture.reference()},
                 {"retention", source.at("retention").at("append_status")},
                 {"delivery", "not_selected"}});
    sections["provenance"] =
        section({{"engine_version", version},
                 {"source_path", path},
                 {"source_sha256", expected},
                 {"source_authorship", "not_verified"},
                 {"acquisition", "offline_local_file"},
                 {"owner_versions", versions()},
                 {"requested_capture_description", p.at("capture_description")},
                 {"resolved_capture_description",
                  capture_description(capture.description())},
                 {"provider_calls", "0"},
                 {"transformation", "none"}});
    sections["diagnostics"] =
        section({{"exactly_once_claimed", false},
                 {"destination_commit_claimed", false},
                 {"initialized_book_claimed", false},
                 {"source_authorization_verified", false}});
    sqfv::ContextStats allocation;
    owner_ok(flow.stats(allocation), "SQFV allocation stats");
    sections["resources"] = section(
        {{"owner_profile", profile},
         {"limits", p.at("limits")},
         {"flow_peak_allocation_units", dec(allocation.peak_allocation_bytes)},
         {"deadline_unix_ms",
          end == e::no_deadline ? Json(nullptr) : Json(dec(end))}});
    outside_store(output, root);
    return persist(std::move(result), p, "source-retain", end);
  } catch (const std::exception &) {
    return recovery("sbv.source_retention_unpublished", "result_publication",
                    true, receipt(retained_receipt));
  }
}

struct RetainedOriginal::Impl {
  // Reverse destruction releases capture/lease/session before source/context.
  sqfv::Context flow;
  sqmv::Manifest manifest;
  sqdv::RetainedSource source;
  sqdv::Session session;
  sqdv::Delivery delivery;
  sqav::Capture capture;
  Json identity, evidence;
  std::int64_t end;
  bool acknowledge = false, finished = false;
  Impl(const Json &p, std::int64_t d) : end(d) {
    keys(p, {"reference", "delivery"});
    const auto &ref = p.at("reference"), &choice = p.at("delivery");
    keys(ref, {"path", "expected_sha256", "pointer"});
    keys(choice, {"profile", "view_id", "recipient_id", "recipient_interface",
                  "first_sequence", "outstanding_byte_credit",
                  "max_unacknowledged_batches", "processed_ack", "checkpoint"});
    need(choice.at("profile") == "retained_before_delivery",
         "unsupported retained delivery profile");
    need(choice.at("checkpoint").is_null(),
         "durable delivery checkpoint is not implemented by this SBV adapter");
    need(choice.at("processed_ack") == "none" ||
             choice.at("processed_ack") == "dataset_admitted",
         "unsupported processing acknowledgement boundary");
    acknowledge = choice.at("processed_ack") == "dataset_admitted";
    hash_string(ref.at("expected_sha256"));
    const auto raw = read_file(local_path(ref.at("path")), end);
    auto outer = e::parse_bounded_json(raw, artifact_bytes, artifact_values);
    validate_result(outer);
    need(outer.at("content_sha256") == ref.at("expected_sha256"),
         "retained source result identity mismatch");
    const auto ptr = str(ref.at("pointer"));
    need(ptr.empty() || ptr.front() == '/', "JSON pointer required");
    const Json::json_pointer pointer(ptr);
    need(outer.contains(pointer), "retained source pointer missing");
    const auto &s = outer.at(pointer);
    keys(s, {"protocol", "owner_profile", "owner_versions", "profile_contract",
             "original", "capture", "metadata", "batch", "limits",
             "access_evidence", "retention", "delivery"});
    need(s.at("protocol") == retained_protocol &&
             s.at("owner_profile") == profile &&
             s.at("owner_versions") == versions(),
         "retained source profile/version mismatch");
    need(s.at("profile_contract") == source_owner_profile(),
         "retained composed route contract mismatch");
    need(s.at("delivery") == Json{{"status", "not_selected"}},
         "retention is not delivery evidence");
    const Limits limits(s.at("limits"));
    keys(s.at("original"),
         {"source_path", "source_sha256", "dataset", "bytes", "record_count",
          "dbn_version", "native_encoding", "encoded_metadata_sha256"});
    const auto source_path = local_path(s.at("original").at("source_path"));
    hash_string(s.at("original").at("source_sha256"));
    keys(s.at("metadata"),
         {"reference", "encoded_sha256", "bytes", "description"});
    owner_ok(sqmv::Manifest::create(
                 manifest_description(s.at("metadata").at("description")),
                 limits.metadata, manifest),
             "reconstruct SQMV manifest");
    need(manifest_evidence(manifest) == s.at("metadata"),
         "retained manifest canonical identity mismatch");
    const auto &r = s.at("retention");
    keys(r, {"root", "options", "receipt", "snapshot", "append_status"});
    need(r.at("append_status") == "committed" ||
             r.at("append_status") == "duplicate",
         "retained append was not confirmed");
    const auto selected = options(r.at("options"), limits);
    need(str(r.at("root")).size() <= 4096 && !selected.partition.empty() &&
             selected.partition.size() <= 4096,
         "retained composed owner route root/partition incompatible with SQDV");
    const auto &historical = r.at("snapshot");
    keys(historical, {"committed_batches", "next_sequence", "store_bytes",
                      "sequence_exhausted", "staged"});
    need(historical.at("sequence_exhausted").is_boolean() &&
             historical.at("staged").is_boolean(),
         "retention snapshot booleans required");
    const auto committed = u64(historical.at("committed_batches"));
    need(committed > 0 && committed <= selected.limits.max_batches &&
             u64(historical.at("store_bytes")) > 0 &&
             u64(historical.at("store_bytes")) <=
                 selected.limits.max_store_bytes &&
             historical.at("staged") == false,
         "retention snapshot confirmed prefix required");
    need(committed - 1 <= UINT64_MAX - selected.first_sequence,
         "retention prefix sequence overflow");
    const auto last = selected.first_sequence + committed - 1;
    need(u64(historical.at("next_sequence")) ==
                 (last == UINT64_MAX ? last : last + 1) &&
             historical.at("sequence_exhausted") == (last == UINT64_MAX),
         "retention snapshot cursor mismatch");
    keys(s.at("batch"), {"descriptor", "content_id"});
    keys(s.at("batch").at("descriptor"),
         {"binding", "partition", "source_binding", "source_position",
          "producer_generation", "batch_sequence", "record_count"});
    keys(s.at("batch").at("descriptor").at("binding"),
         {"metadata_ref", "dataset_revision", "schema_version",
          "layout_version", "access_scope"});
    const auto sequence =
        u64(s.at("batch").at("descriptor").at("batch_sequence"));
    need(u64(choice.at("first_sequence")) == sequence &&
             sequence >= selected.first_sequence,
         "delivery must select the retained descriptor batch sequence");
    need(sequence <= last,
         "retained batch lies outside the recorded confirmed prefix");
    sqdv::Config config{str(choice.at("view_id")),
                        str(choice.at("recipient_id")),
                        str(choice.at("recipient_interface")),
                        selected.partition,
                        selected.producer_generation,
                        sequence,
                        sqdv::Profile::retained_before_delivery};
    sqdv::Limits delivery_limits{
        u64(choice.at("outstanding_byte_credit")),
        number<std::uint32_t>(choice.at("max_unacknowledged_batches"))};
    owner_ok(sqfv::Context::create(limits.flow, flow), "SQFV retained context");
    deadline(end);
    owner_ok(sqdv::RetainedSource::open(local_path(r.at("root")), manifest,
                                        selected, source),
             "SQDV actual retained source open");
    sqpv::AsyncSnapshot retained_status;
    owner_ok(source.retention_status(retained_status),
             "SQDV actual confirmed retention status");
    const auto &confirmed = retained_status.confirmed;
    need(committed <= confirmed.committed_batches &&
             u64(historical.at("store_bytes")) <= confirmed.store_bytes,
         "recorded retention prefix exceeds actual confirmed storage");
    if (committed == confirmed.committed_batches && !confirmed.staged)
      need(historical == snapshot(confirmed),
           "recorded retention snapshot differs from actual same prefix");
    owner_ok(sqdv::Session::create(flow, manifest, config, delivery_limits,
                                   &source, nullptr, session),
             "SQDV retained session");
    const auto before = stats(session), baseline = checkpoint(session);
    owner_ok(session.offer_next(flow), "SQDV retained offer");
    owner_ok(session.take(delivery), "SQDV retained take");
    need(delivery.origin() == sqdv::Origin::retained &&
             delivery.retention_receipt() != nullptr,
         "actual retained delivery receipt required");
    need(receipt(*delivery.retention_receipt()) == r.at("receipt") &&
             descriptor(delivery.descriptor()) ==
                 s.at("batch").at("descriptor") &&
             hex(delivery.content_id()) == str(s.at("batch").at("content_id")),
         "retained batch or receipt identity mismatch");
    need(u64(historical.at("store_bytes")) >
             delivery.retention_receipt()->frame_bytes,
         "recorded store bytes cannot hold selected frame and owner metadata");
    owner_ok(sqav::Capture::from_delivery(delivery.payload(),
                                          delivery.descriptor(), manifest,
                                          limits.capture, capture),
             "SQAV retained delivery capture");
    need(capture_evidence(capture) == s.at("capture"),
         "retained capture identity mismatch");
    sqmv::Manifest factory;
    owner_ok(capture.metadata(false, access(s.at("access_evidence")),
                              limits.metadata, factory),
             "SQAV retained factory metadata");
    need(manifest_evidence(factory) == s.at("metadata"),
         "retained metadata does not describe this capture/access evidence");
    db::FileView view;
    owner_ok(db::inspect_capture(capture, limits.dbn, view),
             "DBN retained original validation");
    need(original(source_path, view) == s.at("original"),
         "retained original bytes/metadata identity mismatch");
    deadline(end);
    identity = {{"source_path", source_path},
                {"source_sha256", s.at("original").at("source_sha256")},
                {"dataset", s.at("original").at("dataset")}};
    evidence = {{"protocol", "symphony.sbv.source-delivery.v1"},
                {"reference", ref},
                {"reference_file_sha256", e::sha256_hex(raw)},
                {"source", s},
                {"choices", choice},
                {"origin", "retained"},
                {"view_reference", session.view_reference()},
                {"sequence", dec(delivery.sequence())},
                {"receipt", receipt(*delivery.retention_receipt())},
                {"before_offer", before},
                {"taken", stats(session)},
                {"baseline_checkpoint", baseline},
                {"confirmed_retention", snapshot(confirmed)},
                {"recorded_snapshot_verification",
                 committed == confirmed.committed_batches && !confirmed.staged
                     ? "exact_current_prefix"
                     : "historical_within_current_prefix"},
                {"source_authorship", "not_verified"},
                {"exactly_once_claimed", false},
                {"destination_commit_claimed", false}};
  }
};
RetainedOriginal::RetainedOriginal(const Json &p, std::int64_t end)
    : impl_(std::make_unique<Impl>(p, end)) {}
RetainedOriginal::~RetainedOriginal() = default;
RetainedOriginal::RetainedOriginal(RetainedOriginal &&) noexcept = default;
RetainedOriginal &
RetainedOriginal::operator=(RetainedOriginal &&) noexcept = default;
std::span<const std::uint8_t> RetainedOriginal::original_bytes() const {
  need(impl_ && !impl_->finished,
       "retained original span is no longer available");
  return impl_->capture.original();
}
const Json &RetainedOriginal::identity() const {
  need(impl_ != nullptr, "moved retained source handle");
  return impl_->identity;
}
Json RetainedOriginal::after_admission() {
  need(impl_ && !impl_->finished, "retained source admission is single use");
  auto &i = *impl_;
  deadline(i.end);
  // No acknowledgement occurs during construction, validation, destruction or
  // payload release. The Dataset caller invokes this only after owning decode.
  if (i.acknowledge)
    owner_ok(i.session.acknowledge_processed(i.delivery),
             "SQDV dataset admitted acknowledgement");
  i.finished = true;
  i.evidence["acknowledgement"] = i.acknowledge ? "dataset_admitted" : "none";
  i.evidence["after_ack_before_release"] = stats(i.session);
  i.delivery.release_payload();
  i.capture = sqav::Capture{};
  i.evidence["after_release"] = stats(i.session);
  i.evidence["final_checkpoint"] = checkpoint(i.session);
  i.evidence["checkpoint_persistence"] = {{"status", "not_selected"},
                                          {"durable", false}};
  i.delivery.reset();
  i.session.reset();
  i.source.reset();
  i.manifest = sqmv::Manifest{};
  i.flow = sqfv::Context{};
  return i.evidence;
}

Json retained_delivery_identity(const Json &selected, const Json &j) {
  keys(selected, {"reference", "delivery"});
  keys(selected.at("reference"), {"path", "expected_sha256", "pointer"});
  keys(j, {"protocol",
           "reference",
           "reference_file_sha256",
           "source",
           "choices",
           "origin",
           "view_reference",
           "sequence",
           "receipt",
           "before_offer",
           "taken",
           "baseline_checkpoint",
           "confirmed_retention",
           "recorded_snapshot_verification",
           "source_authorship",
           "exactly_once_claimed",
           "destination_commit_claimed",
           "acknowledgement",
           "after_ack_before_release",
           "after_release",
           "final_checkpoint",
           "checkpoint_persistence"});
  need(j.at("protocol") == "symphony.sbv.source-delivery.v1" &&
           j.at("origin") == "retained" &&
           j.at("reference") == selected.at("reference") &&
           j.at("choices") == selected.at("delivery"),
       "retained selection and delivery evidence disagree");
  hash_string(j.at("reference_file_sha256"));
  hash_string(j.at("reference").at("expected_sha256"));
  (void)local_path(j.at("reference").at("path"));
  const Json::json_pointer pointer(str(j.at("reference").at("pointer")));
  (void)pointer;
  const auto &choice = j.at("choices"), &s = j.at("source");
  keys(choice, {"profile", "view_id", "recipient_id", "recipient_interface",
                "first_sequence", "outstanding_byte_credit",
                "max_unacknowledged_batches", "processed_ack", "checkpoint"});
  need(choice.at("profile") == "retained_before_delivery" &&
           choice.at("checkpoint").is_null() &&
           (choice.at("processed_ack") == "none" ||
            choice.at("processed_ack") == "dataset_admitted"),
       "retained delivery lifecycle choice unsupported");
  for (auto field : {"view_id", "recipient_id", "recipient_interface"})
    need(!str(choice.at(field)).empty() && str(choice.at(field)).size() <= 4096,
         "retained delivery identity field invalid");
  keys(s, {"protocol", "owner_profile", "owner_versions", "profile_contract",
           "original", "capture", "metadata", "batch", "limits",
           "access_evidence", "retention", "delivery"});
  need(s.at("protocol") == retained_protocol &&
           s.at("owner_profile") == profile &&
           s.at("owner_versions") == versions() &&
           s.at("profile_contract") == source_owner_profile() &&
           s.at("delivery") == Json{{"status", "not_selected"}},
       "retained source profile mismatch");
  const Limits limits(s.at("limits"));
  sqmv::Manifest manifest;
  keys(s.at("metadata"),
       {"reference", "encoded_sha256", "bytes", "description"});
  owner_ok(sqmv::Manifest::create(
               manifest_description(s.at("metadata").at("description")),
               limits.metadata, manifest),
           "retained result manifest reconstruction");
  need(manifest_evidence(manifest) == s.at("metadata"),
       "retained result manifest identity mismatch");
  const auto &o = s.at("original"), &c = s.at("capture"),
             &r = s.at("retention");
  keys(o, {"source_path", "source_sha256", "dataset", "bytes", "record_count",
           "dbn_version", "native_encoding", "encoded_metadata_sha256"});
  hash_string(o.at("source_sha256"));
  hash_string(o.at("encoded_metadata_sha256"));
  (void)local_path(o.at("source_path"));
  need(!str(o.at("dataset")).empty() &&
           u64(o.at("bytes")) <= limits.dbn.max_file_bytes &&
           u64(o.at("record_count")) <= limits.dbn.max_records &&
           (o.at("dbn_version") == "1" || o.at("dbn_version") == "3") &&
           str(o.at("native_encoding")) ==
               db::encoding_for_version(
                   number<std::uint8_t>(o.at("dbn_version"))),
       "retained result original identity invalid");
  keys(c, {"reference", "source_reference", "encoded_sha256", "bytes",
           "description"});
  hash_string(c.at("encoded_sha256"));
  const auto capture_ref = str(c.at("reference"));
  need(capture_ref.starts_with("sqac1-sha256-"),
       "retained capture reference format invalid");
  hash_string(capture_ref.substr(std::string_view("sqac1-sha256-").size()));
  const auto description = capture_description(c.at("description"));
  need(description.source.dataset_id == str(o.at("dataset")) &&
           description.source.provider_ref == "databento" &&
           description.source.adapter_ref == db::adapter_id &&
           description.source.adapter_version == db::adapter_version &&
           description.source.native_schema_ref == db::native_schema &&
           description.source.native_encoding_ref ==
               str(o.at("native_encoding")) &&
           description.source_record_count &&
           *description.source_record_count == u64(o.at("record_count")),
       "retained capture/original declarations disagree");
  // This validates the declaration and its source reference only. Empty bytes
  // deliberately cannot reconstruct or authenticate the original capture.
  sqav::Capture declaration;
  owner_ok(sqav::Capture::create(description, {}, limits.capture, declaration),
           "retained capture declaration validation");
  need(capture_description(declaration.description()) == c.at("description") &&
           declaration.source_reference() == str(c.at("source_reference")),
       "retained capture declaration source identity mismatch");
  const auto capture_bytes = u64(c.at("bytes"));
  need(capture_bytes > u64(o.at("bytes")) &&
           capture_bytes <= limits.capture.max_capture_bytes &&
           capture_bytes <= limits.flow.max_payload_bytes,
       "retained capture byte counts invalid");
  keys(r, {"root", "options", "receipt", "snapshot", "append_status"});
  const auto configured = options(r.at("options"), limits);
  need(str(r.at("root")).size() <= 4096 && configured.partition.size() > 0 &&
           configured.partition.size() <= 4096,
       "retained result composed route incompatible");
  (void)local_path(r.at("root"));
  need(r.at("append_status") == "committed" ||
           r.at("append_status") == "duplicate",
       "retained result append unconfirmed");
  keys(s.at("batch"), {"descriptor", "content_id"});
  const auto &b = s.at("batch").at("descriptor");
  keys(b, {"binding", "partition", "source_binding", "source_position",
           "producer_generation", "batch_sequence", "record_count"});
  sqfv::Binding binding;
  owner_ok(manifest.binding(binding), "retained result manifest binding");
  const auto seq = u64(b.at("batch_sequence"));
  const auto expected_binding = descriptor(
      {binding, configured.partition, str(c.at("source_reference")),
       str(c.at("reference")), configured.producer_generation, seq, 1});
  need(b == expected_binding && u64(j.at("sequence")) == seq &&
           u64(choice.at("first_sequence")) == seq &&
           seq >= configured.first_sequence,
       "retained result batch binding mismatch");
  const auto &receipt_j = r.at("receipt");
  keys(receipt_j,
       {"store_generation", "producer_generation", "batch_sequence",
        "frame_bytes", "content_id", "frame_sha256", "commit_sha256"});
  hash_string(receipt_j.at("content_id"));
  hash_string(receipt_j.at("frame_sha256"));
  hash_string(receipt_j.at("commit_sha256"));
  need(j.at("receipt") == receipt_j &&
           receipt_j.at("store_generation") ==
               r.at("options").at("store_generation") &&
           receipt_j.at("producer_generation") == b.at("producer_generation") &&
           receipt_j.at("batch_sequence") == b.at("batch_sequence") &&
           receipt_j.at("content_id") == s.at("batch").at("content_id") &&
           u64(receipt_j.at("frame_bytes")) > capture_bytes &&
           u64(receipt_j.at("frame_bytes")) <= limits.store.max_frame_bytes &&
           u64(receipt_j.at("frame_bytes")) <= limits.flow.max_frame_bytes,
       "retained result receipt correspondence mismatch");
  const auto access_evidence = access(s.at("access_evidence"));
  const auto &md = manifest.description();
  need(md.dataset_id == description.source.dataset_id &&
           md.dataset_revision == description.source.dataset_revision &&
           md.schema_version == sqav::capture_schema &&
           md.layout_version == sqav::capture_layout &&
           md.access_scope == description.source.access_scope &&
           md.producer_ref == description.attribution_ref,
       "retained result metadata/capture correspondence mismatch");
  bool matched_access = false, matched_capture = false, matched_source = false;
  for (const auto &ev : md.evidence) {
    if (ev.role == sqmv::EvidenceRole::access)
      matched_access = ev.producer_ref == access_evidence.producer_ref &&
                       ev.evidence_ref == access_evidence.evidence_ref;
    if (ev.role == sqmv::EvidenceRole::lineage &&
        ev.evidence_ref == str(c.at("reference")) &&
        ev.producer_ref == description.attribution_ref)
      matched_capture = true;
    if (ev.role == sqmv::EvidenceRole::source &&
        ev.evidence_ref == str(c.at("source_reference")) &&
        ev.producer_ref == description.attribution_ref)
      matched_source = true;
  }
  need(matched_access && matched_capture && matched_source,
       "retained result manifest evidence correspondence mismatch");
  auto prefix = [&](const Json &p) {
    keys(p, {"committed_batches", "next_sequence", "store_bytes",
             "sequence_exhausted", "staged"});
    need(p.at("sequence_exhausted").is_boolean() && p.at("staged").is_boolean(),
         "retained snapshot flags invalid");
    const auto count = u64(p.at("committed_batches"));
    need(count > 0 && count <= configured.limits.max_batches &&
             count - 1 <= UINT64_MAX - configured.first_sequence,
         "retained snapshot count invalid");
    const auto last = configured.first_sequence + count - 1;
    need(seq <= last &&
             u64(p.at("next_sequence")) ==
                 (last == UINT64_MAX ? last : last + 1) &&
             p.at("sequence_exhausted") == (last == UINT64_MAX) &&
             u64(p.at("store_bytes")) > u64(receipt_j.at("frame_bytes")) &&
             u64(p.at("store_bytes")) <= configured.limits.max_store_bytes,
         "retained snapshot prefix/bytes invalid");
    return count;
  };
  const auto recorded_count = prefix(r.at("snapshot")),
             confirmed_count = prefix(j.at("confirmed_retention"));
  need(r.at("snapshot").at("staged") == false &&
           recorded_count <= confirmed_count &&
           u64(r.at("snapshot").at("store_bytes")) <=
               u64(j.at("confirmed_retention").at("store_bytes")),
       "retained result historical/current snapshots disagree");
  const bool exact = recorded_count == confirmed_count &&
                     j.at("confirmed_retention").at("staged") == false;
  need(j.at("recorded_snapshot_verification") ==
               (exact ? "exact_current_prefix"
                      : "historical_within_current_prefix") &&
           (!exact || r.at("snapshot") == j.at("confirmed_retention")),
       "retained snapshot verification claim mismatch");
  const bool ack = choice.at("processed_ack") == "dataset_admitted",
             exhausted = seq == UINT64_MAX;
  const auto next = exhausted ? seq : seq + 1;
  need(u64(choice.at("outstanding_byte_credit")) >= capture_bytes &&
           u64(choice.at("outstanding_byte_credit")) <= (64U << 20) &&
           u64(choice.at("max_unacknowledged_batches")) >= 1 &&
           u64(choice.at("max_unacknowledged_batches")) <= 65536,
       "retained delivery resources invalid");
  const Json before{
      {"next_offer_sequence", dec(seq)}, {"next_processed_sequence", dec(seq)},
      {"outstanding_bytes", "0"},        {"unacknowledged_batches", "0"},
      {"pending_deliveries", "0"},       {"offer_exhausted", false},
      {"processed_exhausted", false}};
  auto taken = before;
  taken["next_offer_sequence"] = dec(next);
  taken["offer_exhausted"] = exhausted;
  taken["outstanding_bytes"] = dec(capture_bytes);
  taken["unacknowledged_batches"] = "1";
  auto after_ack = taken;
  if (ack) {
    after_ack["next_processed_sequence"] = dec(next);
    after_ack["processed_exhausted"] = exhausted;
    after_ack["unacknowledged_batches"] = "0";
  }
  auto released = after_ack;
  released["outstanding_bytes"] = "0";
  need(j.at("before_offer") == before && j.at("taken") == taken &&
           j.at("after_ack_before_release") == after_ack &&
           j.at("after_release") == released &&
           j.at("acknowledgement") == choice.at("processed_ack"),
       "retained delivery acknowledgement/lease state mismatch");
  const auto view_ref = str(j.at("view_reference"));
  need(view_ref.starts_with("sqdv1-sha256-"),
       "retained delivery view reference required");
  hash_string(view_ref.substr(std::string_view("sqdv1-sha256-").size()));
  need(j.at("baseline_checkpoint") ==
               Json{{"view_reference", j.at("view_reference")},
                    {"next_sequence", dec(seq)},
                    {"sequence_exhausted", false}} &&
           j.at("final_checkpoint") ==
               Json{{"view_reference", j.at("view_reference")},
                    {"next_sequence", dec(ack ? next : seq)},
                    {"sequence_exhausted", ack && exhausted}} &&
           j.at("checkpoint_persistence") ==
               Json{{"status", "not_selected"}, {"durable", false}} &&
           j.at("source_authorship") == "not_verified" &&
           j.at("exactly_once_claimed") == false &&
           j.at("destination_commit_claimed") == false,
       "retained delivery checkpoint or authority claim mismatch");
  return {{"source_path", o.at("source_path")},
          {"source_sha256", o.at("source_sha256")},
          {"dataset", o.at("dataset")}};
}

Json source_export(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "retained_source", "kind", "output_path", "receipt_path",
           "extensions"});
  need(p.at("protocol") == "symphony.sbv.source-export-input.v1" &&
           p.at("extensions").is_object(),
       "source-export protocol/extensions required");
  const auto kind = str(p.at("kind"));
  need(kind == "original" || kind == "capture" || kind == "manifest",
       "unsupported source export kind");
  const auto output = local_path(p.at("output_path")),
             result_path = local_path(p.at("receipt_path"));
  need(output != result_path,
       "binary output and result receipt must be different paths");
  require_new_file(output);
  require_new_file(result_path);
  keys(p.at("retained_source"), {"reference", "delivery"});
  need(p.at("retained_source").at("delivery").is_object() &&
           p.at("retained_source").at("delivery").contains("processed_ack"),
       "explicit source export delivery choice required");
  need(p.at("retained_source").at("delivery").at("processed_ack") == "none",
       "source-export requires explicit processed_ack:none; dataset admission "
       "is a different boundary");
  auto result = base("source_export");
  result["sections"]["choices"] = section(p);
  validate_result(seal_result(result));
  RetainedOriginal retained(p.at("retained_source"), end);
  auto &i = *retained.impl_;
  const auto &source = i.evidence.at("source");
  const auto root = str(source.at("retention").at("root"));
  outside_store(output, root);
  outside_store(result_path, root);
  const auto payload = kind == "original"  ? i.capture.original()
                       : kind == "capture" ? i.capture.encoded()
                                           : i.manifest.encoded();
  Json binary{{"protocol", "symphony.sbv.source-binary-receipt.v1"},
              {"path", output},
              {"bytes", dec(payload.size())},
              {"sha256", e::sha256_hex(payload)},
              {"kind", kind},
              {"reference", p.at("retained_source").at("reference")},
              {"capture_reference", i.capture.reference()},
              {"metadata_reference", i.manifest.reference()},
              {"batch_content_id", source.at("batch").at("content_id")},
              {"retention_receipt", source.at("retention").at("receipt")}};
  bool binary_confirmed = false;
  try {
    // create_file stages, fsyncs, publishes without replacement, then fsyncs
    // the parent directory. The selected capture profile fits its existing
    // bound.
    create_file(output,
                std::string(reinterpret_cast<const char *>(payload.data()),
                            payload.size()),
                end);
    binary_confirmed = true;
    result["sections"]["binary"] = section(binary);
    result["sections"]["delivery"] = section(retained.after_admission());
    result["sections"]["summary"] = section({{"kind", kind},
                                             {"path", output},
                                             {"bytes", binary.at("bytes")},
                                             {"sha256", binary.at("sha256")},
                                             {"processed_ack", "none"}});
    result["sections"]["provenance"] =
        section({{"engine_version", version},
                 {"source_reference", p.at("retained_source").at("reference")},
                 {"source_authorship", "not_verified"},
                 {"transformation", "none"},
                 {"provider_calls", "0"}});
    auto destination = p;
    destination["output_path"] = result_path;
    outside_store(result_path, root);
    return persist(std::move(result), destination, "source-export", end);
  } catch (const std::exception &) {
    return {{"protocol", "symphony.sbv.source-export.v1"},
            {"status", "recovery_required"},
            {"code", "sbv.source_export_unpublished"},
            {"stage",
             binary_confirmed ? "receipt_publication" : "binary_publication"},
            {"request_sha256", e::sha256_hex(p.dump())},
            {"output_path", output},
            {"receipt_path", result_path},
            {"automatic_retry", false},
            {"rollback_performed", false},
            {"recovery",
             {{"binary_publication_confirmed", binary_confirmed},
              {"expected_binary", binary}}}};
  }
}
} // namespace symphony::sbv::detail
