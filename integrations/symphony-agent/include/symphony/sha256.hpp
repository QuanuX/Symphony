#pragma once
#include <algorithm>
#include <array>
#include <cerrno>
#include <cstddef>
#include <cstdint>
#include <fcntl.h>
#include <iomanip>
#include <sstream>
#include <stdexcept>
#include <string>
#include <sys/stat.h>
#include <unistd.h>
#ifdef __APPLE__
#include <CommonCrypto/CommonDigest.h>
#else
#include <openssl/evp.h>
#endif
namespace symphony {
inline std::string file_sha256(const std::string& path, std::size_t max_bytes=1024ULL*1024*1024) {
  if(path.find('\0')!=std::string::npos || max_bytes==0)
    throw std::runtime_error("Invalid digest input or byte limit");
  // Nonblocking open prevents a path replaced with a FIFO from hanging before
  // it can be inspected. The opened descriptor, not a prior path stat, defines
  // the file being hashed. Existing symlink selections remain supported.
  struct FileDescriptor {
    int value;
    ~FileDescriptor() { if(value>=0) ::close(value); }
  } in{::open(path.c_str(),O_RDONLY|O_NONBLOCK|O_CLOEXEC)};
  if(in.value<0) throw std::runtime_error("Cannot open digest input");
  struct stat before{};
  if(::fstat(in.value,&before)!=0 || !S_ISREG(before.st_mode) || before.st_size<0)
    throw std::runtime_error("Digest input must be a regular file");
  const auto initial_size=static_cast<std::uintmax_t>(before.st_size);
  if(initial_size>max_bytes) throw std::runtime_error("Digest input exceeds byte limit");
  std::array<unsigned char,32> digest{};
  std::array<char,65536> buffer{};
#ifdef __APPLE__
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
  CC_SHA256_CTX ctx; CC_SHA256_Init(&ctx);
#else
  struct DigestContext {
    EVP_MD_CTX* value=EVP_MD_CTX_new();
    ~DigestContext() { EVP_MD_CTX_free(value); }
  } ctx;
  if(!ctx.value) throw std::runtime_error("SHA256 allocation failed");
  if(EVP_DigestInit_ex(ctx.value,EVP_sha256(),nullptr)!=1) throw std::runtime_error("SHA256 init failed");
#endif
  std::uintmax_t consumed=0;
  while(consumed<initial_size) {
    const auto requested=static_cast<std::size_t>(std::min<std::uintmax_t>(buffer.size(),initial_size-consumed));
    ssize_t count;
    do { count=::read(in.value,buffer.data(),requested); } while(count<0&&errno==EINTR);
    if(count<0) throw std::runtime_error("Digest input read failed");
    if(count==0) throw std::runtime_error("Digest input changed during hashing");
    consumed+=static_cast<std::size_t>(count);
#ifdef __APPLE__
    CC_SHA256_Update(&ctx,buffer.data(),static_cast<CC_LONG>(count));
#else
    if(EVP_DigestUpdate(ctx.value,buffer.data(),static_cast<std::size_t>(count))!=1)
      throw std::runtime_error("SHA256 update failed");
#endif
  }
  char extra;
  ssize_t extra_count;
  do { extra_count=::read(in.value,&extra,1); } while(extra_count<0&&errno==EINTR);
  if(extra_count<0) throw std::runtime_error("Digest input read failed");
  struct stat after{};
  if(extra_count!=0 || ::fstat(in.value,&after)!=0 ||
     before.st_dev!=after.st_dev || before.st_ino!=after.st_ino ||
     before.st_size!=after.st_size || before.st_mode!=after.st_mode)
    throw std::runtime_error("Digest input changed during hashing");
#ifdef __APPLE__
  const auto old_mtime=before.st_mtimespec, new_mtime=after.st_mtimespec;
  const auto old_ctime=before.st_ctimespec, new_ctime=after.st_ctimespec;
#else
  const auto old_mtime=before.st_mtim, new_mtime=after.st_mtim;
  const auto old_ctime=before.st_ctim, new_ctime=after.st_ctim;
#endif
  if(old_mtime.tv_sec!=new_mtime.tv_sec || old_mtime.tv_nsec!=new_mtime.tv_nsec ||
     old_ctime.tv_sec!=new_ctime.tv_sec || old_ctime.tv_nsec!=new_ctime.tv_nsec)
    throw std::runtime_error("Digest input changed during hashing");
#ifdef __APPLE__
  CC_SHA256_Final(digest.data(),&ctx);
#pragma clang diagnostic pop
#else
  unsigned size=0;
  if(EVP_DigestFinal_ex(ctx.value,digest.data(),&size)!=1 || size!=32)
    throw std::runtime_error("SHA256 final failed");
#endif
  std::ostringstream out; out<<"sha256:"<<std::hex<<std::setfill('0');
  for(auto byte:digest) out<<std::setw(2)<<static_cast<unsigned>(byte);
  return out.str();
}
}
