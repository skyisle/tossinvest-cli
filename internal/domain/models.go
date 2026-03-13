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
