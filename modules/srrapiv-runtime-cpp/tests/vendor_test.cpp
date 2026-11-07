// Offline SDK qualification: actual vendor libraries, no REngine construction
// and no provider connection. Include the implementation to exercise native
// callbacks directly rather than fabricating provider traffic in production.
#include "../src/vendor.cpp"
#include <cassert>
#include <iostream>
#include <fstream>

using namespace symphony::srrapiv;
int main() {
    std::vector<Json> events;
    Vendor vendor(Json{{"secrets",{{"test-login-secret","private-test-value"}}}},[&](Json e){events.push_back(std::move(e));});
    auto description=vendor.describe();
    assert(description.at("sdk_available")==true);
    assert(description.at("sdk_native_version")=="13.7.0.0");
    assert(description.at("operations").size()==118);
    assert(description.at("callbacks").size()==95);
    assert(description.dump().find("private-test-value")==std::string::npos);
    auto version=vendor.invoke("REngine.getVersion#1",Json::object());
    assert(version.at("native_return")==OK);
    assert(version.at("outputs").at("pVersion").at("text")=="13.7.0.0");
    auto blocked=vendor.invoke("runtime.initialize",Json::object());
    assert(blocked.at("code")=="PROVIDER_INITIALIZATION_DISABLED");
    auto missing=vendor.invoke("REngine.ping#1",Json{{"iConnId",1},{"pContext",nullptr}});
    assert(missing.at("code")=="ENGINE_NOT_INITIALIZED");

    Json parameters{{"llQty","9223372036854775807"},{"sTicker",{{"bytes_hex","ff0041"}}},{"dPrice",{{"f64_bits","8000000000000000"}}},{"bMinQty",true},{"llMinQty","1"}};
    auto inspect=vendor.invoke("object.inspect",Json{{"type","OrderParams"},{"value",parameters}});
    assert(inspect.at("status")=="ok");
    if (!inspect.contains("value")) { std::cerr << "inspect: " << inspect.dump() << std::endl; return 1; }
    assert(inspect.at("value").at("llQty")=="9223372036854775807");
    assert(inspect.at("value").at("sTicker").at("bytes_hex")=="ff0041");
    assert(!inspect.at("value").at("sTicker").contains("text"));
    assert(inspect.at("value").at("dPrice").at("f64_bits")=="8000000000000000");
    auto empty=vendor.invoke("object.inspect",Json{{"type","OrderParams"},{"value",{{"sTicker",""},{"sExchange",nullptr}}}});
    assert(empty.at("value").at("sTicker").at("pointer_present")==true);
    assert(empty.at("value").at("sExchange").at("pointer_present")==false);
    auto overflow=vendor.invoke("object.inspect",Json{{"type","OrderParams"},{"value",{{"llQty","9223372036854775808"}}}});
    assert(overflow.at("status")=="error");assert(overflow.at("native_called")==false);
    auto untyped=vendor.invoke("object.inspect",Json{{"type","OrderParams"},{"value",{{"llQty",9007199254740992ULL}}}});
    assert(untyped.at("status")=="error");
    auto unknown=vendor.invoke("object.inspect",Json{{"type","OrderParams"},{"value",{{"misspelledQuantity",1}}}});
    assert(unknown.at("status")=="error");
    auto inherited=vendor.invoke("object.inspect",Json{{"type","OrderFillReport"},{"value",{{"sOrderNum","parent-field"},{"sFillType","child-field"}}}});
    if (!inherited.contains("value")) { std::cerr << "inherited: " << inherited.dump() << std::endl; return 1; }
    assert(inherited.at("value").at("sOrderNum").at("text")=="parent-field");
    assert(inherited.at("value").at("sFillType").at("text")=="child-field");
    auto accounts=vendor.invoke("object.inspect",Json{{"type","AccountListInfo"},{"value",{{"asAccountInfoArray",Json::array({Json{{"sAccountId","a"}},Json{{"sAccountId","b"}}})},{"iArrayLen",2}}}});
    if (!accounts.contains("value")) { std::cerr << "accounts: " << accounts.dump() << std::endl; return 1; }
    assert(accounts.at("value").at("asAccountInfoArray").size()==2);
    assert(accounts.at("value").at("asAccountInfoArray").at(1).at("sAccountId").at("text")=="b");
    auto mismatch=vendor.invoke("object.inspect",Json{{"type","AccountListInfo"},{"value",{{"asAccountInfoArray",Json::array({Json::object()})},{"iArrayLen",2}}}});
    assert(mismatch.at("status")=="error");
    auto shared_mismatch=vendor.invoke("object.inspect",Json{{"type","BarInfo"},{"value",{{"adProfilePrices",Json::array({1.0})},{"allProfileNoAggressorVolumes",Json::array({"1","2"})}}}});
    assert(shared_mismatch.at("status")=="error");
    auto shared_optional=vendor.invoke("object.inspect",Json{{"type","BarInfo"},{"value",{{"adProfilePrices",Json::array({1.0})},{"allProfileNoAggressorVolumes",nullptr},{"iProfileArrayLen",1}}}});
    assert(shared_optional.at("status")=="ok");
    assert(shared_optional.at("value").at("iProfileArrayLen")==1);
    assert(shared_optional.at("value").at("allProfileNoAggressorVolumes").is_null());
    auto bracket=vendor.invoke("object.construct",Json{{"type","BracketTier"},{"iTicksIn",3},{"llQtyIn","4294967296"},{"llQtyReleasedIn","0"}});
    if (!bracket.contains("value")) { std::cerr << "bracket: " << bracket.dump() << std::endl; return 1; }
    assert(bracket.at("value").at("llQty")=="4294967296");
    auto convert=vendor.invoke("object.convert",Json{{"type","OrderParams"},{"from_type","LimitOrderParams"},{"value",{{"llQty","4294967296"},{"sTicker","native-conversion"},{"pAccount",{{"sAccountId","test-account"},{"sFcmId","test-fcm"},{"sIbId","test-ib"},{"sAccountName","test-name"}}},{"sExchange","CME"},{"sBuySellType","B"},{"sDuration","DAY"},{"sEntryType","M"},{"sTradeRoute","test-route"},{"dPrice",1.0},{"sTradingAlgorithm",""},{"sRoutingInstructions",""},{"sTag",""},{"sUserMsg",""}}}});
    if (!convert.contains("value")) { std::cerr << "convert: " << convert.dump() << std::endl; return 1; }
    assert(convert.at("value").at("llQty")=="4294967296");
    auto login=vendor.invoke("object.inspect",Json{{"type","LoginParams"},{"value",{{"sTsPassword",{{"secret_ref","test-login-secret"}}}}}});
    if (!login.contains("value")) { std::cerr << "login: " << login.dump() << std::endl; return 1; }
    assert(login.at("value").at("sTsPassword").at("redacted")==true);
    assert(login.dump().find("private-test-value")==std::string::npos);
    auto plaintext=vendor.invoke("object.inspect",Json{{"type","LoginParams"},{"value",{{"sTsPassword","plaintext-denied"}}}});
    assert(plaintext.at("status")=="error");
    auto private_environment=vendor.invoke("object.inspect",Json{{"type","EnvironmentVariable"},{"value",{{"sName","API_KEY"},{"sValue","fixture-key"}}}});
    assert(private_environment.at("value").at("sValue").at("redacted")==true);
    assert(private_environment.dump().find("fixture-key")==std::string::npos);
    char destination[]="/private/tmp/srrapiv-secret-output-XXXXXX";
    int destination_fd=mkstemp(destination);assert(destination_fd>=0);assert(fchmod(destination_fd,0600)==0);close(destination_fd);
    {
        Vendor exporter(Json{{"secret_outputs",{{"environment-test",destination}}},{"protected_environment_outputs",{{"API_KEY","environment-test"}}}},{});
        auto protected_result=exporter.invoke("object.inspect",Json{{"type","EnvironmentVariable"},{"value",{{"sName","API_KEY"},{"sValue","fixture-key"}}}});
        assert(protected_result.at("value").at("sValue").at("written")==true);
        assert(protected_result.dump().find("fixture-key")==std::string::npos);
        std::ifstream input(destination);std::string written;std::getline(input,written);assert(written=="fixture-key");
    }
    assert(unlink(destination)==0);
    Vendor deny(Json{{"allowed_methods",Json::array()}},{});
    assert(deny.invoke("REngine.getVersion#1",Json::object()).at("code")=="METHOD_NOT_ALLOWED");
    assert(vendor.invoke("REngine.sendOrder#1",Json::object()).at("code")=="TRADING_DISABLED");

    State state(Json::object(),[&](Json e){events.push_back(std::move(e));});
#include "../src/vendor_callback_checks.inc"
    QuoteReport unavailable{};initialize_public(unavailable);unavailable.bFill=false;
    unavailable.sFillSide={reinterpret_cast<char*>(1),5};
    Encoding unavailable_encoding(state);auto unavailable_fields=encode(unavailable,unavailable_encoding);
    assert(unavailable_fields.at("sFillSide").at("state")=="native-value-unavailable");
    auto context=state.context(Json{{"context_id","callback-test"}});
    auto handle=state.handle(context,"context");assert(state.context(handle)==context);
    TradeInfo trade{};initialize_public(trade);char symbol[]="ESZ6";trade.sTicker={symbol,4};int code=-1;
    assert(static_cast<Callbacks&>(*state.callbacks).TradePrint(&trade,context,&code)==OK);
    assert(code==API_OK);auto captured=events.back();symbol[0]='N';
    assert(captured.at("payload").at("pInfo").at("sTicker").at("text")=="ESZ6");
    assert(captured.at("payload").at("pContext")==handle);
    LimitOrderBookInfo book{};initialize_public(book);double dummy=0;book.adAskPriceArray=&dummy;book.iAskArrayLen=static_cast<int>(state.max_array)+1;
    assert(static_cast<Callbacks&>(*state.callbacks).LimitOrderBook(&book,nullptr,&code)==OK);
    assert(events.back().at("event")=="runtime.callback_gap");
    assert(events.back().at("state_uncertain")==true);
    tsNCharcb timer_name{const_cast<char*>("test-timer"),10};state.timer->call(&timer_name,context,&code);assert(events.back().at("event")=="OmneTimerCB.call#1");
    state.stop();bool stale=false;try{state.context(handle);}catch(const std::invalid_argument&){stale=true;}assert(stale);
    assert(state.engine==nullptr);
    std::cout<<"SRRAPIV offline SDK checks passed: 118 dispatch signatures, 95 callbacks, exact types, arrays, inherited fields, ownership and policy. No engine initialization or provider calls.\n";
}
