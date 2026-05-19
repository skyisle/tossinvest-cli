package domain

import (
	"encoding/json"
	"time"
)

type Account struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	Markets     []string `json:"markets,omitempty"`
	Primary     bool     `json:"primary,omitempty"`
}

type AccountSummary struct {
	TotalAssetAmount      float64                         `json:"total_asset_amount"`
	EvaluatedProfitAmount float64                         `json:"evaluated_profit_amount"`
	ProfitRate            float64                         `json:"profit_rate"`
	OrderableAmountKRW    float64                         `json:"orderable_amount_krw"`
	OrderableAmountUSD    float64                         `json:"orderable_amount_usd"`
	WithdrawableKR        map[string]any                  `json:"withdrawable_kr,omitempty"`
	WithdrawableUS        map[string]any                  `json:"withdrawable_us,omitempty"`
	Markets               map[string]AccountMarketSummary `json:"markets,omitempty"`
}

type AccountMarketSummary struct {
	Market                string  `json:"market"`
	PendingBuyOrderAmount float64 `json:"pending_buy_order_amount"`
	EvaluatedAmount       float64 `json:"evaluated_amount"`
	PrincipalAmount       float64 `json:"principal_amount"`
	EvaluatedProfitAmount float64 `json:"evaluated_profit_amount"`
	ProfitRate            float64 `json:"profit_rate"`
	TotalAssetAmount      float64 `json:"total_asset_amount"`
	OrderableAmountKRW    float64 `json:"orderable_amount_krw"`
	OrderableAmountUSD    float64 `json:"orderable_amount_usd"`
}

type Position struct {
	ProductCode     string  `json:"product_code,omitempty"`
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name,omitempty"`
	MarketType      string  `json:"market_type,omitempty"`
	MarketCode      string  `json:"market_code,omitempty"`
	Quantity        float64 `json:"quantity"`
	AveragePrice    float64 `json:"average_price,omitempty"`
	CurrentPrice    float64 `json:"current_price,omitempty"`
	MarketValue     float64 `json:"market_value,omitempty"`
	UnrealizedPnL   float64 `json:"unrealized_pnl,omitempty"`
	ProfitRate      float64 `json:"profit_rate,omitempty"`
	DailyProfitLoss float64 `json:"daily_profit_loss,omitempty"`
	DailyProfitRate float64 `json:"daily_profit_rate,omitempty"`

	AveragePriceUSD    float64 `json:"average_price_usd,omitempty"`
	CurrentPriceUSD    float64 `json:"current_price_usd,omitempty"`
	MarketValueUSD     float64 `json:"market_value_usd,omitempty"`
	UnrealizedPnLUSD   float64 `json:"unrealized_pnl_usd,omitempty"`
	ProfitRateUSD      float64 `json:"profit_rate_usd,omitempty"`
	DailyProfitLossUSD float64 `json:"daily_profit_loss_usd,omitempty"`
	DailyProfitRateUSD float64 `json:"daily_profit_rate_usd,omitempty"`
}

type Order struct {
	ID                    string          `json:"id"`
	ResolvedFromID        string          `json:"resolved_from_id,omitempty"`
	Symbol                string          `json:"symbol"`
	Name                  string          `json:"name,omitempty"`
	Market                string          `json:"market,omitempty"`
	Side                  string          `json:"side,omitempty"`
	Status                string          `json:"status,omitempty"`
	Quantity              float64         `json:"quantity,omitempty"`
	FilledQuantity        float64         `json:"filled_quantity,omitempty"`
	Price                 float64         `json:"price,omitempty"`
	AverageExecutionPrice float64         `json:"average_execution_price,omitempty"`
	OrderDate             string          `json:"order_date,omitempty"`
	SubmittedAt           *time.Time      `json:"submitted_at,omitempty"`
	Raw                   json.RawMessage `json:"raw,omitempty"`
}

type WatchlistItem struct {
	Group    string  `json:"group,omitempty"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name,omitempty"`
	Currency string  `json:"currency,omitempty"`
	Base     float64 `json:"base,omitempty"`
	Last     float64 `json:"last,omitempty"`
}

