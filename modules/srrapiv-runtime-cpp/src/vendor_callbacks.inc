// Generated callback adapters; every overload is distinct.
class Callbacks final : public RCallbacks {
public:
  explicit Callbacks(State& s) : state(s) {}
  State& state;
  int AccountList(AccountListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AccountList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AccountList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AccountList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AccountUpdate(AccountUpdateInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AccountUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AccountUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AccountUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AgreementList(AgreementListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AgreementList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AgreementList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AgreementList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Aggregator(AggregatorInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Aggregator#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Aggregator#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Aggregator#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Alert(AlertInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Alert#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Alert#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Alert#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AskQuote(AskInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AskQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AskQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AskQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AssignedUserList(AssignedUserListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AssignedUserList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AssignedUserList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AssignedUserList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AutoLiquidate(AutoLiquidateInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AutoLiquidate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AutoLiquidate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AutoLiquidate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int AuxRefData(AuxRefDataInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.AuxRefData#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.AuxRefData#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.AuxRefData#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Bar(BarInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Bar#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Bar#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Bar#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BarReplay(BarReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BarReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BarReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BarReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BestAskQuote(AskInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BestAskQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BestAskQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BestAskQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BestBidAskQuote(BidInfo * pBid, AskInfo * pAsk, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pBid"] = encode_pointer(pBid, e);
      payload["pAsk"] = encode_pointer(pAsk, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BestBidAskQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BestBidAskQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BestBidAskQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BestBidQuote(BidInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BestBidQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BestBidQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BestBidQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BidQuote(BidInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BidQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BidQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BidQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BinaryContractList(BinaryContractListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BinaryContractList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BinaryContractList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BinaryContractList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BracketReplay(BracketReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BracketReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BracketReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BracketReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BracketTierModify(BracketTierModifyInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BracketTierModify#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BracketTierModify#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BracketTierModify#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BracketUpdate(BracketInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BracketUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BracketUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BracketUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int BustReport(OrderBustReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.BustReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.BustReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.BustReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int CancelReport(OrderCancelReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.CancelReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.CancelReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.CancelReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int CloseMidPrice(CloseMidPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.CloseMidPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.CloseMidPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.CloseMidPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ClosePrice(ClosePriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ClosePrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ClosePrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ClosePrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ClosingIndicator(ClosingIndicatorInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ClosingIndicator#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ClosingIndicator#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ClosingIndicator#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Dbo(DboInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Dbo#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Dbo#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Dbo#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int DboBookRebuild(DboBookRebuildInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.DboBookRebuild#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.DboBookRebuild#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.DboBookRebuild#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int EasyToBorrow(EasyToBorrowInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.EasyToBorrow#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.EasyToBorrow#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.EasyToBorrow#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int EasyToBorrowList(EasyToBorrowListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.EasyToBorrowList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.EasyToBorrowList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.EasyToBorrowList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int EndQuote(EndQuoteInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.EndQuote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.EndQuote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.EndQuote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int EquityOptionStrategyList(EquityOptionStrategyListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.EquityOptionStrategyList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.EquityOptionStrategyList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.EquityOptionStrategyList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ExchangeList(ExchangeListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ExchangeList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ExchangeList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ExchangeList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ExecutionReplay(ExecutionReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ExecutionReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ExecutionReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ExecutionReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int FailureReport(OrderFailureReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.FailureReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.FailureReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.FailureReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int FillReport(OrderFillReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.FillReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.FillReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.FillReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int HighBidPrice(HighBidPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.HighBidPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.HighBidPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.HighBidPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int HighPrice(HighPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.HighPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.HighPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.HighPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int HighPriceLimit(HighPriceLimitInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.HighPriceLimit#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.HighPriceLimit#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.HighPriceLimit#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int IbList(IbListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.IbList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.IbList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.IbList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int InstrumentByUnderlying(InstrumentByUnderlyingInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.InstrumentByUnderlying#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.InstrumentByUnderlying#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.InstrumentByUnderlying#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int InstrumentSearch(InstrumentSearchInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.InstrumentSearch#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.InstrumentSearch#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.InstrumentSearch#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int LimitOrderBook(LimitOrderBookInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.LimitOrderBook#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.LimitOrderBook#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.LimitOrderBook#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int LineUpdate(LineInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.LineUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.LineUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.LineUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int LowAskPrice(LowAskPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.LowAskPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.LowAskPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.LowAskPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int LowPrice(LowPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.LowPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.LowPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.LowPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int LowPriceLimit(LowPriceLimitInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.LowPriceLimit#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.LowPriceLimit#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.LowPriceLimit#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int MarketMode(MarketModeInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.MarketMode#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.MarketMode#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.MarketMode#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int MidPrice(MidPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.MidPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.MidPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.MidPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ModifyReport(OrderModifyReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ModifyReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ModifyReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ModifyReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int NotCancelledReport(OrderNotCancelledReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.NotCancelledReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.NotCancelledReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.NotCancelledReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int NotModifiedReport(OrderNotModifiedReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.NotModifiedReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.NotModifiedReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.NotModifiedReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OpenInterest(OpenInterestInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OpenInterest#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OpenInterest#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OpenInterest#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OpenOrderReplay(OrderReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OpenOrderReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OpenOrderReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OpenOrderReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OpenPrice(OpenPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OpenPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OpenPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OpenPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OpeningIndicator(OpeningIndicatorInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OpeningIndicator#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OpeningIndicator#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OpeningIndicator#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OptionList(OptionListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OptionList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OptionList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OptionList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OrderHistoryDates(OrderHistoryDatesInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OrderHistoryDates#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OrderHistoryDates#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OrderHistoryDates#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OrderReplay(OrderReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OrderReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OrderReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OrderReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int OtherReport(OrderReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.OtherReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.OtherReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.OtherReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int PasswordChange(PasswordChangeInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.PasswordChange#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.PasswordChange#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.PasswordChange#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Ping(PingInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Ping#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Ping#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Ping#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int PnlReplay(PnlReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.PnlReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.PnlReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.PnlReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int PnlUpdate(PnlInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.PnlUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.PnlUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.PnlUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int PositionExit(PositionExitInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.PositionExit#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.PositionExit#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.PositionExit#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int PriceIncrUpdate(PriceIncrInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.PriceIncrUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.PriceIncrUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.PriceIncrUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ProductRmsList(ProductRmsListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ProductRmsList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ProductRmsList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ProductRmsList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int ProjectedSettlementPrice(ProjectedSettlementPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.ProjectedSettlementPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.ProjectedSettlementPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.ProjectedSettlementPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int RefData(RefDataInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.RefData#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.RefData#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.RefData#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int RejectReport(OrderRejectReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.RejectReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.RejectReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.RejectReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int SettlementPrice(SettlementPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.SettlementPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.SettlementPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.SettlementPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int SingleOrderReplay(SingleOrderReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.SingleOrderReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.SingleOrderReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.SingleOrderReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int SodUpdate(SodReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.SodUpdate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.SodUpdate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.SodUpdate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int StatusReport(OrderStatusReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.StatusReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.StatusReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.StatusReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Strategy(StrategyInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Strategy#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Strategy#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Strategy#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int StrategyList(StrategyListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.StrategyList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.StrategyList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.StrategyList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeReplay(TradeReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeRoute(TradeRouteInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeRoute#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeRoute#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeRoute#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeRouteList(TradeRouteListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeRouteList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeRouteList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeRouteList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TriggerPulledReport(OrderTriggerPulledReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TriggerPulledReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TriggerPulledReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TriggerPulledReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TriggerReport(OrderTriggerReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TriggerReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TriggerReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TriggerReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeCondition(TradeInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeCondition#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeCondition#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeCondition#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeCorrectReport(OrderTradeCorrectReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeCorrectReport#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeCorrectReport#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeCorrectReport#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradePrint(TradeInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradePrint#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradePrint#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradePrint#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int TradeVolume(TradeVolumeInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.TradeVolume#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.TradeVolume#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.TradeVolume#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int User(UserInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.User#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.User#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.User#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int UserDefinedSpreadCreate(UserDefinedSpreadCreateInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.UserDefinedSpreadCreate#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.UserDefinedSpreadCreate#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.UserDefinedSpreadCreate#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int UserList(UserListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.UserList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.UserList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.UserList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int UserProfile(UserProfileInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.UserProfile#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.UserProfile#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.UserProfile#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int VolumeAtPrice(VolumeAtPriceInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.VolumeAtPrice#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.VolumeAtPrice#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.VolumeAtPrice#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Quote(QuoteReport * pReport, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pReport"] = encode_pointer(pReport, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Quote#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Quote#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Quote#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Quote(QuoteInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.Quote#2", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.Quote#2", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.Quote#2", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int QuoteReplay(QuoteReplayInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("RCallbacks.QuoteReplay#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("RCallbacks.QuoteReplay#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("RCallbacks.QuoteReplay#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
};
class AdminCallbacks final : public AdmCallbacks {
public:
  explicit AdminCallbacks(State& s) : state(s) {}
  State& state;
  int Alert(AlertInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("AdmCallbacks.Alert#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("AdmCallbacks.Alert#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("AdmCallbacks.Alert#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int Environment(EnvironmentInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("AdmCallbacks.Environment#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("AdmCallbacks.Environment#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("AdmCallbacks.Environment#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
  int EnvironmentList(EnvironmentListInfo * pInfo, void * pContext, int * aiCode) override {
    try { Encoding e(state); Json payload = Json::object();
      payload["pInfo"] = encode_pointer(pInfo, e);
      payload["pContext"] = encode_pointer(pContext, e);
      state.deliver("AdmCallbacks.EnvironmentList#1", std::move(payload));
    } catch (const std::exception&) { state.encoding_failure("AdmCallbacks.EnvironmentList#1", "callback payload failed bounded encoding"); } catch (...) { state.encoding_failure("AdmCallbacks.EnvironmentList#1", "non-standard callback exception"); }
    if (aiCode) *aiCode = API_OK; return OK;
  }
};
