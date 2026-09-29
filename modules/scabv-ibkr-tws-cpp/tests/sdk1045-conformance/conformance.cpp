// Compile-only boundary for an externally authorized SDK. This translation unit
// contains no IBKR implementation, login, event loop, or runtime conformance.
#include <Contract.h>
#include <EClient.h>
#include <symphony/scabv/tws.hpp>
template class symphony::scabv::tws::Sdk1045Driver<EClient, Contract,
                                                   TagValueListSPtr>;