type Transaction struct {
	Type             string          `json:"type"`
	Category         string          `json:"category"`
	Code             string          `json:"code,omitempty"`
	DisplayName      string          `json:"display_name,omitempty"`
	DisplayType      string          `json:"display_type,omitempty"`
	Summary          string          `json:"summary,omitempty"`
	Market           string          `json:"market"`
	Currency         string          `json:"currency"`
	StockCode        string          `json:"stock_code,omitempty"`
	StockName        string          `json:"stock_name,omitempty"`
	Quantity         float64         `json:"quantity,omitempty"`
	Amount           float64         `json:"amount"`
	AdjustedAmount   float64         `json:"adjusted_amount"`
	CommissionAmount float64         `json:"commission_amount,omitempty"`
	TaxAmount        float64         `json:"tax_amount,omitempty"`
	BalanceAmount    float64         `json:"balance_amount,omitempty"`
	Date             string          `json:"date,omitempty"`
	DateTime         string          `json:"datetime,omitempty"`
	OrderDate        string          `json:"order_date,omitempty"`
	SettlementDate   string          `json:"settlement_date,omitempty"`
	TradeType        string          `json:"trade_type,omitempty"`
	ReferenceType    string          `json:"reference_type,omitempty"`
	ReferenceID      string          `json:"reference_id,omitempty"`
	SortKey          string          `json:"sort_key,omitempty"`
	Raw              json.RawMessage `json:"raw,omitempty"`
}

type TransactionPage struct {
	Market   string        `json:"market"`
	Items    []Transaction `json:"items"`
	LastPage bool          `json:"last_page"`
	Next     *PagingParam  `json:"next,omitempty"`
}

type PagingParam struct {
	Number  int    `json:"number,omitempty"`
	Size    int    `json:"size,omitempty"`
	Key     string `json:"key,omitempty"`
	Filters string `json:"filters,omitempty"`
	Type    string `json:"type,omitempty"`
}

type TransactionOverview struct {
	Market                  string                         `json:"market"`
	OrderableKRW            float64                        `json:"orderable_krw"`
	OrderableUSD            float64                        `json:"orderable_usd"`
	Withdrawable            []SettlementBucket             `json:"withdrawable,omitempty"`
	DisplayWithdrawable     []SettlementBucket             `json:"display_withdrawable,omitempty"`
	Deposit                 []SettlementBucket             `json:"deposit,omitempty"`
	EstimateSettlement      []SettlementEstimate           `json:"estimate_settlement,omitempty"`
	WithdrawableBottomSheet []WithdrawableBottomSheetEntry `json:"withdrawable_bottom_sheet,omitempty"`
}

type SettlementBucket struct {
	Date string  `json:"date,omitempty"`
	KRW  float64 `json:"krw,omitempty"`
	USD  float64 `json:"usd,omitempty"`
}

type SettlementEstimate struct {
	Date       string  `json:"date,omitempty"`
	BuyAmount  float64 `json:"buy_amount,omitempty"`
	SellAmount float64 `json:"sell_amount,omitempty"`
}

type WithdrawableBottomSheetEntry struct {
	Title string  `json:"title"`
	KRW   float64 `json:"krw,omitempty"`
	USD   float64 `json:"usd,omitempty"`
}

