#include "../src/provider.hpp"
#include <atomic>
#include <chrono>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <future>
#include <iostream>
#include <iterator>
#include <source_location>
#include <thread>
#include <unistd.h>
#include <vector>
namespace e = symphony::knowledge::engine;
namespace d = symphony::sbv::detail;
namespace db = symphony::sqav::databento;
using J = symphony::sbv::Json;
namespace {
unsigned checks = 0;
constexpr auto never = e::no_deadline;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("provider assertion line " +
                             std::to_string(at.line()));
}
template <class F>
void refused(F &&fn,
             std::source_location at = std::source_location::current()) {
  bool failed = false;
  try {
    fn();
  } catch (const std::exception &) {
    failed = true;
  }
  check(failed, at);
}
std::string read(const std::string &path) {
  std::ifstream f(path, std::ios::binary);
  if (!f)
    throw std::runtime_error("fixture input missing");
  return {std::istreambuf_iterator<char>(f), std::istreambuf_iterator<char>()};
}
J selected(const std::string &path, const std::string &role = "strategy",
           const J &parameters = J::object(),
           const std::string &mode = "serialized_instance") {
  return {
      {"protocol", "symphony.sbv.native-provider-selection.v1"},
      {"library",
       {{"path", path}, {"expected_sha256", e::sha256_hex(read(path))}}},
      {"id", "sbv-test-provider"},
      {"version", "1"},
      {"role", role},
      {"input_profile", "symphony.sbv.provider-databento-mbo-event.v1"},
      {"concurrency", mode},
      {"parameters", parameters},
      {"dependencies",
       {{"capture", "uncaptured"},
        {"description", "Independent fixture uses the platform C++ runtime"},
        {"artifacts", J::array()}}},
      {"extensions", J::object()}};
}
std::vector<db::Mbo> events() {
  std::vector<db::Mbo> out(3);
  for (std::size_t n = 0; n < out.size(); ++n) {
    auto &v = out[n];
    v.publisher_id = static_cast<std::uint16_t>(57000 + n);
    v.instrument_id = static_cast<std::uint32_t>(4000000000ULL + n);
    v.ts_event = 1700000000000000100ULL + n;
    v.order_id = 18000000000000000000ULL + n;
    v.price = -120000000001LL - static_cast<std::int64_t>(n);
    v.size = static_cast<std::uint32_t>(3000000000ULL + n);
    v.flags = static_cast<std::uint8_t>(200 + n);
    v.channel_id = static_cast<std::uint8_t>(99 + n);
    v.action = 'T';
    v.side = 'A';
    v.ts_recv = 1700000000000000200ULL + n;
    v.ts_in_delta = -1700000000 + static_cast<std::int32_t>(n);
    v.sequence = static_cast<std::uint32_t>(2000000000ULL + n);
  }
  out[2].ts_out = UINT64_MAX;
  return out;
}
void fields(const J &ext, const db::Mbo &v, std::size_t ordinal) {
  const auto eq = [&](const char *name, auto value) {
    check(ext.at(std::string("last_") + name) == std::to_string(value));
  };
  eq("source_ordinal", ordinal);
  eq("publisher_id", v.publisher_id);
  eq("instrument_id", v.instrument_id);
  eq("ts_event", v.ts_event);
  eq("order_id", v.order_id);
  eq("price_nanos", v.price);
  eq("size", v.size);
  eq("flags", v.flags);
  eq("channel_id", v.channel_id);
  eq("action", v.action);
  eq("side", v.side);
  eq("ts_recv", v.ts_recv);
  eq("ts_in_delta", v.ts_in_delta);
  eq("sequence", v.sequence);
  eq("has_ts_out", v.ts_out ? 1 : 0);
  eq("ts_out", v.ts_out.value_or(0));
}
J signal() {
  return {{"signal", {{"signal_id", "signal/0~"}}},
          {"source_sha256", std::string(64, 'a')}};
}
J inspect(const J &selection) {
  return d::provider_inspect(
      {{"protocol", "symphony.sbv.provider-inspect-input.v1"},
       {"provider", selection},
       {"extensions", {{"caller", "preserved"}}}},
      never);
}
} // namespace
int main(int argc, char **argv) try {
  check(argc == 6);
  const auto fixture = std::filesystem::canonical(argv[1]).string();
#if defined(__APPLE__)
  char temp[] = "/private/tmp/sbv-provider-test-XXXXXX";
#else
  char temp[] = "/tmp/sbv-provider-test-XXXXXX";
#endif
  check(::mkdtemp(temp) != nullptr);
  const std::string root = temp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  const auto source = events();
  auto selection = selected(
      fixture, "strategy", {{"emit_every", "2"}, {"emissions_per_event", "2"}});
  const auto inspection = inspect(selection);
  check(inspection.at("protocol") == "symphony.sbv.provider-inspect.v1");
  check(inspection.at("extensions").at("caller") == "preserved");
  const auto evidence = inspection.at("provider");
  check(evidence.at("selection_sha256") == e::sha256_hex(selection.dump()));
  check(evidence.at("library").at("expected_sha256") ==
        selection.at("library").at("expected_sha256"));
  check(evidence.at("library").at("bytes") ==
        std::to_string(read(fixture).size()));
  check(evidence.at("descriptor_sha256") ==
        e::sha256_hex(evidence.at("descriptor_json").get<std::string>()));
  check(J::parse(evidence.at("descriptor_json").get<std::string>())
            .at("config_schema")
            .at("maxProperties") == 1000);
  check(evidence.at("loading").at("profile") == "private_verified_copy");
  check(evidence.at("loading").at("isolation") == false);
  check(evidence ==
        inspect(selection).at("provider")); // No ephemeral stage identity.
  // Invalid portable configuration must fail before any source open, library
  // initializer or provider callback. The deliberately missing path makes
  // this ordering independently observable through the rejection reason.
  for (const auto *field : {"parameters", "extensions"}) {
    for (const auto &number : std::vector<J>{J(-1), J(UINT64_MAX), J(0.5)}) {
      auto nonportable = selection;
      nonportable["library"]["path"] = root + "/must-not-open";
      nonportable[field]["user-defined"] = {
          {"nested", J::array({nullptr, true, {{"value", number}}})}};
      std::string reason;
      try {
        d::NativeProvider provider(nonportable, never);
      } catch (const std::exception &error) {
        reason = error.what();
      }
      check(
          reason.find(
              "provider parameters/extensions require exact numeric strings") !=
          std::string::npos);
    }
  }
  auto portable = selection;
  portable["parameters"]["user-defined"] = {
      {"nested", J::array({nullptr, true, "0.5", "18446744073709551615"})}};
  portable["extensions"]["user-defined"] = {
      {"nested", J::array({false, "-1"})}};
  check(inspect(portable).at("provider").at("selection_sha256") ==
        e::sha256_hex(portable.dump()));
  d::validate_native_provider_evidence(selection, evidence);
  auto contradictory = evidence;
  auto raw_descriptor =
      J::parse(evidence.at("descriptor_json").get<std::string>());
  raw_descriptor["id"] = "contradictory-provider";
  contradictory["descriptor_json"] = raw_descriptor.dump();
  contradictory["descriptor_sha256"] = e::sha256_hex(raw_descriptor.dump());
  refused(
      [&] { d::validate_native_provider_evidence(selection, contradictory); });
  contradictory = evidence;
  contradictory["abi_version"] = "2";
  refused(
      [&] { d::validate_native_provider_evidence(selection, contradictory); });
  contradictory = evidence;
  contradictory["library"]["bytes"] = "00";
  refused(
      [&] { d::validate_native_provider_evidence(selection, contradictory); });
  contradictory = evidence;
  contradictory["loading"]["isolation"] = true;
  refused(
      [&] { d::validate_native_provider_evidence(selection, contradictory); });
  contradictory = evidence;
  contradictory["dependencies"]["description"] = "different declaration";
  refused(
      [&] { d::validate_native_provider_evidence(selection, contradictory); });
  // Self-consistency admission intentionally does not require the library to
  // remain at its historical path and does not prove author authenticity.
  auto absent_selection = selection;
  absent_selection["library"]["path"] =
      root + "/historical-library-no-longer-present";
  auto absent_evidence = evidence;
  absent_evidence["library"]["path"] = absent_selection["library"]["path"];
  absent_evidence["selection_sha256"] = e::sha256_hex(absent_selection.dump());
  d::validate_native_provider_evidence(absent_selection, absent_evidence);
  std::unique_ptr<d::NativeProviderInstance> surviving;
  {
    d::NativeProvider provider(selection, never);
    check(provider.evidence() == evidence);
    check(provider.descriptor().at("id") == "sbv-test-provider");
    surviving = provider.create(never);
  }
  // Instance holds its stage and library after the wrapper goes out of scope.
  refused([&] { surviving->event(source, 1, never); });
  auto first = surviving->event(source, 0, never);
  check(first.status == SBV_PROVIDER_OK && first.value.size() == 2 &&
        first.error.is_null());
  check(first.value.at(0).at("signal_id") == "provider-0-0");
  check(first.value.at(1).at("signal_id") == "provider-0-1");
  check(first.value.at(0).at("anchor_price_nanos") == "-120000000001");
  check(first.value.at(0).at("context_reference") ==
        "fixture://mbo/18000000000000000000/200/99");
  auto second = surviving->event(source, 1, never);
  check(second.status == SBV_PROVIDER_OK && second.value.is_null());
  check(surviving->event(source, 2, never).value.size() == 2);
  const auto finished = surviving->finish(never);
  check(finished.status == SBV_PROVIDER_OK);
  check(finished.value.at("extensions").at("events") == "3");
  fields(finished.value.at("extensions"), source.back(), 2);
  refused([&] { surviving->event(source, 0, never); });
  refused([&] { surviving->finish(never); });
  surviving.reset();
  {
    d::NativeProvider provider(selected(fixture), never);
    auto instance = provider.create(never);
    instance->event(source, 0, never);
    fields(instance->finish(never).value.at("extensions"), source.front(), 0);
  }
  const auto expected_model = [&](int variant) {
    const auto p = selected(fixture, "model",
                            {{"model_variant", std::to_string(variant)}});
    d::NativeProvider provider(p, never);
    auto instance = provider.create(never);
    auto result = instance->evaluate(source, 0, source.size(), signal(),
                                     J::object(), source.back().ts_recv, never);
    check(result.status == SBV_PROVIDER_OK);
    check(result.value.at("signal_id") == "signal/0~");
    check(result.value.at("support").at(0).at("price_nanos") ==
          std::to_string(source[0].price - variant));
    check(result.value.at("support").at(1).at("price_nanos") ==
          std::to_string(source[0].price + variant));
    check(
        result.value.at("execution_probability").at("value").at("numerator") ==
        (variant == 1 ? "1" : "3"));
    check(instance
              ->evaluate(source, 0, 1, signal(), J::object(), source[0].ts_recv,
                         never)
              .value == result.value);
    refused([&] {
      instance->evaluate(source, 0, 0, signal(), J::object(), 0, never);
    });
    refused([&] {
      instance->evaluate(source, 0, 4, signal(), J::object(), 0, never);
    });
    refused([&] { instance->event(source, 0, never); });
    refused([&] { instance->finish(never); });
    return result.value;
  };
  const auto model1 = expected_model(1), model2 = expected_model(2);
  check(model1 != model2);
  for (const auto *concurrency :
       {"serialized_instance", "shared_reentrant_instance",
        "per_worker_instances"}) {
    d::NativeProvider provider(
        selected(fixture, "model", J::object(), concurrency), never);
    auto shared = provider.create(never);
    std::vector<std::future<J>> jobs;
    for (unsigned n = 0; n < 6; ++n)
      jobs.push_back(std::async(std::launch::async, [&] {
        auto own = std::string(concurrency) == "per_worker_instances"
                       ? provider.create(never)
                       : nullptr;
        auto *instance = own ? own.get() : shared.get();
        auto reply =
            instance->evaluate(source, 0, source.size(), signal(), J::object(),
                               source.back().ts_recv, never);
        if (reply.status != SBV_PROVIDER_OK)
          throw std::runtime_error("parallel fixture failure");
        return reply.value;
      }));
    for (auto &job : jobs)
      check(job.get() == model1);
  }
  {
    d::NativeProvider provider(
        selected(fixture, "model", {{"mode", "unavailable"}}), never);
    auto reply = provider.create(never)->evaluate(
        source, 0, 3, signal(), J::object(), source.back().ts_recv, never);
    check(reply.status == SBV_PROVIDER_UNAVAILABLE && reply.value.is_null());
    check(reply.error.at("message") == "selected model unavailable");
  }
  // Refuse selection/table mismatches before exposing a usable provider.
  auto reject_selection = [&](const J &p) {
    refused([&] { d::NativeProvider provider(p, never); });
  };
  auto wrong = selection;
  wrong["library"]["expected_sha256"] = std::string(64, '0');
  reject_selection(wrong);
  wrong = selection;
  wrong["id"] = "wrong";
  reject_selection(wrong);
  wrong = selection;
  wrong["version"] = "2";
  reject_selection(wrong);
  wrong = selection;
  wrong["role"] = "both";
  reject_selection(wrong);
  wrong = selection;
  wrong["input_profile"] = "partial";
  reject_selection(wrong);
  wrong = selection;
  wrong["concurrency"] = "shared_reentrant_instance";
  reject_selection(wrong);
  wrong = selection;
  wrong.erase("dependencies");
  reject_selection(wrong);
  wrong = selection;
  wrong["library"]["path"] = root + "/missing";
  reject_selection(wrong);
  for (int n = 2; n < argc; ++n)
    reject_selection(selected(std::filesystem::canonical(argv[n]).string()));
  std::filesystem::create_symlink(fixture, root + "/symlink");
  wrong = selection;
  wrong["library"]["path"] = root + "/symlink";
  reject_selection(wrong);
  std::filesystem::create_directory_symlink(
      std::filesystem::path(fixture).parent_path(), root + "/parentlink");
  wrong = selection;
  wrong["library"]["path"] = root + "/parentlink/" +
                             std::filesystem::path(fixture).filename().string();
  reject_selection(wrong);
  // Independent digest oracle across more than two streaming buffer boundaries.
  const auto dependency_path = root + "/declared-dependency";
  std::string dependency(131071, '\0');
  for (std::size_t n = 0; n < dependency.size(); ++n)
    dependency[n] = static_cast<char>(n * 73U);
  {
    std::ofstream output(dependency_path, std::ios::binary);
    output.write(dependency.data(), dependency.size());
  }
  auto declared = selection;
  declared["dependencies"]["capture"] = "declared_artifacts";
  declared["dependencies"]["artifacts"] =
      J::array({{{"path", dependency_path},
                 {"expected_sha256", e::sha256_hex(dependency)}}});
  check(inspect(declared)
            .at("provider")
            .at("dependencies")
            .at("artifacts")
            .at(0)
            .at("bytes") == "131071");
  wrong = declared;
  wrong["dependencies"]["artifacts"][0]["expected_sha256"] =
      std::string(64, '0');
  reject_selection(wrong);
  wrong = declared;
  wrong["dependencies"]["artifacts"].push_back(
      wrong["dependencies"]["artifacts"].at(0));
  reject_selection(wrong);
  wrong = declared;
  wrong["dependencies"]["capture"] = "uncaptured";
  reject_selection(wrong);
  // Every published buffer is released on parse/status/ABI validation failures.
  for (const auto *mode :
       {"malformed_json", "bad_buffer_abi", "zero_size_owned", "unknown_status",
        "owned_error"}) {
    d::NativeProvider provider(
        selected(fixture, "strategy",
                 {{"mode", mode}, {"report_counters", "true"}}),
        never);
    auto instance = provider.create(never);
    refused([&] { instance->event(source, 0, never); });
    auto ext = instance->finish(never).value.at("extensions");
    check(ext.at("releases") ==
          (std::string(mode) == "owned_error" ? "3" : "2"));
  }
  {
    d::NativeProvider provider(
        selected(fixture, "strategy",
                 {{"mode", "provider_failure"}, {"report_counters", "true"}}),
        never);
    auto instance = provider.create(never);
    auto reply = instance->event(source, 0, never);
    check(reply.status == SBV_PROVIDER_FAILED && reply.value.is_null());
    check(reply.error.at("code") == "fixture");
    check(instance->finish(never).value.at("extensions").at("releases") == "2");
  }
  {
    d::NativeProvider provider(
        selected(fixture, "strategy", {{"mode", "create_failure"}}), never);
    std::string error;
    try {
      provider.create(never);
    } catch (const std::exception &ex) {
      error = ex.what();
    }
    check(error.find("selected create failure") != std::string::npos);
  }
  {
    d::NativeProvider provider(selected(fixture, "strategy",
                                        {{"mode", "create_failure_owned"},
                                         {"report_counters", "true"}}),
                               never);
    refused([&] { provider.create(never); });
    auto instance = provider.create(never);
    const auto ext = instance->finish(never).value.at("extensions");
    check(ext.at("creates") == "2" && ext.at("destroys") == "1" &&
          ext.at("releases") == "2");
  }
  {
    d::NativeProvider provider(
        selected(fixture, "strategy", {{"fail_after", "2"}}), never);
    auto instance = provider.create(never);
    check(instance->event(source, 0, never).status == SBV_PROVIDER_OK);
    check(instance->event(source, 1, never).status == SBV_PROVIDER_OK);
    check(instance->event(source, 2, never).status == SBV_PROVIDER_FAILED);
    check(instance->finish(never).value.at("extensions").at("events") == "2");
  }
  {
    d::NativeProvider provider(selected(fixture), never);
    std::atomic_bool cancelled{true};
    refused([&] { provider.create(never, &cancelled); });
    auto instance = provider.create(never);
    const auto reply = instance->event(source, 0, never, &cancelled);
    check(reply.status == SBV_PROVIDER_CANCELLED &&
          reply.error.at("code") == "host_cancelled");
    // Pre-cancel does not invoke the provider or advance ordinal.
    cancelled = false;
    check(instance->event(source, 0, never, &cancelled).status ==
          SBV_PROVIDER_OK);
    check(instance->finish(never).value.at("extensions").at("events") == "1");
  }
  {
    d::NativeProvider provider(
        selected(fixture, "strategy", {{"mode", "cancel_poll"}}), never);
    auto instance = provider.create(never);
    std::atomic_bool cancelled{false};
    std::jthread trigger([&] {
      std::this_thread::sleep_for(std::chrono::milliseconds(20));
      cancelled = true;
    });
    const auto reply = instance->event(source, 0, never, &cancelled);
    check(reply.status == SBV_PROVIDER_CANCELLED);
    check(reply.error.at("message") == "cooperative cancellation observed");
  }
  {
    d::NativeProvider provider(
        selected(fixture, "strategy", {{"mode", "cancel_poll"}}), never);
    auto instance = provider.create(never);
    refused([&] { instance->event(source, 0, e::unix_time_ms() + 20); });
    refused([&] { provider.create(e::unix_time_ms() - 1); });
  }
  std::cout << "provider runtime: " << checks << " checks passed\n";
  return 0;
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
