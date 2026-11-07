// Initialize public interchange fields explicitly; native constructors still own private state.
void initialize_public(SearchTerm& value);
void initialize_public(AccountInfo& value);
void initialize_public(AccountListInfo& value);
void initialize_public(AccountUpdateInfo& value);
void initialize_public(AggregatorInfo& value);
void initialize_public(AgreementInfo& value);
void initialize_public(AgreementListInfo& value);
void initialize_public(AlertInfo& value);
void initialize_public(AskInfo& value);
void initialize_public(AssignedUserListInfo& value);
void initialize_public(AutoLiquidateInfo& value);
void initialize_public(AuxRefDataInfo& value);
void initialize_public(BidInfo& value);
void initialize_public(BinaryContractListInfo& value);
void initialize_public(BracketTierModifyInfo& value);
void initialize_public(BracketTier& value);
void initialize_public(BracketInfo& value);
void initialize_public(BracketReplayInfo& value);
void initialize_public(CloseMidPriceInfo& value);
void initialize_public(ClosePriceInfo& value);
void initialize_public(ClosingIndicatorInfo& value);
void initialize_public(DboInfo& value);
void initialize_public(DboBookRebuildInfo& value);
void initialize_public(EasyToBorrowInfo& value);
void initialize_public(EasyToBorrowListInfo& value);
void initialize_public(EndQuoteInfo& value);
void initialize_public(EnvironmentVariable& value);
void initialize_public(EnvironmentInfo& value);
void initialize_public(EnvironmentListInfo& value);
void initialize_public(EquityOptionStrategyListInfo& value);
void initialize_public(ExchangeListInfo& value);
void initialize_public(ExecutionReplayInfo& value);
void initialize_public(HighBidPriceInfo& value);
void initialize_public(HighPriceInfo& value);
void initialize_public(HighPriceLimitInfo& value);
void initialize_public(IbListInfo& value);
void initialize_public(InstrumentByUnderlyingInfo& value);
void initialize_public(InstrumentSearchInfo& value);
void initialize_public(LimitOrderBookInfo& value);
void initialize_public(LineInfo& value);
void initialize_public(LowAskPriceInfo& value);
void initialize_public(LowPriceInfo& value);
void initialize_public(LowPriceLimitInfo& value);
void initialize_public(MarketModeInfo& value);
void initialize_public(MidPriceInfo& value);
void initialize_public(OpenInterestInfo& value);
void initialize_public(OpenPriceInfo& value);
void initialize_public(OpeningIndicatorInfo& value);
void initialize_public(OptionListInfo& value);
void initialize_public(OrderHistoryDatesInfo& value);
void initialize_public(OrderReplayInfo& value);
void initialize_public(PasswordChangeInfo& value);
void initialize_public(PingInfo& value);
void initialize_public(PnlInfo& value);
void initialize_public(PnlReplayInfo& value);
void initialize_public(PositionExitInfo& value);
void initialize_public(PriceIncrInfo& value);
void initialize_public(PriceIncrRow& value);
void initialize_public(ProductRmsInfo& value);
void initialize_public(ProductRmsListInfo& value);
void initialize_public(ProjectedSettlementPriceInfo& value);
void initialize_public(QuoteInfo& value);
void initialize_public(QuoteReplayInfo& value);
void initialize_public(QuoteReport& value);
void initialize_public(RefDataInfo& value);
void initialize_public(RmsInfo& value);
void initialize_public(SettlementPriceInfo& value);
void initialize_public(SingleOrderReplayInfo& value);
void initialize_public(StrategyLegInfo& value);
void initialize_public(StrategyInfo& value);
void initialize_public(StrategyListInfo& value);
void initialize_public(TradeInfo& value);
void initialize_public(TradeReplayInfo& value);
void initialize_public(TradeRouteInfo& value);
void initialize_public(TradeRouteListInfo& value);
void initialize_public(TradeVolumeInfo& value);
void initialize_public(UserInfo& value);
void initialize_public(UserDefinedSpreadCreateInfo& value);
void initialize_public(UserListInfo& value);
void initialize_public(UserProfileInfo& value);
void initialize_public(VolumeAtPriceInfo& value);
void initialize_public(SodReport& value);
void initialize_public(OrderReport& value);
void initialize_public(OrderBustReport& value);
void initialize_public(OrderCancelReport& value);
void initialize_public(OrderFailureReport& value);
void initialize_public(OrderFillReport& value);
void initialize_public(OrderModifyReport& value);
void initialize_public(OrderNotCancelledReport& value);
void initialize_public(OrderNotModifiedReport& value);
void initialize_public(OrderRejectReport& value);
void initialize_public(OrderStatusReport& value);
void initialize_public(OrderTradeCorrectReport& value);
void initialize_public(OrderTriggerPulledReport& value);
void initialize_public(OrderTriggerReport& value);
void initialize_public(OrderParams& value);
void initialize_public(LimitOrderParams& value);
void initialize_public(MarketOrderParams& value);
void initialize_public(StopLimitOrderParams& value);
void initialize_public(StopMarketOrderParams& value);
void initialize_public(ModifyOrderParams& value);
void initialize_public(ModifyLimitOrderParams& value);
void initialize_public(ModifyStopLimitOrderParams& value);
void initialize_public(ModifyStopMarketOrderParams& value);
void initialize_public(BracketParams& value);
void initialize_public(QuoteParams& value);
void initialize_public(QuoteCancelParams& value);
void initialize_public(BarParams& value);
void initialize_public(ReplayBarParams& value);
void initialize_public(BarInfo& value);
void initialize_public(BarReplayInfo& value);
void initialize_public(REngineParams& value);
void initialize_public(LoginParams& value);
void initialize_public(SearchTerm& value) {
  value.eField = {};
  value.eOperator = {};
  value.sTerm = {};
  value.bCaseSensitive = {};
}
void initialize_public(AccountInfo& value) {
  value.sFcmId = {};
  value.sIbId = {};
  value.sAccountId = {};
  value.sAccountName = {};
  value.iCreationSsboe = {};
  value.iCreationUsecs = {};
  value.pRmsInfo = {};
  value.pAutoLiquidateInfo = {};
}
void initialize_public(AccountListInfo& value) {
  value.asAccountInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(AccountUpdateInfo& value) {
  initialize_public(value.oAccount);
  value.sAction = {};
  value.sUser = {};
}
void initialize_public(AggregatorInfo& value) {
  value.iRpCode = {};
}
void initialize_public(AgreementInfo& value) {
  value.sTitle = {};
  value.sStatus = {};
  value.bMandatory = {};
  value.iPublicationSsboe = {};
  value.iInForceSsboe = {};
  value.sEndUserAcceptanceStatus = {};
  value.iEndUserAcceptanceSsboe = {};
  value.sMarketDataUsageCapacity = {};
}
void initialize_public(AgreementListInfo& value) {
  value.bAccepted = {};
  value.asAgreementInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(AlertInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sMessage = {};
  value.iAlertType = {};
  value.iConnectionId = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(AskInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.llSize = {};
  value.bSizeFlag = {};
  value.llImpliedSize = {};
  value.iNumOrders = {};
  value.bLeanPriceFlag = {};
  value.dLeanPrice = {};
  value.iUpdateType = {};
  value.iType = {};
  value.iConnId = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(AssignedUserListInfo& value) {
  initialize_public(value.oAccount);
  value.pUserArray = {};
  value.iUserArrayLen = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(AutoLiquidateInfo& value) {
  initialize_public(value.oAccount);
  value.eAutoLiquidateThresholdCurrentValue = {};
  value.dAutoLiquidateThresholdCurrentValue = {};
  value.eAutoLiquidateThresholdCurrentValueSsboe = {};
  value.iAutoLiquidateThresholdCurrentValueSsboe = {};
  value.iCallbackType = {};
}
void initialize_public(AuxRefDataInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sFirstNoticeDate = {};
  value.sLastNoticeDate = {};
  value.sFirstTradingDate = {};
  value.sLastTradingDate = {};
  value.sFirstDeliveryDate = {};
  value.sLastDeliveryDate = {};
  value.sFirstPositionDate = {};
  value.sLastPositionDate = {};
  value.sSettlementMethod = {};
  value.sUnitOfMeasure = {};
  value.bUnitOfMeasureQty = {};
  value.dUnitOfMeasureQty = {};
  value.iSsboe = {};
  value.iRpCode = {};
  value.iType = {};
}
void initialize_public(BidInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.llSize = {};
  value.bSizeFlag = {};
  value.llImpliedSize = {};
  value.iNumOrders = {};
  value.bLeanPriceFlag = {};
  value.dLeanPrice = {};
  value.iUpdateType = {};
  value.iType = {};
  value.iConnId = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(BinaryContractListInfo& value) {
  value.pExchangeArray = {};
  value.iExchangeArrayLen = {};
  value.pProductCodeArray = {};
  value.iProductCodeArrayLen = {};
  value.pBinaryContractTypeArray = {};
  value.iBinaryContractTypeArrayLen = {};
  value.pPeriodCodeArray = {};
  value.iPeriodCodeArrayLen = {};
  value.pExpirationTimeArray = {};
  value.iExpirationTimeArrayLen = {};
  value.asRefDataInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(BracketTierModifyInfo& value) {
  initialize_public(value.oAccount);
  value.sOrderNum = {};
  value.bTarget = {};
  value.iOldTickOffset = {};
  value.iNewTickOffset = {};
  value.pContext = {};
  value.iRpCode = {};
}
void initialize_public(BracketTier& value) {
  value.iTicks = {};
  value.llQty = {};
  value.llQtyReleased = {};
}
void initialize_public(BracketInfo& value) {
  initialize_public(value.oAccount);
  value.sOrderNum = {};
  value.sTargetsToMarketWhenTouched = {};
  value.bTargetsToMarketAt = {};
  value.iTargetsToMarketAtSsboe = {};
  value.iTargetsToMarketAtUsecs = {};
  value.bTargetsToMarketAfterPartialFillInSeconds = {};
  value.iTargetsToMarketAfterPartialFillInSeconds = {};
  value.iTargetArrayLen = {};
  value.asTargetArray = {};
  value.sStopsToMarketOnReject = {};
  value.bStopsToMarketAt = {};
  value.iStopsToMarketAtSsboe = {};
  value.iStopsToMarketAtUsecs = {};
  value.bTrailingStop = {};
  value.iTrailByPriceId = {};
  value.bTrailingStopTriggerTicks = {};
  value.iTrailingStopTriggerTicks = {};
  value.iStopArrayLen = {};
  value.asStopArray = {};
}
void initialize_public(BracketReplayInfo& value) {
  initialize_public(value.oAccount);
  value.iArrayLen = {};
  value.asBracketArray = {};
  value.iRpCode = {};
}
void initialize_public(CloseMidPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.bClosePrice = {};
  value.dClosePrice = {};
  value.sClosePriceDate = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(ClosePriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(ClosingIndicatorInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(DboInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sSide = {};
  value.dPrice = {};
  value.llSize = {};
  value.sDboUpdateType = {};
  value.sPriority = {};
  value.sExchOrdId = {};
  value.bPreviousPrice = {};
  value.dPreviousPrice = {};
  value.iSourceSsboe = {};
  value.iSourceUsecs = {};
  value.iSourceNsecs = {};
  value.iJopSsboe = {};
  value.iJopNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.iType = {};
  value.pContext = {};
  value.bPrice = {};
  value.bSize = {};
}
void initialize_public(DboBookRebuildInfo& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.bPrice = {};
  value.dPrice = {};
  value.pContext = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(EasyToBorrowInfo& value) {
  value.sBrokerDealer = {};
  value.sTicker = {};
  value.sDescription = {};
  value.llQtyAvailable = {};
  value.llQtyNeeded = {};
  value.bBorrowable = {};
  value.iType = {};
  value.pContext = {};
}
void initialize_public(EasyToBorrowListInfo& value) {
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(EndQuoteInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.iUpdateType = {};
  value.iType = {};
  value.iConnId = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(EnvironmentVariable& value) {
  value.sName = {};
  value.sValue = {};
}
void initialize_public(EnvironmentInfo& value) {
  value.sKey = {};
  value.asVariableArray = {};
  value.iArrayLen = {};
  value.pContext = {};
}
void initialize_public(EnvironmentListInfo& value) {
  value.asKeyArray = {};
  value.iArrayLen = {};
  value.pContext = {};
}
void initialize_public(EquityOptionStrategyListInfo& value) {
  value.pExchangeArray = {};
  value.iExchangeArrayLen = {};
  value.pUnderlyingArray = {};
  value.iUnderlyingArrayLen = {};
  value.pStrategyTypeArray = {};
  value.iStrategyTypeArrayLen = {};
  value.pExpirationArray = {};
  value.iExpirationArrayLen = {};
  value.pTickerArray = {};
  value.iTickerArrayLen = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(ExchangeListInfo& value) {
  value.asExchangeArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(ExecutionReplayInfo& value) {
  initialize_public(value.oAccount);
  value.asFillReportArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(HighBidPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(HighPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(HighPriceLimitInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(IbListInfo& value) {
  value.pIbIdArray = {};
  value.iIbIdArrayLen = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(InstrumentByUnderlyingInfo& value) {
  value.pUnderlyingArray = {};
  value.iUnderlyingArrayLen = {};
  value.pExchangeArray = {};
  value.iExchangeArrayLen = {};
  value.pExpirationArray = {};
  value.iExpirationArrayLen = {};
  value.asRefDataInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(InstrumentSearchInfo& value) {
  value.sExchange = {};
  value.aoTerms = {};
  value.iNumTerms = {};
  value.asArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(LimitOrderBookInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.adAskPriceArray = {};
  value.allAskSizeArray = {};
  value.allAskImpliedSizeArray = {};
  value.aiAskNumOrdersArray = {};
  value.aiAskSsboeArray = {};
  value.aiAskUsecsArray = {};
  value.iAskArrayLen = {};
  value.adBidPriceArray = {};
  value.allBidSizeArray = {};
  value.allBidImpliedSizeArray = {};
  value.aiBidNumOrdersArray = {};
  value.aiBidSsboeArray = {};
  value.aiBidUsecsArray = {};
  value.iBidArrayLen = {};
  value.iType = {};
  value.iConnId = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(LineInfo& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.sOrderNum = {};
  value.sOriginalOrderNum = {};
  value.sInitialSequenceNumber = {};
  value.sCurrentSequenceNumber = {};
  value.sExchOrdId = {};
  value.sTickerPlantExchOrdId = {};
  value.sOmnibusAccount = {};
  value.sOrderType = {};
  value.sOriginalOrderType = {};
  value.sBuySellType = {};
  value.sOrderDuration = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.sTradeRoute = {};
  value.sRoutingInstructions = {};
  value.sOrderOperationType = {};
  value.sBracketType = {};
  value.sLinkedOrderNums = {};
  value.sOriginatorWindowName = {};
  value.sWindowName = {};
  value.sReleaseCommand = {};
  value.bReleaseSsboe = {};
  value.iReleaseSsboe = {};
  value.bReleaseUsecs = {};
  value.iReleaseUsecs = {};
  value.bCancelSsboe = {};
  value.iCancelSsboe = {};
  value.bCancelUsecs = {};
  value.iCancelUsecs = {};
  value.bCancelAfter = {};
  value.iCancelAfter = {};
  value.bToMarketAt = {};
  value.iToMarketAtSsboe = {};
  value.iToMarketAtUsecs = {};
  value.bToMarketAfterPartialFillInSeconds = {};
  value.iToMarketAfterPartialFillInSeconds = {};
  value.sToMarketOnReject = {};
  value.sToMarketWhenLimitPriceTouched = {};
  value.sStatus = {};
  value.sCompletionReason = {};
  value.sRemarks = {};
  value.sText = {};
  value.sTag = {};
  value.sUserTag = {};
  value.sUserMsg = {};
  value.pContext = {};
  value.sUser = {};
  value.iGatewayBufferBytesBehind = {};
  value.iGatewaySsboe = {};
  value.iGatewayUsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.llQuantityToFill = {};
  value.llMinQty = {};
  value.llMaxShowQty = {};
  value.llFilled = {};
  value.llUnfilled = {};
  value.llCancelled = {};
  value.llReleasePendingSize = {};
  value.llCancelPendingSize = {};
  value.llModifyPendingSize = {};
  value.dPriceToFill = {};
  value.bPriceToFillFlag = {};
  value.dAvgFillPrice = {};
  value.bAvgFillPriceFlag = {};
  value.dTriggerPrice = {};
  value.bTriggerPriceFlag = {};
  value.sTriggerTicker = {};
  value.sTriggerExchange = {};
  value.iTriggerPriceId = {};
  value.iTriggerCmpOperator = {};
  value.iTrailByTicks = {};
  value.iTrailByPriceId = {};
  value.iType = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(LowAskPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(LowPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(LowPriceLimitInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(MarketModeInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sMarketMode = {};
  value.sReason = {};
  value.sEvent = {};
  value.iType = {};
  value.iSourceSsboe = {};
  value.iSourceUsecs = {};
  value.iSourceNsecs = {};
  value.iJopSsboe = {};
  value.iJopNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(MidPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.bLastPrice = {};
  value.dLastPrice = {};
  value.bOpenPrice = {};
  value.dOpenPrice = {};
  value.bHighPrice = {};
  value.dHighPrice = {};
  value.bLowPrice = {};
  value.dLowPrice = {};
  value.bNetChange = {};
  value.dNetChange = {};
  value.bPercentChange = {};
  value.dPercentChange = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(OpenInterestInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.llQuantity = {};
  value.bQuantityFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(OpenPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(OpeningIndicatorInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.llVolume = {};
  value.bVolumeFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(OptionListInfo& value) {
  value.pExchangeArray = {};
  value.iExchangeArrayLen = {};
  value.pProductCodeArray = {};
  value.iProductCodeArrayLen = {};
  value.pExpirationCCYYMMArray = {};
  value.iExpirationCCYYMMArrayLen = {};
  value.asRefDataInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(OrderHistoryDatesInfo& value) {
  value.asDateArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(OrderReplayInfo& value) {
  initialize_public(value.oAccount);
  value.sDate = {};
  value.asLineInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(PasswordChangeInfo& value) {
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(PingInfo& value) {
  value.iConnId = {};
  value.dElapsedSeconds = {};
  value.pContext = {};
}
void initialize_public(PnlInfo& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.eOpenPnl = {};
  value.dOpenPnl = {};
  value.eClosedPnl = {};
  value.dClosedPnl = {};
  value.eCashOnHand = {};
  value.dCashOnHand = {};
  value.eAccountBalance = {};
  value.dAccountBalance = {};
  value.eMarginBalance = {};
  value.dMarginBalance = {};
  value.eReservedMargin = {};
  value.dReservedMargin = {};
  value.eAvailableBuyingPower = {};
  value.dAvailableBuyingPower = {};
  value.eReservedBuyingPower = {};
  value.dReservedBuyingPower = {};
  value.eUsedBuyingPower = {};
  value.dUsedBuyingPower = {};
  value.ePosition = {};
  value.llPosition = {};
  value.eBuyQty = {};
  value.llBuyQty = {};
  value.eSellQty = {};
  value.llSellQty = {};
  value.eAvgOpenFillPrice = {};
  value.dAvgOpenFillPrice = {};
  value.eImpliedNetQty = {};
  value.llImpliedNetQty = {};
  value.eImpliedMarginReserved = {};
  value.dImpliedMarginReserved = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.bPossOutOfOrder = {};
  value.eLongExposure = {};
  value.llLongExposure = {};
  value.eShortExposure = {};
  value.llShortExposure = {};
  value.eBuyWorkingQty = {};
  value.llBuyWorkingQty = {};
  value.eSellWorkingQty = {};
  value.llSellWorkingQty = {};
}
void initialize_public(PnlReplayInfo& value) {
  initialize_public(value.oAccount);
  value.asPnlInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(PositionExitInfo& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.sEntryType = {};
  value.sTag = {};
  value.sTradingAlgorithm = {};
  value.sUserMsg = {};
  value.sWindowName = {};
  value.pContext = {};
  value.iRpCode = {};
}
void initialize_public(PriceIncrInfo& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.iArrayLen = {};
  value.asPriceIncrArray = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(PriceIncrRow& value) {
  value.dPriceIncr = {};
  value.iPrecision = {};
  value.iFirstOperator = {};
  value.dFirstPrice = {};
  value.iLastOperator = {};
  value.dLastPrice = {};
}
void initialize_public(ProductRmsInfo& value) {
  initialize_public(value.oAccount);
  value.sProductCode = {};
  value.bBuyLimit = {};
  value.llBuyLimit = {};
  value.bSellLimit = {};
  value.llSellLimit = {};
  value.bLossLimit = {};
  value.dLossLimit = {};
  value.bMaxOrderQty = {};
  value.llMaxOrderQty = {};
  value.bCommissionFillRate = {};
  value.dCommissionFillRate = {};
  value.bBuyLongMarginRate = {};
  value.dBuyLongMarginRate = {};
  value.bSellShortMarginRate = {};
  value.dSellShortMarginRate = {};
}
void initialize_public(ProductRmsListInfo& value) {
  value.asProductRmsInfoArray = {};
  value.iArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(ProjectedSettlementPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(QuoteInfo& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.sQuoteId = {};
  value.sQuoteMsgId = {};
  value.sStatus = {};
  value.sText = {};
  value.bBid = {};
  value.dBidPriceToFill = {};
  value.llBidQuantityToFill = {};
  value.bAsk = {};
  value.dAskPriceToFill = {};
  value.llAskQuantityToFill = {};
  value.sUserMsg = {};
  value.sUser = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.iType = {};
}
void initialize_public(QuoteReplayInfo& value) {
  initialize_public(value.oAccount);
  value.pContext = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(QuoteReport& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.sQuoteId = {};
  value.sQuoteMsgId = {};
  value.sReportType = {};
  value.sReportId = {};
  value.sRemarks = {};
  value.sUser = {};
  value.sUserMsg = {};
  value.bBid = {};
  value.dBidPriceToFill = {};
  value.llBidQuantityToFill = {};
  value.bAsk = {};
  value.dAskPriceToFill = {};
  value.llAskQuantityToFill = {};
  value.bFill = {};
  value.sFillSide = {};
  value.dFillPrice = {};
  value.llFillSize = {};
  value.iConnId = {};
  value.iSourceSsboe = {};
  value.iSourceNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.iType = {};
}
void initialize_public(RefDataInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sDescription = {};
  value.sExpiration = {};
  value.sProductCode = {};
  value.sInstrumentType = {};
  value.sIsTradable = {};
  value.sBinaryContractType = {};
  value.sPutCallIndicator = {};
  value.sUnderlying = {};
  value.sTradingTicker = {};
  value.sTradingExchange = {};
  value.sOpenTime = {};
  value.sPeriodCode = {};
  value.sExpirationTime = {};
  value.sStrikeOperator = {};
  value.bCapPrice = {};
  value.dCapPrice = {};
  value.bFloorPrice = {};
  value.dFloorPrice = {};
  value.bSinglePointValue = {};
  value.dSinglePointValue = {};
  value.sCurrency = {};
  value.bMaxPriceVariation = {};
  value.dMaxPriceVariation = {};
  value.bStrikePrice = {};
  value.dStrikePrice = {};
  value.bMinSizeIncrement = {};
  value.llMinSizeIncrement = {};
  value.bSizeMultiplier = {};
  value.dSizeMultiplier = {};
  value.iSizeMultiplierPrecision = {};
  value.iSsboe = {};
  value.iType = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(RmsInfo& value) {
  value.sStatus = {};
  value.sCurrency = {};
  value.llBuyLimit = {};
  value.llSellLimit = {};
  value.llMaxOrderQty = {};
  value.dMinAccountBalance = {};
  value.dLossLimit = {};
  value.bCheckMinAccountBalance = {};
  value.bMinMarginBalance = {};
  value.dMinMarginBalance = {};
  value.bDefaultCommission = {};
  value.dDefaultCommission = {};
  value.sAlgorithm = {};
  value.sAutoLiquidate = {};
  value.sAutoLiquidateCriteria = {};
  value.bAutoLiquidateThreshold = {};
  value.dAutoLiquidateThreshold = {};
  value.bAutoLiquidateMaxMinAccountBalance = {};
  value.dAutoLiquidateMaxMinAccountBalance = {};
  value.sDisableOnAutoLiquidate = {};
  value.bAutoLiquidateAtSsm = {};
  value.iAutoLiquidateAtSsm = {};
  value.sAutoLiquidateIgnoreOpenPnlWhenTrailing = {};
}
void initialize_public(SettlementPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.sPriceType = {};
  value.sDate = {};
  value.iType = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(SingleOrderReplayInfo& value) {
  initialize_public(value.oAccount);
  value.sOrderNum = {};
  value.sDate = {};
  value.pContext = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(StrategyLegInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sProductCode = {};
  value.sInstrumentType = {};
  value.iRatio = {};
}
void initialize_public(StrategyInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.sProductCode = {};
  value.sType = {};
  value.asLegArray = {};
  value.iLegArrayLen = {};
  value.pContext = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(StrategyListInfo& value) {
  value.pExchangeArray = {};
  value.iExchangeArrayLen = {};
  value.pProductCodeArray = {};
  value.iProductCodeArrayLen = {};
  value.pStrategyTypeArray = {};
  value.iStrategyTypeArrayLen = {};
  value.pExpirationCCYYMMArray = {};
  value.iExpirationCCYYMMArrayLen = {};
  value.pTickerArray = {};
  value.iTickerArrayLen = {};
  value.iRpCode = {};
  value.sRpCode = {};
  value.pContext = {};
}
void initialize_public(TradeInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.dPrice = {};
  value.bPriceFlag = {};
  value.llSize = {};
  value.dNetChange = {};
  value.bNetChangeFlag = {};
  value.dPercentChange = {};
  value.bPercentChangeFlag = {};
  value.sAggressorSide = {};
  value.sCondition = {};
  value.sAggressorExchOrdId = {};
  value.sExchOrdId = {};
  value.bVolumeBoughtFlag = {};
  value.llVolumeBought = {};
  value.bVolumeSoldFlag = {};
  value.llVolumeSold = {};
  value.dVwap = {};
  value.bVwapFlag = {};
  value.dVwapLong = {};
  value.bVwapLongFlag = {};
  value.iType = {};
  value.iConnId = {};
  value.iSourceSsboe = {};
  value.iSourceUsecs = {};
  value.iSourceNsecs = {};
  value.iJopSsboe = {};
  value.iJopNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(TradeReplayInfo& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.iStartSsboe = {};
  value.iEndSsboe = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(TradeRouteInfo& value) {
  value.sFcmId = {};
  value.sIbId = {};
  value.sExchange = {};
  value.sTradeRoute = {};
  value.sStatus = {};
  value.sDefault = {};
  value.iType = {};
  value.pContext = {};
}
void initialize_public(TradeRouteListInfo& value) {
  value.asTradeRouteInfoArray = {};
  value.iArrayLen = {};
  value.pContext = {};
  value.iRpCode = {};
}
void initialize_public(TradeVolumeInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.bTotalVolumeFlag = {};
  value.llTotalVolume = {};
  value.iType = {};
  value.iSourceSsboe = {};
  value.iSourceUsecs = {};
  value.iSourceNsecs = {};
  value.iJopSsboe = {};
  value.iJopNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
}
void initialize_public(UserInfo& value) {
  value.sFcmId = {};
  value.sIbId = {};
  value.sUser = {};
  value.sUserType = {};
  value.sStatus = {};
  value.sFirstName = {};
  value.sLastName = {};
  value.sAddress1 = {};
  value.sAddress2 = {};
  value.sCity = {};
  value.sStateProvinceRegion = {};
  value.sPostalCode = {};
  value.sCountry = {};
  value.sEmail = {};
  value.iCreationSsboe = {};
  value.iCreationUsecs = {};
  value.iFirstLoginSsboe = {};
  value.iFirstLoginUsecs = {};
  value.iLastLoginSsboe = {};
  value.iLastLoginUsecs = {};
  value.sLastLoginResult = {};
  value.sLastLoginRemarks = {};
  value.iDisableSsboe = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.sAction = {};
  value.iCallbackType = {};
  value.pContext = {};
}
void initialize_public(UserDefinedSpreadCreateInfo& value) {
  value.sText = {};
  value.sTicker = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(UserListInfo& value) {
  value.sIbId = {};
  value.sUserType = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(UserProfileInfo& value) {
  value.sUser = {};
  value.sUserType = {};
  value.sFirstName = {};
  value.sLastName = {};
  value.sAddress1 = {};
  value.sAddress2 = {};
  value.sCity = {};
  value.sStateProvinceRegion = {};
  value.sPostalCode = {};
  value.sCountry = {};
  value.sResidencePhone = {};
  value.sWorkPhone = {};
  value.sMobilePhone = {};
  value.sFaxPhone = {};
  value.sEmail = {};
  value.iCurrentHistorySessionCount = {};
  value.iCurrentOrdersSessionCount = {};
  value.iCurrentPnlSessionCount = {};
  value.iCurrentRepositorySessionCount = {};
  value.iCurrentMarketDataSessionCount = {};
  value.iPeakHistorySessionCount = {};
  value.iPeakOrdersSessionCount = {};
  value.iPeakPnlSessionCount = {};
  value.iPeakRepositorySessionCount = {};
  value.iPeakMarketDataSessionCount = {};
  value.iMaxHistorySessionCount = {};
  value.iMaxOrdersSessionCount = {};
  value.iMaxPnlSessionCount = {};
  value.iMaxRepositorySessionCount = {};
  value.iMaxMarketDataSessionCount = {};
  value.iConnId = {};
  value.iRpCode = {};
  value.pContext = {};
}
void initialize_public(VolumeAtPriceInfo& value) {
  value.sTicker = {};
  value.sExchange = {};
  value.pContext = {};
  value.adPriceArray = {};
  value.allVolumeArray = {};
  value.iArrayLen = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.iRpCode = {};
}
void initialize_public(SodReport& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.dPrevClosePrice = {};
  value.llCarriedSize = {};
  value.iSsboe = {};
  value.sReportType = {};
}
void initialize_public(OrderReport& value) {
  initialize_public(value.oAccount);
  value.sTicker = {};
  value.sExchange = {};
  value.sOrderNum = {};
  value.sOriginalOrderNum = {};
  value.sInitialSequenceNumber = {};
  value.sCurrentSequenceNumber = {};
  value.sExchOrdId = {};
  value.sTickerPlantExchOrdId = {};
  value.sOmnibusAccount = {};
  value.sPositionDisposition = {};
  value.sOrderType = {};
  value.sOriginalOrderType = {};
  value.sBuySellType = {};
  value.sOrderDuration = {};
  value.sEntryType = {};
  value.sLinkedOrderNums = {};
  value.dPriceToFill = {};
  value.bPriceToFillFlag = {};
  value.dTriggerPrice = {};
  value.bTriggerPriceFlag = {};
  value.sTriggerTicker = {};
  value.sTriggerExchange = {};
  value.iTriggerCmpOperator = {};
  value.iTriggerPriceId = {};
  value.dFillPrice = {};
  value.bFillPriceFlag = {};
  value.llFillSize = {};
  value.llTotalFilled = {};
  value.llTotalUnfilled = {};
  value.dAvgFillPrice = {};
  value.bAvgFillPriceFlag = {};
  value.sReportId = {};
  value.sReportType = {};
  value.sText = {};
  value.sWindowName = {};
  value.sOriginatorWindowName = {};
  value.sTag = {};
  value.sUserTag = {};
  value.sUserMsg = {};
  value.pContext = {};
  value.sUser = {};
  value.iConnId = {};
  value.iGatewayBufferBytesBehind = {};
  value.iGatewaySsboe = {};
  value.iGatewayUsecs = {};
  value.iSourceSsboe = {};
  value.iSourceNsecs = {};
  value.iSsboe = {};
  value.iUsecs = {};
  value.iType = {};
}
void initialize_public(OrderBustReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.sFillType = {};
  value.sTradeDate = {};
  value.sSettlementDate = {};
  value.sSettlementCurrency = {};
}
void initialize_public(OrderCancelReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llCancelledSize = {};
}
void initialize_public(OrderFailureReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llCancelledSize = {};
  value.sStatus = {};
}
void initialize_public(OrderFillReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.sFillType = {};
  value.sTradeDate = {};
  value.sSettlementDate = {};
  value.sSettlementCurrency = {};
  value.sLiquidityIndicator = {};
  value.sQuoteId = {};
  value.sQuoteMsgId = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(OrderModifyReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llNewSize = {};
}
void initialize_public(OrderNotCancelledReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llNotCancelledSize = {};
}
void initialize_public(OrderNotModifiedReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llNotModifiedSize = {};
}
void initialize_public(OrderRejectReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llRejectedSize = {};
  value.bReplacementOrderToFollow = {};
}
void initialize_public(OrderStatusReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llConfirmedSize = {};
}
void initialize_public(OrderTradeCorrectReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.sFillType = {};
  value.sTradeDate = {};
  value.sSettlementDate = {};
  value.sSettlementCurrency = {};
}
void initialize_public(OrderTriggerPulledReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
}
void initialize_public(OrderTriggerReport& value) {
  initialize_public(static_cast<OrderReport&>(value));
  value.llConfirmedSize = {};
}
void initialize_public(OrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sBuySellType = {};
  value.sOrderType = {};
  value.sDuration = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.bMinQty = {};
  value.llMinQty = {};
  value.bMaxShowQty = {};
  value.llMaxShowQty = {};
  value.dPrice = {};
  value.dTriggerPrice = {};
  value.bTrailingStop = {};
  value.iTrailByTicks = {};
  value.iTrailByPriceId = {};
  value.sTriggerExchange = {};
  value.sTriggerTicker = {};
  value.iTriggerPriceId = {};
  value.iTriggerCmpOperator = {};
  value.sTradeRoute = {};
  value.sRoutingInstructions = {};
  value.sReleaseCommand = {};
  value.bReleaseSsboe = {};
  value.iReleaseSsboe = {};
  value.bReleaseUsecs = {};
  value.iReleaseUsecs = {};
  value.bCancelSsboe = {};
  value.iCancelSsboe = {};
  value.bCancelUsecs = {};
  value.iCancelUsecs = {};
  value.bCancelAfter = {};
  value.iCancelAfter = {};
  value.bToMarketAt = {};
  value.iToMarketAtSsboe = {};
  value.iToMarketAtUsecs = {};
  value.bToMarketAfterPartialFillInSeconds = {};
  value.iToMarketAfterPartialFillInSeconds = {};
  value.sToMarketOnReject = {};
  value.sToMarketWhenLimitPriceTouched = {};
  value.bReleaseCondition = {};
  value.iReleaseConditionPriceId = {};
  value.iReleaseConditionPriceCmpOperator = {};
  value.dReleaseConditionPrice = {};
  value.iReleaseConditionSizeId = {};
  value.iReleaseConditionSizeCmpOperator = {};
  value.llReleaseConditionSize = {};
  value.sTag = {};
  value.sUserTag = {};
  value.sUserMsg = {};
  value.sWindowName = {};
  value.pContext = {};
}
void initialize_public(LimitOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sBuySellType = {};
  value.sDuration = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.dPrice = {};
  value.sTradeRoute = {};
  value.sRoutingInstructions = {};
  value.sTag = {};
  value.sUserMsg = {};
  value.pContext = {};
}
void initialize_public(MarketOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sBuySellType = {};
  value.sDuration = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.sTradeRoute = {};
  value.sRoutingInstructions = {};
  value.sTag = {};
  value.sUserMsg = {};
  value.pContext = {};
}
void initialize_public(StopLimitOrderParams& value) {
  initialize_public(static_cast<LimitOrderParams&>(value));
  value.dTriggerPrice = {};
}
void initialize_public(StopMarketOrderParams& value) {
  initialize_public(static_cast<MarketOrderParams&>(value));
  value.dTriggerPrice = {};
}
void initialize_public(ModifyOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sOrderNum = {};
  value.sOrderType = {};
  value.sDuration = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.bQty = {};
  value.llQty = {};
  value.bMinQty = {};
  value.llMinQty = {};
  value.dPrice = {};
  value.bPrice = {};
  value.dTriggerPrice = {};
  value.bTriggerPrice = {};
  value.bTrailByTicks = {};
  value.iTrailByTicks = {};
  value.sTriggerExchange = {};
  value.sTriggerTicker = {};
  value.sUserMsg = {};
  value.sWindowName = {};
  value.pContext = {};
}
void initialize_public(ModifyLimitOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sOrderNum = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.dPrice = {};
  value.sUserMsg = {};
  value.pContext = {};
}
void initialize_public(ModifyStopLimitOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sOrderNum = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.dPrice = {};
  value.dTriggerPrice = {};
  value.sUserMsg = {};
  value.pContext = {};
}
void initialize_public(ModifyStopMarketOrderParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sOrderNum = {};
  value.sEntryType = {};
  value.sTradingAlgorithm = {};
  value.llQty = {};
  value.dTriggerPrice = {};
  value.sUserMsg = {};
  value.pContext = {};
}
void initialize_public(BracketParams& value) {
  value.sBracketType = {};
  value.sOrderOperationType = {};
  value.sTargetsToMarketWhenTouched = {};
  value.bTargetsToMarketAt = {};
  value.iTargetsToMarketAtSsboe = {};
  value.iTargetsToMarketAtUsecs = {};
  value.bTargetsToMarketAfterPartialFillInSeconds = {};
  value.iTargetsToMarketAfterPartialFillInSeconds = {};
  value.iTargetArrayLen = {};
  value.asTargetArray = {};
  value.sStopsToMarketOnReject = {};
  value.bStopsToMarketAt = {};
  value.iStopsToMarketAtSsboe = {};
  value.iStopsToMarketAtUsecs = {};
  value.bTrailingStop = {};
  value.iTrailByPriceId = {};
  value.bTrailingStopTriggerTicks = {};
  value.iTrailingStopTriggerTicks = {};
  value.iStopArrayLen = {};
  value.asStopArray = {};
}
void initialize_public(QuoteParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.bBid = {};
  value.llBidQty = {};
  value.dBidPrice = {};
  value.llAskQty = {};
  value.dAskPrice = {};
  value.sUserMsg = {};
}
void initialize_public(QuoteCancelParams& value) {
  value.pAccount = {};
  value.sExchange = {};
  value.sTicker = {};
  value.sUserMsg = {};
}
void initialize_public(BarParams& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.iType = {};
  value.dSpecifiedRange = {};
  value.iSpecifiedTicks = {};
  value.llSpecifiedVolume = {};
  value.iSpecifiedMinutes = {};
  value.iSpecifiedSeconds = {};
  value.bCustomSession = {};
  value.iOpenSsm = {};
  value.iCloseSsm = {};
  value.bProfile = {};
  value.pContext = {};
}
void initialize_public(ReplayBarParams& value) {
  initialize_public(static_cast<BarParams&>(value));
  value.iStartSsboe = {};
  value.iStartUsecs = {};
  value.iEndSsboe = {};
  value.iEndUsecs = {};
  value.sStartDate = {};
  value.sEndDate = {};
}
void initialize_public(BarInfo& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.iType = {};
  value.dSpecifiedRange = {};
  value.iSpecifiedTicks = {};
  value.llSpecifiedVolume = {};
  value.iSpecifiedMinutes = {};
  value.iSpecifiedSeconds = {};
  value.sSpecifiedDate = {};
  value.bCustomSession = {};
  value.iOpenSsm = {};
  value.iCloseSsm = {};
  value.dOpenPrice = {};
  value.dHighPrice = {};
  value.dLowPrice = {};
  value.dClosePrice = {};
  value.bSettlementPrice = {};
  value.dSettlementPrice = {};
  value.bBuyerAsAggressorVolume = {};
  value.llBuyerAsAggressorVolume = {};
  value.bSellerAsAggressorVolume = {};
  value.llSellerAsAggressorVolume = {};
  value.bVwap = {};
  value.dVwap = {};
  value.bVwapLong = {};
  value.dVwapLong = {};
  value.iStartSsboe = {};
  value.iStartUsecs = {};
  value.iEndSsboe = {};
  value.iEndUsecs = {};
  value.llVolume = {};
  value.iNumTrades = {};
  value.iProfileArrayLen = {};
  value.adProfilePrices = {};
  value.allProfileNoAggressorVolumes = {};
  value.allProfileBuyerAsAggressorVolumes = {};
  value.allProfileSellerAsAggressorVolumes = {};
  value.aiProfileNoAggressorNumTrades = {};
  value.aiProfileBuyerAsAggressorNumTrades = {};
  value.aiProfileSellerAsAggressorNumTrades = {};
  value.pContext = {};
}
void initialize_public(BarReplayInfo& value) {
  value.sExchange = {};
  value.sTicker = {};
  value.iType = {};
  value.dSpecifiedRange = {};
  value.iSpecifiedTicks = {};
  value.llSpecifiedVolume = {};
  value.iSpecifiedMinutes = {};
  value.iSpecifiedSeconds = {};
  value.bCustomSession = {};
  value.iOpenSsm = {};
  value.iCloseSsm = {};
  value.pContext = {};
  value.iStartSsboe = {};
  value.iStartUsecs = {};
  value.iEndSsboe = {};
  value.iEndUsecs = {};
  value.sStartDate = {};
  value.sEndDate = {};
  value.bProfile = {};
  value.iRpCode = {};
  value.sRpCode = {};
}
void initialize_public(REngineParams& value) {
  value.sAppName = {};
  value.sAppVersion = {};
  value.sLogFilePath = {};
  value.envp = {};
  value.pContext = {};
  value.pAdmCallbacks = {};
}
void initialize_public(LoginParams& value) {
  value.sMdEnvKey = {};
  value.sMdUser = {};
  value.sMdPassword = {};
  value.sMdCnnctPt = {};
  value.sTsEnvKey = {};
  value.sTsUser = {};
  value.sTsPassword = {};
  value.sTsCnnctPt = {};
  value.sPnlCnnctPt = {};
  value.sIhEnvKey = {};
  value.sIhUser = {};
  value.sIhPassword = {};
  value.sIhCnnctPt = {};
  value.pCallbacks = {};
}