type Quote struct {
	ProductCode    string    `json:"product_code,omitempty"`
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name,omitempty"`
	MarketCode     string    `json:"market_code,omitempty"`
	Market         string    `json:"market,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	ReferencePrice float64   `json:"reference_price,omitempty"`
	Last           float64   `json:"last,omitempty"`
	Change         float64   `json:"change,omitempty"`
	ChangeRate     float64   `json:"change_rate,omitempty"`
	Volume         float64   `json:"volume,omitempty"`
	Status         string    `json:"status,omitempty"`
	BadgeCount     int       `json:"badge_count,omitempty"`
	NoticeCount    int       `json:"notice_count,omitempty"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// MoneyPair is Toss's two-currency amount envelope used across the profit API.
// KR-only realized P&L populates only krw; US realized P&L populates both with
// usd holding the actual cash amount and krw the KRW-equivalent at the trade's
// FX rate. JSON shape matches the upstream Toss response one-to-one.
type MoneyPair struct {
	KRW float64 `json:"krw"`
	USD float64 `json:"usd"`
}

// ProfitOverview is the aggregate realized P&L for a date window, broken down
// by source (sales / lending / dividend / maturity / interest). The window is
// either the user-provided [from, to] or the Toss "all-time" default (since
// account opening). Each leaf is a MoneyPair carrying both KRW and USD totals.
type ProfitOverview struct {
	RangeType        string         `json:"range_type"`
	StartDate        string         `json:"start_date"`
	EndDate          string         `json:"end_date"`
	TotalAssetAmount MoneyPair      `json:"total_asset_amount"`
	EarningAmount    MoneyPair      `json:"earning_amount"`
	Sales            ProfitOverviewLeg  `json:"sales"`
	Lending          ProfitOverviewLeg  `json:"lending"`
	Dividend         ProfitOverviewLeg  `json:"dividend"`
	Maturity         ProfitOverviewLeg  `json:"maturity"`
	Interest         float64        `json:"interest"`
}

// ProfitOverviewLeg is one row of the lifetime breakdown (sales/lending/...)
// in /api/v1/profit/overview. Toss returns nil purchaseAmount for legs without
// a notional basis (lending/dividend/maturity); we keep it as a pointer so
// callers can distinguish "no basis" from "zero".
type ProfitOverviewLeg struct {
	Amount         MoneyPair  `json:"amount"`
	EarningRate    MoneyPair  `json:"earning_rate"`
	PurchaseAmount *MoneyPair `json:"purchase_amount,omitempty"`
}

// ProfitDailyItem is one row from /api/v1/profit/wts/daily/market — a single
// (date, symbol) pair's realized P&L with the buy/sell totals and FX deltas
// that fed into it. baseDate comes back as "26.5.18" from Toss; we normalize
// to YYYY-MM-DD in BaseDate but keep the raw string in BaseDateRaw for parity
// with the upstream payload.
type ProfitDailyItem struct {
	Key                       string    `json:"key"`
	MarketType                string    `json:"market_type"`
	ProductCode               string    `json:"product_code"`
	ISIN                      string    `json:"isin,omitempty"`
	Symbol                    string    `json:"symbol,omitempty"`
	ProductName               string    `json:"product_name"`
	BaseDate                  string    `json:"base_date"`
	BaseDateRaw               string    `json:"base_date_raw,omitempty"`
	Quantity                  float64   `json:"quantity"`
	ProfitLossAmount          MoneyPair `json:"profit_loss_amount"`
	ProfitRate                float64   `json:"profit_rate"`
	SellAmount                MoneyPair `json:"sell_amount"`
	BuyAmount                 MoneyPair `json:"buy_amount"`
	Commission                MoneyPair `json:"commission"`
	Tax                       MoneyPair `json:"tax"`
	AverageSellPrice          MoneyPair `json:"average_sell_price"`
	AverageBuyPrice           MoneyPair `json:"average_buy_price"`
	AverageProfitLossPrice    MoneyPair `json:"average_profit_loss_price"`
	BuyExchangeRate           *float64  `json:"buy_exchange_rate,omitempty"`
	SellExchangeRate          *float64  `json:"sell_exchange_rate,omitempty"`
	ExchangeProfitLossAmount  *float64  `json:"exchange_profit_loss_amount,omitempty"`
	BuyTax                    MoneyPair `json:"buy_tax"`
	SellTax                   MoneyPair `json:"sell_tax"`
	BuyCommission             MoneyPair `json:"buy_commission"`
	SellCommission            MoneyPair `json:"sell_commission"`
}

// ProfitDailyPage wraps one paginated response from
// /api/v1/profit/wts/daily/market. HasNext mirrors the server-side flag; when
// true callers should request page CurrentPage+1 with the same body.
type ProfitDailyPage struct {
	Items       []ProfitDailyItem `json:"items"`
	CurrentPage int               `json:"current_page"`
	PageSize    int               `json:"page_size"`
	HasNext     bool              `json:"has_next"`
}

// ProfitGroupItem is one row from /api/v2/profit/group/market — a single
// symbol's realized P&L across the requested window for one market (kr|us).
// Commission and Tax are flattened to a MoneyPair here even though Toss
// returns a nested {buyAmount, sellAmount, totalAmount, krw, usd}; only the
// total is actionable for the CLI.
type ProfitGroupItem struct {
	Key            string    `json:"key"`
	ProductCode    string    `json:"product_code"`
	ISIN           string    `json:"isin,omitempty"`
	Symbol         string    `json:"symbol,omitempty"`
	Name           string    `json:"name"`
	LogoImageURL   string    `json:"logo_image_url,omitempty"`
	ProfitType     string    `json:"profit_type"`
	EarningAmount  MoneyPair `json:"earning_amount"`
	EarningRate    MoneyPair `json:"earning_rate"`
	PurchaseAmount MoneyPair `json:"purchase_amount"`
	Commission     MoneyPair `json:"commission"`
	Tax            MoneyPair `json:"tax"`
	Quantity       float64   `json:"quantity"`
}

// ProfitGroupPage is one paginated response from /api/v2/profit/group/market.
// IsLast is the server's authoritative end-of-stream signal; NextPage echoes
// the {page,size} the client should send for the next request.
type ProfitGroupPage struct {
	Market             string            `json:"market"`
	ProfitType         string            `json:"profit_type"`
	Currency           string            `json:"currency"`
	TotalEarningAmount MoneyPair         `json:"total_earning_amount"`
	TotalEarningRate   MoneyPair         `json:"total_earning_rate"`
	Items              []ProfitGroupItem `json:"items"`
	IsLast             bool              `json:"is_last"`
	NextPage           *ProfitPageCursor `json:"next_page,omitempty"`
}

// ProfitPageCursor is the {page,size} shape Toss echoes back in nextPage.
type ProfitPageCursor struct {
	Page int `json:"page"`
	Size int `json:"size"`
}
