{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AccountListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AccountList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AccountList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AccountUpdateInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AccountUpdate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AccountUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AgreementListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AgreementList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AgreementList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AggregatorInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Aggregator(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Aggregator#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AlertInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Alert(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Alert#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AskInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AskQuote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AskQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AssignedUserListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AssignedUserList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AssignedUserList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AutoLiquidateInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AutoLiquidate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AutoLiquidate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AuxRefDataInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.AuxRefData(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.AuxRefData#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BarInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Bar(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Bar#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BarReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BarReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BarReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  AskInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BestAskQuote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BestAskQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BidInfo pBid{};
  initialize_public(pBid);
  AskInfo pAsk{};
  initialize_public(pAsk);
  int code = -1;
  assert(cb.BestBidAskQuote(&pBid, &pAsk, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BestBidAskQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BidInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BestBidQuote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BestBidQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BidInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BidQuote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BidQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BinaryContractListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BinaryContractList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BinaryContractList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BracketReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BracketReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BracketReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BracketTierModifyInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BracketTierModify(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BracketTierModify#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  BracketInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.BracketUpdate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BracketUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderBustReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.BustReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.BustReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderCancelReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.CancelReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.CancelReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  CloseMidPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.CloseMidPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.CloseMidPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ClosePriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ClosePrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ClosePrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ClosingIndicatorInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ClosingIndicator(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ClosingIndicator#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  DboInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Dbo(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Dbo#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  DboBookRebuildInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.DboBookRebuild(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.DboBookRebuild#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  EasyToBorrowInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.EasyToBorrow(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.EasyToBorrow#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  EasyToBorrowListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.EasyToBorrowList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.EasyToBorrowList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  EndQuoteInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.EndQuote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.EndQuote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  EquityOptionStrategyListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.EquityOptionStrategyList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.EquityOptionStrategyList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ExchangeListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ExchangeList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ExchangeList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ExecutionReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ExecutionReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ExecutionReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderFailureReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.FailureReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.FailureReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderFillReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.FillReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.FillReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  HighBidPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.HighBidPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.HighBidPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  HighPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.HighPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.HighPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  HighPriceLimitInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.HighPriceLimit(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.HighPriceLimit#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  IbListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.IbList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.IbList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  InstrumentByUnderlyingInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.InstrumentByUnderlying(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.InstrumentByUnderlying#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  InstrumentSearchInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.InstrumentSearch(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.InstrumentSearch#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  LimitOrderBookInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.LimitOrderBook(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.LimitOrderBook#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  LineInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.LineUpdate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.LineUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  LowAskPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.LowAskPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.LowAskPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  LowPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.LowPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.LowPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  LowPriceLimitInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.LowPriceLimit(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.LowPriceLimit#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  MarketModeInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.MarketMode(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.MarketMode#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  MidPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.MidPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.MidPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderModifyReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.ModifyReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ModifyReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderNotCancelledReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.NotCancelledReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.NotCancelledReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderNotModifiedReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.NotModifiedReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.NotModifiedReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OpenInterestInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OpenInterest(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OpenInterest#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OpenOrderReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OpenOrderReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OpenPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OpenPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OpenPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OpeningIndicatorInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OpeningIndicator(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OpeningIndicator#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OptionListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OptionList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OptionList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderHistoryDatesInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OrderHistoryDates(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OrderHistoryDates#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.OrderReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OrderReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.OtherReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.OtherReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PasswordChangeInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.PasswordChange(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.PasswordChange#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PingInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Ping(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Ping#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PnlReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.PnlReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.PnlReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PnlInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.PnlUpdate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.PnlUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PositionExitInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.PositionExit(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.PositionExit#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  PriceIncrInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.PriceIncrUpdate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.PriceIncrUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ProductRmsListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ProductRmsList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ProductRmsList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  ProjectedSettlementPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.ProjectedSettlementPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.ProjectedSettlementPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  RefDataInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.RefData(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.RefData#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderRejectReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.RejectReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.RejectReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  SettlementPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.SettlementPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.SettlementPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  SingleOrderReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.SingleOrderReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.SingleOrderReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  SodReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.SodUpdate(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.SodUpdate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderStatusReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.StatusReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.StatusReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  StrategyInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Strategy(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Strategy#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  StrategyListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.StrategyList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.StrategyList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradeReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeReplay#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeRouteInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradeRoute(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeRoute#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeRouteListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradeRouteList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeRouteList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderTriggerPulledReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.TriggerPulledReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TriggerPulledReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderTriggerReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.TriggerReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TriggerReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradeCondition(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeCondition#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  OrderTradeCorrectReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.TradeCorrectReport(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeCorrectReport#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradePrint(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradePrint#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  TradeVolumeInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.TradeVolume(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.TradeVolume#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  UserInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.User(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.User#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  UserDefinedSpreadCreateInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.UserDefinedSpreadCreate(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.UserDefinedSpreadCreate#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  UserListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.UserList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.UserList#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  UserProfileInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.UserProfile(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.UserProfile#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  VolumeAtPriceInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.VolumeAtPrice(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.VolumeAtPrice#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  QuoteReport pReport{};
  initialize_public(pReport);
  int code = -1;
  assert(cb.Quote(&pReport, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Quote#1");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  QuoteInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Quote(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.Quote#2");
}
{
  Callbacks& cb = static_cast<Callbacks&>(*state.callbacks);
  QuoteReplayInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.QuoteReplay(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "RCallbacks.QuoteReplay#1");
}
{
  AdminCallbacks& cb = static_cast<AdminCallbacks&>(*state.admin);
  AlertInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Alert(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "AdmCallbacks.Alert#1");
}
{
  AdminCallbacks& cb = static_cast<AdminCallbacks&>(*state.admin);
  EnvironmentInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.Environment(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "AdmCallbacks.Environment#1");
}
{
  AdminCallbacks& cb = static_cast<AdminCallbacks&>(*state.admin);
  EnvironmentListInfo pInfo{};
  initialize_public(pInfo);
  int code = -1;
  assert(cb.EnvironmentList(&pInfo, nullptr, &code) == OK); assert(code == API_OK);
  assert(events.back().at("event") == "AdmCallbacks.EnvironmentList#1");
}
