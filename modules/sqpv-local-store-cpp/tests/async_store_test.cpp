#include <symphony/sqpv/async_store.hpp>
#include "hook.hpp"
#include <array>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <signal.h>
#include <sys/wait.h>
#include <unistd.h>
using namespace symphony;
namespace {
void check(bool p, const char* why) { if (!p) { std::fprintf(stderr,"async retention: %s\n",why); std::abort(); } }
struct Root { std::string path;
  Root() { char n[]="/private/tmp/sqv20-async-XXXXXX"; auto p=::mkdtemp(n); check(p,"root"); path=p; }
  ~Root() { std::filesystem::remove_all(path); }
};
struct Fixture {
  sqfv::Context context;
  sqmv::Manifest manifest;
  sqpv::Options options;
  Fixture() {
    check(sqfv::Context::create({4096,8192,4096,1<<20,4},context)==sqfv::Status::ok,"context");
    sqmv::Description d{"dataset:fixture","revision:1","schema:bytes","layout:bytes","private","producer",
      {{sqmv::EvidenceRole::schema,"producer","schema"},{sqmv::EvidenceRole::layout,"producer","layout"},{sqmv::EvidenceRole::access,"producer","grant"}}};
    check(sqmv::Manifest::create(d,{65536,4096,128},manifest)==sqmv::Status::ok,"manifest");
    options.partition="partition:a";options.producer_generation[0]=1;options.store_generation[0]=2;
    options.first_sequence=7;options.limits={8192,1<<20,8};
  }
  sqfv::Batch batch(std::uint64_t n, std::uint8_t value=42) {
    sqfv::Descriptor d;check(manifest.binding(d.binding)==sqmv::Status::ok,"binding");
    d.partition=options.partition;d.producer_generation=options.producer_generation;d.batch_sequence=n;d.record_count=1;
    sqfv::Batch out; const std::array bytes{value};
    check(context.prepare_copy(d,bytes,out)==sqfv::Status::ok,"batch");return out;
  }
};
void signal_byte(int fd) { char v=1;check(::write(fd,&v,1)==1,"signal"); }
void await_byte(int fd) { char v=0;check(::read(fd,&v,1)==1,"wait signal"); }
void bounded_queue() {
  Root root; Fixture f;int notice[2],resume[2];check(::pipe(notice)==0&&::pipe(resume)==0,"pipes");
  sqpv::testing::pause_at(1,notice[1],resume[0]);
  sqpv::AsyncStore store;
  check(sqpv::AsyncStore::create(root.path,f.manifest,f.options,f.context,{2,8192},store)==sqpv::Status::ok,"create");
  auto a=f.batch(7),b=f.batch(8),c=f.batch(9),conflict=f.batch(7,43);
  check(store.submit(a)==sqpv::Status::ok,"first admission");await_byte(notice[0]);
  check(store.submit(a)==sqpv::Status::duplicate,"active exact retry");
  check(store.submit(conflict)==sqpv::Status::conflict,"active conflict");
  check(store.submit(b)==sqpv::Status::ok,"second admission");
  check(store.submit(c)==sqpv::Status::busy,"active write included in queue bound");
  sqpv::AsyncSnapshot status;
  check(store.snapshot(status)==sqpv::Status::ok && status.pending_batches==2 && status.confirmed.committed_batches==0 && status.next_admission_sequence==9,"queued progress distinct from confirmed");
  // Last caller context and producer handles disappear while writer is paused.
  a={};b={};c={};conflict={};f.context={};
  signal_byte(resume[1]);signal_byte(resume[1]);
  check(store.finish()==sqpv::Status::ok,"drain");
  check(store.snapshot(status)==sqpv::Status::ok && status.confirmed.committed_batches==2 && !status.pending_batches && !status.pending_frame_bytes,"pins drain and watermark advances");
  store.reset();sqpv::testing::pause_at(0,-1,-1);
  for(int fd:{notice[0],notice[1],resume[0],resume[1]})::close(fd);
}
void uncertain_failure() {
  Root root;Fixture f;sqpv::testing::fail_at(2);sqpv::AsyncStore store;
  check(sqpv::AsyncStore::create(root.path,f.manifest,f.options,f.context,{2,8192},store)==sqpv::Status::ok,"failure create");
  auto batch=f.batch(7);check(store.submit(batch)==sqpv::Status::ok,"failure admission");
  check(store.finish()==sqpv::Status::outcome_uncertain,"uncertainty retained");
  sqpv::AsyncSnapshot status;check(store.snapshot(status)==sqpv::Status::ok && status.failure==sqpv::Status::outcome_uncertain && !status.confirmed.committed_batches && status.pending_batches==1,"uncertainty no false watermark");
  check(store.submit(batch)==sqpv::Status::outcome_uncertain,"failed writer cannot silently retry");
  store.reset();sqpv::testing::fail_at(0);
  check(sqpv::AsyncStore::open(root.path,f.manifest,f.options,f.context,{2,8192},store)==sqpv::Status::ok,"reopen uncertain");
  check(store.snapshot(status)==sqpv::Status::ok && status.next_admission_sequence==7,"uncommitted admission not restored");
  check(store.submit(batch)==sqpv::Status::ok && store.finish()==sqpv::Status::ok,"explicit exact recovery retry");
}
void crash_recovery(int point) {
  Root root;int notice[2],resume[2],admitted[2];check(::pipe(notice)==0&&::pipe(resume)==0&&::pipe(admitted)==0,"crash pipes");
  auto child=::fork();check(child>=0,"fork");
  if(child==0) {
    Fixture f;sqpv::testing::pause_at(point,notice[1],resume[0]);sqpv::AsyncStore store;
    if(sqpv::AsyncStore::create(root.path,f.manifest,f.options,f.context,{2,8192},store)!=sqpv::Status::ok)::_exit(4);
    auto a=f.batch(7),b=f.batch(8);
    if(store.submit(a)!=sqpv::Status::ok||store.submit(b)!=sqpv::Status::ok)::_exit(5);
    signal_byte(admitted[1]);(void)store.finish();::_exit(6);
  }
  await_byte(admitted[0]);await_byte(notice[0]);check(::kill(child,SIGKILL)==0,"kill writer");
  int result=0;check(::waitpid(child,&result,0)==child&&WIFSIGNALED(result)&&WTERMSIG(result)==SIGKILL,"actual crash");
  Fixture f;sqpv::AsyncStore store;
  check(sqpv::AsyncStore::open(root.path,f.manifest,f.options,f.context,{2,8192},store)==sqpv::Status::ok,"crash reopen");
  sqpv::AsyncSnapshot status;check(store.snapshot(status)==sqpv::Status::ok,"crash status");
  check(status.pending_batches==0 && status.next_admission_sequence==(point==6?8:7),"recover actual storage, discard volatile tail");
  if(point==1) {auto a=f.batch(7);check(store.submit(a)==sqpv::Status::ok,"reacquire first");}
  auto b=f.batch(8);check(store.submit(b)==sqpv::Status::ok && store.finish()==sqpv::Status::ok,"explicit replay missing tail");
  sqfv::Batch read;sqpv::Receipt receipt;check(store.read(8,f.context,read,receipt)==sqpv::Status::ok&&read.content_id()==b.content_id(),"tail bytes preserved");
  store.reset();for(int fd:{notice[0],notice[1],resume[0],resume[1],admitted[0],admitted[1]})::close(fd);
}
}
int main(int argc, char**){::alarm(30);bounded_queue();uncertain_failure();if(argc==1){crash_recovery(1);crash_recovery(6);}std::puts("Async retention scenarios passed (SIGKILL cases run only without an argument)");}
