// Generated typed dispatch; overloads resolved by native argument types.
if (id == "REngine.getAccounts#1") {
  validate_keys(p, {"pStatus"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pStatus = decode_pointer<tsNCharcb>(required(p, "pStatus"), arena);
  state.native_call_started = true; int native_return = state.engine->getAccounts(pStatus, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getAdmCallbacks#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  AdmCallbacks* apCallbacks = nullptr;
  state.native_call_started = true; int native_return = state.engine->getAdmCallbacks(&apCallbacks, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["apCallbacks"] = state.handle(apCallbacks, "AdmCallbacks"); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getAdminName#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pAdminName{};
  state.native_call_started = true; int native_return = state.engine->getAdminName(&pAdminName, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pAdminName"] = encode(pAdminName, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getAppName#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pAppName{};
  state.native_call_started = true; int native_return = state.engine->getAppName(&pAppName, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pAppName"] = encode(pAppName, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getAppVersion#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pAppVersion{};
  state.native_call_started = true; int native_return = state.engine->getAppVersion(&pAppVersion, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pAppVersion"] = encode(pAppVersion, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getCallbacks#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  RCallbacks* apCallbacks = nullptr;
  state.native_call_started = true; int native_return = state.engine->getCallbacks(&apCallbacks, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["apCallbacks"] = state.handle(apCallbacks, "RCallbacks"); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getContext#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* apContext = nullptr;
  state.native_call_started = true; int native_return = state.engine->getContext(&apContext, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["apContext"] = state.handle(apContext, "context"); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getIhCnnctPt#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pIhCnnctPt{};
  state.native_call_started = true; int native_return = state.engine->getIhCnnctPt(&pIhCnnctPt, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pIhCnnctPt"] = encode(pIhCnnctPt, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getMdCnnctPt#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pMdCnnctPt{};
  state.native_call_started = true; int native_return = state.engine->getMdCnnctPt(&pMdCnnctPt, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pMdCnnctPt"] = encode(pMdCnnctPt, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getOmne#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* apOmne = nullptr;
  state.native_call_started = true; int native_return = state.engine->getOmne(&apOmne, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["apOmne"] = state.handle(apOmne, "native-engine"); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getOrderContext#1") {
  validate_keys(p, {"pOrderNum"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  void* apContext = nullptr;
  state.native_call_started = true; int native_return = state.engine->getOrderContext(pOrderNum, &apContext, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["apContext"] = state.handle(apContext, "context"); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getPendingInputSize#1") {
  validate_keys(p, {"iConnId"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  int iConnId{}; decode(required(p, "iConnId"), iConnId, arena);
  unsigned long int aulPendingBytes = 0;
  state.native_call_started = true; int native_return = state.engine->getPendingInputSize(iConnId, &aulPendingBytes, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["aulPendingBytes"] = encode(aulPendingBytes, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getPnlCnnctPt#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pPnlCnnctPt{};
  state.native_call_started = true; int native_return = state.engine->getPnlCnnctPt(&pPnlCnnctPt, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pPnlCnnctPt"] = encode(pPnlCnnctPt, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getTsCnnctPt#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pTsCnnctPt{};
  state.native_call_started = true; int native_return = state.engine->getTsCnnctPt(&pTsCnnctPt, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pTsCnnctPt"] = encode(pTsCnnctPt, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getUser#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pUser{};
  state.native_call_started = true; int native_return = state.engine->getUser(&pUser, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pUser"] = encode(pUser, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getUserProfile#1") {
  validate_keys(p, {"iConnId", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  int iConnId{}; decode(required(p, "iConnId"), iConnId, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getUserProfile(iConnId, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getVersion#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pVersion{};
  state.native_call_started = true; int native_return = REngine::getVersion(&pVersion, &native_code);
  Encoding encoding(state); if (native_return == OK) { outputs["pVersion"] = encode(pVersion, encoding); }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.setContext#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->setContext(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.setOrderContext#1") {
  validate_keys(p, {"pOrderNum", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->setOrderContext(pOrderNum, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.cancelAllOrders#1") {
  validate_keys(p, {"pAccount", "pEntryType", "pTradingAlgorithm", "pWindowName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pEntryType = decode_pointer<tsNCharcb>(required(p, "pEntryType"), arena);
  tsNCharcb* pTradingAlgorithm = decode_pointer<tsNCharcb>(required(p, "pTradingAlgorithm"), arena);
  tsNCharcb* pWindowName = decode_pointer<tsNCharcb>(required(p, "pWindowName"), arena);
  state.native_call_started = true; int native_return = state.engine->cancelAllOrders(pAccount, pEntryType, pTradingAlgorithm, pWindowName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.cancelOrder#1") {
  validate_keys(p, {"pAccount", "pOrderNum", "pEntryType", "pTradingAlgorithm", "pUserMsg", "pContext", "pWindowName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  tsNCharcb* pEntryType = decode_pointer<tsNCharcb>(required(p, "pEntryType"), arena);
  tsNCharcb* pTradingAlgorithm = decode_pointer<tsNCharcb>(required(p, "pTradingAlgorithm"), arena);
  tsNCharcb* pUserMsg = decode_pointer<tsNCharcb>(required(p, "pUserMsg"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  tsNCharcb* pWindowName = decode_pointer<tsNCharcb>(required(p, "pWindowName"), arena);
  state.native_call_started = true; int native_return = state.engine->cancelOrder(pAccount, pOrderNum, pEntryType, pTradingAlgorithm, pUserMsg, pContext, pWindowName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.cancelOrderList#1") {
  validate_keys(p, {"aoAccountArray", "asOrderNumArray", "asEntryTypeArray", "asTradingAlgorithmArray", "asUserMsgArray", "apContextArray", "iArrayLen", "pWindowName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto aoAccountArray = decode_array<AccountInfo>(required(p, "aoAccountArray"), arena);
  auto asOrderNumArray = decode_array<tsNCharcb>(required(p, "asOrderNumArray"), arena);
  auto asEntryTypeArray = decode_array<tsNCharcb>(required(p, "asEntryTypeArray"), arena);
  auto asTradingAlgorithmArray = decode_array<tsNCharcb>(required(p, "asTradingAlgorithmArray"), arena);
  auto asUserMsgArray = decode_array<tsNCharcb>(required(p, "asUserMsgArray"), arena);
  auto apContextArray = decode_array<void*>(required(p, "apContextArray"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  tsNCharcb* pWindowName = decode_pointer<tsNCharcb>(required(p, "pWindowName"), arena);
  require_array_count(p.at("aoAccountArray"), iArrayLen);
  require_array_count(p.at("asOrderNumArray"), iArrayLen);
  require_array_count(p.at("asEntryTypeArray"), iArrayLen);
  require_array_count(p.at("asTradingAlgorithmArray"), iArrayLen);
  require_array_count(p.at("asUserMsgArray"), iArrayLen);
  require_array_count(p.at("apContextArray"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->cancelOrderList(aoAccountArray, asOrderNumArray, asEntryTypeArray, asTradingAlgorithmArray, asUserMsgArray, apContextArray, iArrayLen, pWindowName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.cancelQuoteList#1") {
  validate_keys(p, {"pParams", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_array<QuoteCancelParams>(required(p, "pParams"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("pParams"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->cancelQuoteList(pParams, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.changePassword#1") {
  validate_keys(p, {"pOldPassword", "pNewPassword"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb pOldPassword = arena.secret_string(required(p, "pOldPassword"));
  tsNCharcb pNewPassword = arena.secret_string(required(p, "pNewPassword"));
  state.native_call_started = true; int native_return = state.engine->changePassword(&pOldPassword, &pNewPassword, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.createUserDefinedSpread#1") {
  validate_keys(p, {"pAccount", "pExchange", "pTradeRoute", "pEntryType", "pStrategyType", "iNumLegs", "asLegTickers", "aiLegRatios", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTradeRoute = decode_pointer<tsNCharcb>(required(p, "pTradeRoute"), arena);
  tsNCharcb* pEntryType = decode_pointer<tsNCharcb>(required(p, "pEntryType"), arena);
  tsNCharcb* pStrategyType = decode_pointer<tsNCharcb>(required(p, "pStrategyType"), arena);
  int iNumLegs{}; decode(required(p, "iNumLegs"), iNumLegs, arena);
  auto asLegTickers = decode_array<tsNCharcb>(required(p, "asLegTickers"), arena);
  auto aiLegRatios = decode_array<int>(required(p, "aiLegRatios"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  require_array_count(p.at("asLegTickers"), iNumLegs);
  require_array_count(p.at("aiLegRatios"), iNumLegs);
  state.native_call_started = true; int native_return = state.engine->createUserDefinedSpread(pAccount, pExchange, pTradeRoute, pEntryType, pStrategyType, iNumLegs, asLegTickers, aiLegRatios, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.exitPosition#1") {
  validate_keys(p, {"pAccount", "pExchange", "pTicker", "pEntryType", "pTag", "pTradingAlgorithm", "pUserMsg", "pWindowName", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  tsNCharcb* pEntryType = decode_pointer<tsNCharcb>(required(p, "pEntryType"), arena);
  tsNCharcb* pTag = decode_pointer<tsNCharcb>(required(p, "pTag"), arena);
  tsNCharcb* pTradingAlgorithm = decode_pointer<tsNCharcb>(required(p, "pTradingAlgorithm"), arena);
  tsNCharcb* pUserMsg = decode_pointer<tsNCharcb>(required(p, "pUserMsg"), arena);
  tsNCharcb* pWindowName = decode_pointer<tsNCharcb>(required(p, "pWindowName"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->exitPosition(pAccount, pExchange, pTicker, pEntryType, pTag, pTradingAlgorithm, pUserMsg, pWindowName, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getAuxRefData#1") {
  validate_keys(p, {"pExchange", "pTicker"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  state.native_call_started = true; int native_return = state.engine->getAuxRefData(pExchange, pTicker, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getEasyToBorrowList#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getEasyToBorrowList(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getEnvironment#1") {
  validate_keys(p, {"pKey", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pKey = decode_pointer<tsNCharcb>(required(p, "pKey"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getEnvironment(pKey, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getEquityOptionStrategyList#1") {
  validate_keys(p, {"pExchange", "pUnderlying", "pStrategyType", "pExpiration", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pUnderlying = decode_pointer<tsNCharcb>(required(p, "pUnderlying"), arena);
  tsNCharcb* pStrategyType = decode_pointer<tsNCharcb>(required(p, "pStrategyType"), arena);
  tsNCharcb* pExpiration = decode_pointer<tsNCharcb>(required(p, "pExpiration"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getEquityOptionStrategyList(pExchange, pUnderlying, pStrategyType, pExpiration, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getInstrumentByUnderlying#1") {
  validate_keys(p, {"pUnderlying", "pExchange", "pExpiration", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pUnderlying = decode_pointer<tsNCharcb>(required(p, "pUnderlying"), arena);
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pExpiration = decode_pointer<tsNCharcb>(required(p, "pExpiration"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getInstrumentByUnderlying(pUnderlying, pExchange, pExpiration, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getOptionList#1") {
  validate_keys(p, {"pExchange", "pProductCode", "pExpirationCCYYMM", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pProductCode = decode_pointer<tsNCharcb>(required(p, "pProductCode"), arena);
  tsNCharcb* pExpirationCCYYMM = decode_pointer<tsNCharcb>(required(p, "pExpirationCCYYMM"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getOptionList(pExchange, pProductCode, pExpirationCCYYMM, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getPriceIncrInfo#1") {
  validate_keys(p, {"pExchange", "pTicker"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  state.native_call_started = true; int native_return = state.engine->getPriceIncrInfo(pExchange, pTicker, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getProductRmsInfo#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->getProductRmsInfo(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getRefData#1") {
  validate_keys(p, {"pExchange", "pTicker"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  state.native_call_started = true; int native_return = state.engine->getRefData(pExchange, pTicker, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getStrategyInfo#1") {
  validate_keys(p, {"pExchange", "pTicker", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getStrategyInfo(pExchange, pTicker, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getStrategyList#1") {
  validate_keys(p, {"pExchange", "pProductCode", "pStrategyType", "pExpirationCCYYMM", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pProductCode = decode_pointer<tsNCharcb>(required(p, "pProductCode"), arena);
  tsNCharcb* pStrategyType = decode_pointer<tsNCharcb>(required(p, "pStrategyType"), arena);
  tsNCharcb* pExpirationCCYYMM = decode_pointer<tsNCharcb>(required(p, "pExpirationCCYYMM"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getStrategyList(pExchange, pProductCode, pStrategyType, pExpirationCCYYMM, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.getVolumeAtPrice#1") {
  validate_keys(p, {"pExchange", "pTicker", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->getVolumeAtPrice(pExchange, pTicker, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.isThereAnAggregator#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->isThereAnAggregator(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.linkOrders#1") {
  validate_keys(p, {"aoAccountArray", "asOrderNumArray", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto aoAccountArray = decode_array<AccountInfo>(required(p, "aoAccountArray"), arena);
  auto asOrderNumArray = decode_array<tsNCharcb>(required(p, "asOrderNumArray"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("aoAccountArray"), iArrayLen);
  require_array_count(p.at("asOrderNumArray"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->linkOrders(aoAccountArray, asOrderNumArray, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listAgreements#1") {
  validate_keys(p, {"bAccepted", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  bool bAccepted{}; decode(required(p, "bAccepted"), bAccepted, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listAgreements(bAccepted, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listBinaryContracts#1") {
  validate_keys(p, {"pExchange", "pProductCode", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pProductCode = decode_pointer<tsNCharcb>(required(p, "pProductCode"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listBinaryContracts(pExchange, pProductCode, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listEnvironments#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listEnvironments(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listExchanges#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listExchanges(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listIbs#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listIbs(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listOrderHistoryDates#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listOrderHistoryDates(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listAssignedUsers#1") {
  validate_keys(p, {"pAccount", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listAssignedUsers(pAccount, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listTradeRoutes#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listTradeRoutes(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.listUsers#1") {
  validate_keys(p, {"pIbId", "pUserType", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pIbId = decode_pointer<tsNCharcb>(required(p, "pIbId"), arena);
  tsNCharcb* pUserType = decode_pointer<tsNCharcb>(required(p, "pUserType"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->listUsers(pIbId, pUserType, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.login#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<LoginParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->login(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.loginRepository#1") {
  validate_keys(p, {"pEnvKey", "pUser", "pPassword", "pCnnctPt", "pCallbacks"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pEnvKey = decode_pointer<tsNCharcb>(required(p, "pEnvKey"), arena);
  tsNCharcb* pUser = decode_pointer<tsNCharcb>(required(p, "pUser"), arena);
  tsNCharcb pPassword = arena.secret_string(required(p, "pPassword"));
  tsNCharcb* pCnnctPt = decode_pointer<tsNCharcb>(required(p, "pCnnctPt"), arena);
  auto pCallbacks = static_cast<RCallbacks*>(state.callback_handle("RCallbacks", p.value("pCallbacks", Json())));
  state.native_call_started = true; int native_return = state.engine->loginRepository(pEnvKey, pUser, &pPassword, pCnnctPt, pCallbacks, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.logout#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->logout(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.logoutRepository#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->logoutRepository(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyBracketTier#1") {
  validate_keys(p, {"pAccount", "pOrderNum", "bTarget", "iOldTickOffset", "iNewTickOffset", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  bool bTarget{}; decode(required(p, "bTarget"), bTarget, arena);
  int iOldTickOffset{}; decode(required(p, "iOldTickOffset"), iOldTickOffset, arena);
  int iNewTickOffset{}; decode(required(p, "iNewTickOffset"), iNewTickOffset, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->modifyBracketTier(pAccount, pOrderNum, bTarget, iOldTickOffset, iNewTickOffset, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrder#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<ModifyLimitOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->modifyOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrder#2") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<ModifyStopLimitOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->modifyOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrder#3") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<ModifyStopMarketOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->modifyOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrder#4") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<ModifyOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->modifyOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrderList#1") {
  validate_keys(p, {"pParams", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_array<ModifyOrderParams>(required(p, "pParams"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("pParams"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->modifyOrderList(pParams, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.modifyOrderRefData#1") {
  validate_keys(p, {"pAccount", "pOrderNum", "pUserMsg", "pUserTag"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  tsNCharcb* pUserMsg = decode_pointer<tsNCharcb>(required(p, "pUserMsg"), arena);
  tsNCharcb* pUserTag = decode_pointer<tsNCharcb>(required(p, "pUserTag"), arena);
  state.native_call_started = true; int native_return = state.engine->modifyOrderRefData(pAccount, pOrderNum, pUserMsg, pUserTag, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.ping#1") {
  validate_keys(p, {"iConnId", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  int iConnId{}; decode(required(p, "iConnId"), iConnId, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->ping(iConnId, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.rebuildBook#1") {
  validate_keys(p, {"pExchange", "pTicker"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  state.native_call_started = true; int native_return = state.engine->rebuildBook(pExchange, pTicker, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.rebuildDboBook#1") {
  validate_keys(p, {"pExchange", "pTicker", "bPrice", "dPrice", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  bool bPrice{}; decode(required(p, "bPrice"), bPrice, arena);
  double dPrice{}; decode(required(p, "dPrice"), dPrice, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->rebuildDboBook(pExchange, pTicker, bPrice, dPrice, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.removeTimer#1") {
  validate_keys(p, {"pTimerName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pTimerName = decode_pointer<tsNCharcb>(required(p, "pTimerName"), arena);
  state.native_call_started = true; int native_return = state.engine->removeTimer(pTimerName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayAllOrders#1") {
  validate_keys(p, {"pAccount", "iStartSsboe", "iEndSsboe"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  int iStartSsboe{}; decode(required(p, "iStartSsboe"), iStartSsboe, arena);
  int iEndSsboe{}; decode(required(p, "iEndSsboe"), iEndSsboe, arena);
  state.native_call_started = true; int native_return = state.engine->replayAllOrders(pAccount, iStartSsboe, iEndSsboe, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayBrackets#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->replayBrackets(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayExecutions#1") {
  validate_keys(p, {"pAccount", "iStartSsboe", "iEndSsboe"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  int iStartSsboe{}; decode(required(p, "iStartSsboe"), iStartSsboe, arena);
  int iEndSsboe{}; decode(required(p, "iEndSsboe"), iEndSsboe, arena);
  state.native_call_started = true; int native_return = state.engine->replayExecutions(pAccount, iStartSsboe, iEndSsboe, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayHistoricalOrders#1") {
  validate_keys(p, {"pAccount", "pDate"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pDate = decode_pointer<tsNCharcb>(required(p, "pDate"), arena);
  state.native_call_started = true; int native_return = state.engine->replayHistoricalOrders(pAccount, pDate, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayOpenOrders#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->replayOpenOrders(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayPnl#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->replayPnl(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayQuotes#1") {
  validate_keys(p, {"pAccount", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->replayQuotes(pAccount, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replaySingleHistoricalOrder#1") {
  validate_keys(p, {"pAccount", "pOrderNum", "pDate", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  tsNCharcb* pDate = decode_pointer<tsNCharcb>(required(p, "pDate"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->replaySingleHistoricalOrder(pAccount, pOrderNum, pDate, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replaySingleOrder#1") {
  validate_keys(p, {"pAccount", "pOrderNum", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  tsNCharcb* pOrderNum = decode_pointer<tsNCharcb>(required(p, "pOrderNum"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->replaySingleOrder(pAccount, pOrderNum, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayTrades#1") {
  validate_keys(p, {"pExchange", "pTicker", "iStartSsboe", "iEndSsboe"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  int iStartSsboe{}; decode(required(p, "iStartSsboe"), iStartSsboe, arena);
  int iEndSsboe{}; decode(required(p, "iEndSsboe"), iEndSsboe, arena);
  state.native_call_started = true; int native_return = state.engine->replayTrades(pExchange, pTicker, iStartSsboe, iEndSsboe, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.resumeInput#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->resumeInput(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.searchInstrument#1") {
  validate_keys(p, {"pExchange", "asTerms", "iNumTerms", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  auto asTerms = decode_array<SearchTerm>(required(p, "asTerms"), arena);
  int iNumTerms{}; decode(required(p, "iNumTerms"), iNumTerms, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  require_array_count(p.at("asTerms"), iNumTerms);
  state.native_call_started = true; int native_return = state.engine->searchInstrument(pExchange, asTerms, iNumTerms, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendBracketOrder#1") {
  validate_keys(p, {"pEntry", "pBracketType", "pOrderOperationType", "bTrailingStop", "iTrailByPriceId", "iTargetArrayLen", "pTargetQty", "pTargetTicks", "iStopArrayLen", "pStopQty", "pStopTicks"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pEntry = decode_pointer<OrderParams>(required(p, "pEntry"), arena); if (!pEntry) throw std::invalid_argument("null native object: pEntry");
  tsNCharcb* pBracketType = decode_pointer<tsNCharcb>(required(p, "pBracketType"), arena);
  tsNCharcb* pOrderOperationType = decode_pointer<tsNCharcb>(required(p, "pOrderOperationType"), arena);
  bool bTrailingStop{}; decode(required(p, "bTrailingStop"), bTrailingStop, arena);
  int iTrailByPriceId{}; decode(required(p, "iTrailByPriceId"), iTrailByPriceId, arena);
  int iTargetArrayLen{}; decode(required(p, "iTargetArrayLen"), iTargetArrayLen, arena);
  auto pTargetQty = decode_array<int>(required(p, "pTargetQty"), arena);
  auto pTargetTicks = decode_array<int>(required(p, "pTargetTicks"), arena);
  int iStopArrayLen{}; decode(required(p, "iStopArrayLen"), iStopArrayLen, arena);
  auto pStopQty = decode_array<int>(required(p, "pStopQty"), arena);
  auto pStopTicks = decode_array<int>(required(p, "pStopTicks"), arena);
  require_array_count(p.at("pTargetQty"), iTargetArrayLen);
  require_array_count(p.at("pTargetTicks"), iTargetArrayLen);
  require_array_count(p.at("pStopQty"), iStopArrayLen);
  require_array_count(p.at("pStopTicks"), iStopArrayLen);
  state.native_call_started = true; int native_return = state.engine->sendBracketOrder(pEntry, pBracketType, pOrderOperationType, bTrailingStop, iTrailByPriceId, iTargetArrayLen, pTargetQty, pTargetTicks, iStopArrayLen, pStopQty, pStopTicks, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendBracketOrder#2") {
  validate_keys(p, {"pEntry", "pBracketParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pEntry = decode_pointer<OrderParams>(required(p, "pEntry"), arena); if (!pEntry) throw std::invalid_argument("null native object: pEntry");
  auto pBracketParams = decode_pointer<BracketParams>(required(p, "pBracketParams"), arena); if (!pBracketParams) throw std::invalid_argument("null native object: pBracketParams");
  state.native_call_started = true; int native_return = state.engine->sendBracketOrder(pEntry, pBracketParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOcaList#1") {
  validate_keys(p, {"pOcaType", "pParamsArray", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pOcaType = decode_pointer<tsNCharcb>(required(p, "pOcaType"), arena);
  auto pParamsArray = decode_array<OrderParams>(required(p, "pParamsArray"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("pParamsArray"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->sendOcaList(pOcaType, pParamsArray, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOrder#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<LimitOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->sendOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOrder#2") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<MarketOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->sendOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOrder#3") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<StopLimitOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->sendOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOrder#4") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<StopMarketOrderParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->sendOrder(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.sendOrderList#1") {
  validate_keys(p, {"pParamsArray", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParamsArray = decode_array<OrderParams>(required(p, "pParamsArray"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("pParamsArray"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->sendOrderList(pParamsArray, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.setEnvironmentVariable#1") {
  validate_keys(p, {"pKey", "pVariable", "pValue"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pKey = decode_pointer<tsNCharcb>(required(p, "pKey"), arena);
  tsNCharcb* pVariable = decode_pointer<tsNCharcb>(required(p, "pVariable"), arena);
  tsNCharcb* pValue = decode_pointer<tsNCharcb>(required(p, "pValue"), arena);
  state.native_call_started = true; int native_return = state.engine->setEnvironmentVariable(pKey, pVariable, pValue, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.startTimer#1") {
  validate_keys(p, {"pTimerName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pTimerName = decode_pointer<tsNCharcb>(required(p, "pTimerName"), arena);
  state.native_call_started = true; int native_return = state.engine->startTimer(pTimerName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.stopTimer#1") {
  validate_keys(p, {"pTimerName"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pTimerName = decode_pointer<tsNCharcb>(required(p, "pTimerName"), arena);
  state.native_call_started = true; int native_return = state.engine->stopTimer(pTimerName, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.submitQuoteList#1") {
  validate_keys(p, {"pParams", "iArrayLen"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_array<QuoteParams>(required(p, "pParams"), arena);
  int iArrayLen{}; decode(required(p, "iArrayLen"), iArrayLen, arena);
  require_array_count(p.at("pParams"), iArrayLen);
  state.native_call_started = true; int native_return = state.engine->submitQuoteList(pParams, iArrayLen, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribe#1") {
  validate_keys(p, {"pExchange", "pTicker", "iFlags"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  int iFlags{}; decode(required(p, "iFlags"), iFlags, arena);
  state.native_call_started = true; int native_return = state.engine->subscribe(pExchange, pTicker, iFlags, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeAutoLiquidate#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->subscribeAutoLiquidate(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeBracket#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->subscribeBracket(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeByUnderlying#1") {
  validate_keys(p, {"pUnderlying", "pExchange", "pExpiration", "iFlags", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pUnderlying = decode_pointer<tsNCharcb>(required(p, "pUnderlying"), arena);
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pExpiration = decode_pointer<tsNCharcb>(required(p, "pExpiration"), arena);
  int iFlags{}; decode(required(p, "iFlags"), iFlags, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->subscribeByUnderlying(pUnderlying, pExchange, pExpiration, iFlags, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeDbo#1") {
  validate_keys(p, {"pExchange", "pTicker", "bPrice", "dPrice", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  bool bPrice{}; decode(required(p, "bPrice"), bPrice, arena);
  double dPrice{}; decode(required(p, "dPrice"), dPrice, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->subscribeDbo(pExchange, pTicker, bPrice, dPrice, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeEasyToBorrow#1") {
  validate_keys(p, {"pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->subscribeEasyToBorrow(pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeOrder#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->subscribeOrder(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribePnl#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->subscribePnl(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeTradeRoute#1") {
  validate_keys(p, {"pFcmId", "pIbId", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pFcmId = decode_pointer<tsNCharcb>(required(p, "pFcmId"), arena);
  tsNCharcb* pIbId = decode_pointer<tsNCharcb>(required(p, "pIbId"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->subscribeTradeRoute(pFcmId, pIbId, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeUser#1") {
  validate_keys(p, {"pIbId", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pIbId = decode_pointer<tsNCharcb>(required(p, "pIbId"), arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->subscribeUser(pIbId, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.suspendInput#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->suspendInput(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsetEnvironmentVariable#1") {
  validate_keys(p, {"pKey", "pVariable"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pKey = decode_pointer<tsNCharcb>(required(p, "pKey"), arena);
  tsNCharcb* pVariable = decode_pointer<tsNCharcb>(required(p, "pVariable"), arena);
  state.native_call_started = true; int native_return = state.engine->unsetEnvironmentVariable(pKey, pVariable, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribe#1") {
  validate_keys(p, {"pExchange", "pTicker"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  state.native_call_started = true; int native_return = state.engine->unsubscribe(pExchange, pTicker, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeAutoLiquidate#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->unsubscribeAutoLiquidate(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeBracket#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->unsubscribeBracket(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeByUnderlying#1") {
  validate_keys(p, {"pUnderlying", "pExchange", "pExpiration"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pUnderlying = decode_pointer<tsNCharcb>(required(p, "pUnderlying"), arena);
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pExpiration = decode_pointer<tsNCharcb>(required(p, "pExpiration"), arena);
  state.native_call_started = true; int native_return = state.engine->unsubscribeByUnderlying(pUnderlying, pExchange, pExpiration, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeDbo#1") {
  validate_keys(p, {"pExchange", "pTicker", "bPrice", "dPrice", "pContext"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pExchange = decode_pointer<tsNCharcb>(required(p, "pExchange"), arena);
  tsNCharcb* pTicker = decode_pointer<tsNCharcb>(required(p, "pTicker"), arena);
  bool bPrice{}; decode(required(p, "bPrice"), bPrice, arena);
  double dPrice{}; decode(required(p, "dPrice"), dPrice, arena);
  void* pContext = state.context(p.value("pContext", Json()));
  state.native_call_started = true; int native_return = state.engine->unsubscribeDbo(pExchange, pTicker, bPrice, dPrice, pContext, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeEasyToBorrow#1") {
  validate_keys(p, {});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  state.native_call_started = true; int native_return = state.engine->unsubscribeEasyToBorrow(&native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeOrder#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->unsubscribeOrder(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribePnl#1") {
  validate_keys(p, {"pAccount"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pAccount = decode_pointer<AccountInfo>(required(p, "pAccount"), arena); if (!pAccount) throw std::invalid_argument("null native object: pAccount");
  state.native_call_started = true; int native_return = state.engine->unsubscribePnl(pAccount, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeTradeRoute#1") {
  validate_keys(p, {"pFcmId", "pIbId"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pFcmId = decode_pointer<tsNCharcb>(required(p, "pFcmId"), arena);
  tsNCharcb* pIbId = decode_pointer<tsNCharcb>(required(p, "pIbId"), arena);
  state.native_call_started = true; int native_return = state.engine->unsubscribeTradeRoute(pFcmId, pIbId, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeUser#1") {
  validate_keys(p, {"pIbId"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  tsNCharcb* pIbId = decode_pointer<tsNCharcb>(required(p, "pIbId"), arena);
  state.native_call_started = true; int native_return = state.engine->unsubscribeUser(pIbId, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.replayBars#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<ReplayBarParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->replayBars(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.subscribeBar#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<BarParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->subscribeBar(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
if (id == "REngine.unsubscribeBar#1") {
  validate_keys(p, {"pParams"});
  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();
  auto pParams = decode_pointer<BarParams>(required(p, "pParams"), arena); if (!pParams) throw std::invalid_argument("null native object: pParams");
  state.native_call_started = true; int native_return = state.engine->unsubscribeBar(pParams, &native_code);
  Encoding encoding(state); if (native_return == OK) {  }
  return native_result(id, native_return, native_code, std::move(outputs));
}
