#include <symphony/sqdv/delivery.hpp>
#include "hook.hpp"
#include <array>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <future>
#include <thread>
#include <unistd.h>
using namespace symphony;
namespace {
void check(bool p,const char* why){if(!p){std::fprintf(stderr,"preview progress: %s\n",why);std::abort();}}
void byte(int fd,bool send){char c=1;check((send ? ::write(fd,&c,1) : ::read(fd,&c,1))==1,"pipe");}
}
void preview_progress() {
  ::alarm(20);
  char path[]="/private/tmp/sqv20-preview-XXXXXX";check(::mkdtemp(path),"root");
  int notice[2],resume[2];check(::pipe(notice)==0&&::pipe(resume)==0,"pipes");
  sqpv::testing::pause_at(1,notice[1],resume[0]);
  {
    sqfv::Context context;check(sqfv::Context::create({4096,8192,4096,1<<20,4},context)==sqfv::Status::ok,"context");
    sqmv::Manifest manifest;
    check(sqmv::Manifest::create({"dataset","revision","schema","layout","private","producer",{{sqmv::EvidenceRole::schema,"p","s"},{sqmv::EvidenceRole::layout,"p","l"},{sqmv::EvidenceRole::access,"p","a"}}},{65536,4096,128},manifest)==sqmv::Status::ok,"manifest");
    sqpv::Options options;options.partition="part";options.producer_generation[0]=1;options.store_generation[0]=2;options.limits={8192,1<<20,8};
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create_async(path,manifest,options,context,{2,8192},source)==sqdv::Status::ok,"source");
    sqdv::Config config{"view","preview","interface",options.partition,options.producer_generation,0,sqdv::Profile::asynchronous_retention};
    sqdv::Session preview,reader;
    check(sqdv::Session::create(context,manifest,config,{1024,2},&source,nullptr,preview)==sqdv::Status::ok,"preview");
    config.recipient_id="reader";check(sqdv::Session::create(context,manifest,config,{1024,2},&source,nullptr,reader)==sqdv::Status::ok,"reader");
    sqfv::Descriptor descriptor;check(manifest.binding(descriptor.binding)==sqmv::Status::ok,"binding");
    descriptor.partition=options.partition;descriptor.producer_generation=options.producer_generation;descriptor.record_count=1;
    sqfv::Batch batch;const std::array<std::uint8_t,1> bytes{42};check(context.prepare_copy(descriptor,bytes,batch)==sqfv::Status::ok,"batch");
    sqdv::QueuedBatch proof;check(source.enqueue(batch,proof)==sqdv::Status::ok,"enqueue");byte(notice[0],false);
    // Worker is paused inside the real Store append with its disk mutex held.
    auto retained=std::async(std::launch::async,[&]{return reader.offer_next(context);});
    auto drained=std::async(std::launch::async,[&]{return source.finish_retention();});
    check(retained.wait_for(std::chrono::milliseconds(50))==std::future_status::timeout,"read waits for actual disk work");
    check(drained.wait_for(std::chrono::milliseconds(50))==std::future_status::timeout,"drain waits for actual disk work");
    auto live=std::async(std::launch::async,[&]{
      sqdv::Delivery delivery;
      if(preview.offer_preview(proof)!=sqdv::Status::ok || preview.take(delivery)!=sqdv::Status::ok) return false;
      return delivery.retention_receipt()==nullptr && delivery.payload()[0]==42;
    });
    const bool progressed=live.wait_for(std::chrono::seconds(2))==std::future_status::ready;
    // Always release the writer before inspecting an assertion, even for a regression.
    byte(resume[1],true);
    check(live.get()&&progressed,"preview progresses while read and drain wait");
    check(retained.get()==sqdv::Status::ok && drained.get()==sqdv::Status::ok,"disk work subsequently completes");
  }
  sqpv::testing::pause_at(0,-1,-1);
  for(int fd:{notice[0],notice[1],resume[0],resume[1]})::close(fd);
  std::filesystem::remove_all(path);
  std::puts("Preview progress passed with paused disk write, concurrent retained read and drain");
}

int main() { preview_progress(); }
