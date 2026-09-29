#pragma once
#include <algorithm>
#include <cstdlib>
#include <filesystem>
#include <symphony/sqav/capture.hpp>
#include <symphony/sqdv/checkpoint.hpp>
#include <unistd.h>
namespace source_composition {
inline void require(bool b) {
  if (!b)
    std::abort();
}
struct Root {
  std::string path;
  Root() {
    char p[] = "/private/tmp/sqv-source-retain-XXXXXX";
    auto value = ::mkdtemp(p);
    require(value);
    path = value;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
inline void verify(const symphony::sqav::Capture &capture) {
  using namespace symphony;
  Root root;
  sqmv::Manifest manifest;
  require(capture.metadata(
              false, {sqmv::EvidenceRole::access, "fixture", "private:fixture"},
              {65536, 4096, 128}, manifest) == sqav::Status::ok);
  sqfv::Context flow;
  require(sqfv::Context::create({262144, 270336, 4096, 1U << 22, 8}, flow) ==
          sqfv::Status::ok);
  sqav::Position pos{"source", {}, 1};
  pos.producer_generation[0] = 1;
  sqfv::Batch batch;
  require(capture.prepare(flow, manifest, pos, batch) == sqav::Status::ok);
  sqpv::Options options{
      pos.partition, pos.producer_generation, {}, 1, {270336, 1U << 22, 8}};
  options.store_generation[0] = 2;
  sqdv::Config config{"source-view",
                      "consumer",
                      "capture",
                      pos.partition,
                      pos.producer_generation,
                      1,
                      sqdv::Profile::asynchronous_retention};
  {
    sqdv::RetainedSource store;
    require(sqdv::RetainedSource::create_async(root.path, manifest, options,
                                               flow, {2, 540672},
                                               store) == sqdv::Status::ok);
    sqdv::QueuedBatch queued;
    require(store.enqueue(batch, queued) == sqdv::Status::ok);
    require(store.finish_retention() == sqdv::Status::ok);
  }
  sqdv::RetainedSource store;
  require(sqdv::RetainedSource::open_async(root.path, manifest, options, flow,
                                           {2, 540672},
                                           store) == sqdv::Status::ok);
  sqdv::Session replay;
  require(sqdv::Session::create(flow, manifest, config, {262144, 2}, &store,
                                nullptr, replay) == sqdv::Status::ok);
  require(replay.offer_next(flow) == sqdv::Status::ok);
  sqdv::Delivery delivery;
  require(replay.take(delivery) == sqdv::Status::ok);
  sqav::Capture restored;
  require(sqav::Capture::from_delivery(
              delivery.payload(), delivery.descriptor(), manifest,
              {262144, 16384, 4096}, restored) == sqav::Status::ok);
  require(restored.reference() == capture.reference() &&
          restored.description().coverage == capture.description().coverage &&
          std::ranges::equal(restored.original(), capture.original()));
  require(replay.acknowledge_processed(delivery) == sqdv::Status::ok);
}
} // namespace source_composition
