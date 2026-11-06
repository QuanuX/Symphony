// Generated native object helpers.
if (type == "SearchTerm") {
  auto* value = arena.make<SearchTerm>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AccountInfo") {
  auto* value = arena.make<AccountInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AccountListInfo") {
  auto* value = arena.make<AccountListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AccountUpdateInfo") {
  auto* value = arena.make<AccountUpdateInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AggregatorInfo") {
  auto* value = arena.make<AggregatorInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AgreementInfo") {
  auto* value = arena.make<AgreementInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AgreementListInfo") {
  auto* value = arena.make<AgreementListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AlertInfo") {
  auto* value = arena.make<AlertInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AskInfo") {
  auto* value = arena.make<AskInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AssignedUserListInfo") {
  auto* value = arena.make<AssignedUserListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AutoLiquidateInfo") {
  auto* value = arena.make<AutoLiquidateInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "AuxRefDataInfo") {
  auto* value = arena.make<AuxRefDataInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BidInfo") {
  auto* value = arena.make<BidInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BinaryContractListInfo") {
  auto* value = arena.make<BinaryContractListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BracketTierModifyInfo") {
  auto* value = arena.make<BracketTierModifyInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BracketTier") {
  auto* value = arena.make<BracketTier>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BracketInfo") {
  auto* value = arena.make<BracketInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BracketReplayInfo") {
  auto* value = arena.make<BracketReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "CloseMidPriceInfo") {
  auto* value = arena.make<CloseMidPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ClosePriceInfo") {
  auto* value = arena.make<ClosePriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ClosingIndicatorInfo") {
  auto* value = arena.make<ClosingIndicatorInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "DboInfo") {
  auto* value = arena.make<DboInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "DboBookRebuildInfo") {
  auto* value = arena.make<DboBookRebuildInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EasyToBorrowInfo") {
  auto* value = arena.make<EasyToBorrowInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EasyToBorrowListInfo") {
  auto* value = arena.make<EasyToBorrowListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EndQuoteInfo") {
  auto* value = arena.make<EndQuoteInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EnvironmentVariable") {
  auto* value = arena.make<EnvironmentVariable>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EnvironmentInfo") {
  auto* value = arena.make<EnvironmentInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EnvironmentListInfo") {
  auto* value = arena.make<EnvironmentListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "EquityOptionStrategyListInfo") {
  auto* value = arena.make<EquityOptionStrategyListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ExchangeListInfo") {
  auto* value = arena.make<ExchangeListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ExecutionReplayInfo") {
  auto* value = arena.make<ExecutionReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "HighBidPriceInfo") {
  auto* value = arena.make<HighBidPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "HighPriceInfo") {
  auto* value = arena.make<HighPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "HighPriceLimitInfo") {
  auto* value = arena.make<HighPriceLimitInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "IbListInfo") {
  auto* value = arena.make<IbListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "InstrumentByUnderlyingInfo") {
  auto* value = arena.make<InstrumentByUnderlyingInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "InstrumentSearchInfo") {
  auto* value = arena.make<InstrumentSearchInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LimitOrderBookInfo") {
  auto* value = arena.make<LimitOrderBookInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LineInfo") {
  auto* value = arena.make<LineInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LowAskPriceInfo") {
  auto* value = arena.make<LowAskPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LowPriceInfo") {
  auto* value = arena.make<LowPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LowPriceLimitInfo") {
  auto* value = arena.make<LowPriceLimitInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "MarketModeInfo") {
  auto* value = arena.make<MarketModeInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "MidPriceInfo") {
  auto* value = arena.make<MidPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OpenInterestInfo") {
  auto* value = arena.make<OpenInterestInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OpenPriceInfo") {
  auto* value = arena.make<OpenPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OpeningIndicatorInfo") {
  auto* value = arena.make<OpeningIndicatorInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OptionListInfo") {
  auto* value = arena.make<OptionListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderHistoryDatesInfo") {
  auto* value = arena.make<OrderHistoryDatesInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderReplayInfo") {
  auto* value = arena.make<OrderReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PasswordChangeInfo") {
  auto* value = arena.make<PasswordChangeInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PingInfo") {
  auto* value = arena.make<PingInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PnlInfo") {
  auto* value = arena.make<PnlInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PnlReplayInfo") {
  auto* value = arena.make<PnlReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PositionExitInfo") {
  auto* value = arena.make<PositionExitInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PriceIncrInfo") {
  auto* value = arena.make<PriceIncrInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "PriceIncrRow") {
  auto* value = arena.make<PriceIncrRow>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ProductRmsInfo") {
  auto* value = arena.make<ProductRmsInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ProductRmsListInfo") {
  auto* value = arena.make<ProductRmsListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ProjectedSettlementPriceInfo") {
  auto* value = arena.make<ProjectedSettlementPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "QuoteInfo") {
  auto* value = arena.make<QuoteInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "QuoteReplayInfo") {
  auto* value = arena.make<QuoteReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "QuoteReport") {
  auto* value = arena.make<QuoteReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "RefDataInfo") {
  auto* value = arena.make<RefDataInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "RmsInfo") {
  auto* value = arena.make<RmsInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "SettlementPriceInfo") {
  auto* value = arena.make<SettlementPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "SingleOrderReplayInfo") {
  auto* value = arena.make<SingleOrderReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "StrategyLegInfo") {
  auto* value = arena.make<StrategyLegInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "StrategyInfo") {
  auto* value = arena.make<StrategyInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "StrategyListInfo") {
  auto* value = arena.make<StrategyListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "TradeInfo") {
  auto* value = arena.make<TradeInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "TradeReplayInfo") {
  auto* value = arena.make<TradeReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "TradeRouteInfo") {
  auto* value = arena.make<TradeRouteInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "TradeRouteListInfo") {
  auto* value = arena.make<TradeRouteListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "TradeVolumeInfo") {
  auto* value = arena.make<TradeVolumeInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "UserInfo") {
  auto* value = arena.make<UserInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "UserDefinedSpreadCreateInfo") {
  auto* value = arena.make<UserDefinedSpreadCreateInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "UserListInfo") {
  auto* value = arena.make<UserListInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "UserProfileInfo") {
  auto* value = arena.make<UserProfileInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "VolumeAtPriceInfo") {
  auto* value = arena.make<VolumeAtPriceInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "SodReport") {
  auto* value = arena.make<SodReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderReport") {
  auto* value = arena.make<OrderReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderBustReport") {
  auto* value = arena.make<OrderBustReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderCancelReport") {
  auto* value = arena.make<OrderCancelReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderFailureReport") {
  auto* value = arena.make<OrderFailureReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderFillReport") {
  auto* value = arena.make<OrderFillReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderModifyReport") {
  auto* value = arena.make<OrderModifyReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderNotCancelledReport") {
  auto* value = arena.make<OrderNotCancelledReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderNotModifiedReport") {
  auto* value = arena.make<OrderNotModifiedReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderRejectReport") {
  auto* value = arena.make<OrderRejectReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderStatusReport") {
  auto* value = arena.make<OrderStatusReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderTradeCorrectReport") {
  auto* value = arena.make<OrderTradeCorrectReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderTriggerPulledReport") {
  auto* value = arena.make<OrderTriggerPulledReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderTriggerReport") {
  auto* value = arena.make<OrderTriggerReport>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "OrderParams") {
  auto* value = arena.make<OrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LimitOrderParams") {
  auto* value = arena.make<LimitOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "MarketOrderParams") {
  auto* value = arena.make<MarketOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "StopLimitOrderParams") {
  auto* value = arena.make<StopLimitOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "StopMarketOrderParams") {
  auto* value = arena.make<StopMarketOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ModifyOrderParams") {
  auto* value = arena.make<ModifyOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ModifyLimitOrderParams") {
  auto* value = arena.make<ModifyLimitOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ModifyStopLimitOrderParams") {
  auto* value = arena.make<ModifyStopLimitOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ModifyStopMarketOrderParams") {
  auto* value = arena.make<ModifyStopMarketOrderParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BracketParams") {
  auto* value = arena.make<BracketParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "QuoteParams") {
  auto* value = arena.make<QuoteParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "QuoteCancelParams") {
  auto* value = arena.make<QuoteCancelParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BarParams") {
  auto* value = arena.make<BarParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "ReplayBarParams") {
  auto* value = arena.make<ReplayBarParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BarInfo") {
  auto* value = arena.make<BarInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "BarReplayInfo") {
  auto* value = arena.make<BarReplayInfo>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "REngineParams") {
  auto* value = arena.make<REngineParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
if (type == "LoginParams") {
  auto* value = arena.make<LoginParams>();
  if (p.contains("value")) decode(p.at("value"), *value, arena);
  if (action == "clear") throw std::invalid_argument("this native type has no clearHandles method");
  Encoding encoding(state); return Json{{"status","ok"},{"type",type},{"action",action},{"value",encode(*value,encoding)},{"diagnostic_adaptation",action=="dump" ? "structured-inspection" : "native-object-lifecycle"}};
}
