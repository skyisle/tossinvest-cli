package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// Profit endpoints accept dates as YYYYMMDD (no dashes). Toss's 수익분석 page
// derives the value from the user's period selector; the server uses the pair
// to filter — verified 2026-05-19, the year aggregate ≠ the all-time aggregate.
const (
	profitDateLayout = "20060102"

	// ProfitDefaultDailyPageSize / ProfitDefaultGroupPageSize match the values
	// the WTS web bundle sends on first load (size=50 in both cases).
	ProfitDefaultDailyPageSize = 50
	ProfitDefaultGroupPageSize = 50

	// ProfitDefaultPageLimit caps auto-paging loops so a bug in the cursor or
	// hasNext flag can't drive the client into an infinite request loop.
	ProfitDefaultPageLimit = 50
)

// ProfitRangeType mirrors the rangeType strings the WTS web bundle sends. The
// server uses it as a filtering hint alongside startDate/endDate; passing
// "all" with the user's [from, to] is the safe default when the caller does
// not explicitly choose a UI-period.
type ProfitRangeType string

const (
	ProfitRangeDay   ProfitRangeType = "day"
	ProfitRangeWeek  ProfitRangeType = "week"
	ProfitRangeMonth ProfitRangeType = "month"
	ProfitRangeYear  ProfitRangeType = "year"
	ProfitRangeAll   ProfitRangeType = "all"
)

// ProfitType is the realized-profit category. account-interest is reserved
// for the savings-account interest tab and is not implemented by group/market.
type ProfitType string

const (
	ProfitTypeSales           ProfitType = "sales"
	ProfitTypeDividend        ProfitType = "dividend"
	ProfitTypeAccountInterest ProfitType = "account-interest"
)

// ProfitCurrency is the display-currency hint sent in daily/market requests.
// USD is the WTS default and returns both KRW and USD values per item; KRW is
// equivalent in practice but matches the 원 toggle on the UI.
type ProfitCurrency string

const (
	ProfitCurrencyKRW ProfitCurrency = "KRW"
	ProfitCurrencyUSD ProfitCurrency = "USD"
)

// ProfitWindow is the resolved date envelope sent on profit requests. The
// caller passes Go time.Time; we normalize to KST and emit YYYYMMDD strings.
type ProfitWindow struct {
	RangeType ProfitRangeType
	From      time.Time
	To        time.Time
}

func (w ProfitWindow) startDate() string { return w.From.In(KoreaLocation).Format(profitDateLayout) }
func (w ProfitWindow) endDate() string   { return w.To.In(KoreaLocation).Format(profitDateLayout) }

func normalizeProfitWindow(rangeType string, from, to time.Time) (ProfitWindow, error) {
	rt, err := parseProfitRangeType(rangeType)
	if err != nil {
		return ProfitWindow{}, err
	}
	if to.IsZero() {
		to = time.Now().In(KoreaLocation)
	}
	if from.IsZero() {
		// Toss's "전체" selector defaults to the account-open date; we mirror
		// that intent by going back 10 years when the caller does not specify
		// a start. The server still accepts wider ranges and clips to the
		// real account history.
		from = to.AddDate(-10, 0, 0)
	}
	if from.After(to) {
		return ProfitWindow{}, fmt.Errorf("from must be on or before to")
	}
	return ProfitWindow{RangeType: rt, From: from, To: to}, nil
}

func parseProfitRangeType(value string) (ProfitRangeType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return ProfitRangeAll, nil
	case "day":
		return ProfitRangeDay, nil
	case "week":
		return ProfitRangeWeek, nil
	case "month":
		return ProfitRangeMonth, nil
	case "year":
		return ProfitRangeYear, nil
	}
	return "", fmt.Errorf("unsupported range %q; use day|week|month|year|all", value)
}

func parseProfitType(value string) (ProfitType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "sales", "sale":
		return ProfitTypeSales, nil
	case "dividend", "div":
		return ProfitTypeDividend, nil
	case "account-interest", "interest":
		return ProfitTypeAccountInterest, nil
	}
	return "", fmt.Errorf("unsupported profit type %q; use sales|dividend|account-interest", value)
}

func parseProfitCurrency(value string) (ProfitCurrency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "USD":
		return ProfitCurrencyUSD, nil
	case "KRW":
		return ProfitCurrencyKRW, nil
	}
	return "", fmt.Errorf("unsupported currency %q; use KRW or USD", value)
}

func parseProfitMarket(value string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(value))
	if m != "kr" && m != "us" {
		return "", fmt.Errorf("unsupported market %q; use kr or us", value)
	}
	return m, nil
}

// --- Wire envelopes ---

type profitOverviewEnvelope struct {
	Result struct {
		TotalAssetAmount domain.MoneyPair      `json:"totalAssetAmount"`
		EarningAmount    domain.MoneyPair      `json:"earningAmount"`
		Sales            profitOverviewLegRaw  `json:"sales"`
		Lending          profitOverviewLegRaw  `json:"lending"`
		Dividend         profitOverviewLegRaw  `json:"dividend"`
		Maturity         profitOverviewLegRaw  `json:"maturity"`
		Interest         float64               `json:"interest"`
	} `json:"result"`
}

type profitOverviewLegRaw struct {
	Amount         domain.MoneyPair  `json:"amount"`
	EarningRate    domain.MoneyPair  `json:"earningRate"`
	PurchaseAmount *domain.MoneyPair `json:"purchaseAmount"`
}

func (r profitOverviewLegRaw) toDomain() domain.ProfitOverviewLeg {
	return domain.ProfitOverviewLeg{
		Amount:         r.Amount,
		EarningRate:    r.EarningRate,
		PurchaseAmount: r.PurchaseAmount,
	}
}

type profitDailyEnvelope struct {
	Result struct {
		Content     []profitDailyItemRaw `json:"content"`
		CurrentPage int                  `json:"currentPage"`
		PageSize    int                  `json:"pageSize"`
		HasNext     bool                 `json:"hasNext"`
	} `json:"result"`
}

type profitDailyItemRaw struct {
	Key                      string           `json:"key"`
	MarketType               string           `json:"marketType"`
	ProductCode              string           `json:"productCode"`
	ISIN                     string           `json:"isin"`
	Symbol                   string           `json:"symbol"`
	ProductName              string           `json:"productName"`
	BaseDate                 string           `json:"baseDate"`
	Quantity                 float64          `json:"quantity"`
	ProfitLossAmount         domain.MoneyPair `json:"profitLossAmount"`
	ProfitRate               float64          `json:"profitRate"`
	SellAmount               domain.MoneyPair `json:"sellAmount"`
	BuyAmount                domain.MoneyPair `json:"buyAmount"`
	Commission               domain.MoneyPair `json:"commission"`
	Tax                      domain.MoneyPair `json:"tax"`
	AverageSellPrice         domain.MoneyPair `json:"averageSellPrice"`
	AverageBuyPrice          domain.MoneyPair `json:"averageBuyPrice"`
	AverageProfitLossPrice   domain.MoneyPair `json:"averageProfitLossPrice"`
	BuyExchangeRate          *float64         `json:"buyExchangeRate"`
	SellExchangeRate         *float64         `json:"sellExchangeRate"`
	ExchangeProfitLossAmount *float64         `json:"exchangeProfitLossAmount"`
	BuyTax                   domain.MoneyPair `json:"buyTax"`
	SellTax                  domain.MoneyPair `json:"sellTax"`
	BuyCommission            domain.MoneyPair `json:"buyCommission"`
	SellCommission           domain.MoneyPair `json:"sellCommission"`
}

func (r profitDailyItemRaw) toDomain() domain.ProfitDailyItem {
	return domain.ProfitDailyItem{
		Key:                      r.Key,
		MarketType:               r.MarketType,
		ProductCode:              r.ProductCode,
		ISIN:                     r.ISIN,
		Symbol:                   r.Symbol,
		ProductName:              r.ProductName,
		BaseDate:                 normalizeProfitBaseDate(r.BaseDate),
		BaseDateRaw:              r.BaseDate,
		Quantity:                 r.Quantity,
		ProfitLossAmount:         r.ProfitLossAmount,
		ProfitRate:               r.ProfitRate,
		SellAmount:               r.SellAmount,
		BuyAmount:                r.BuyAmount,
		Commission:               r.Commission,
		Tax:                      r.Tax,
		AverageSellPrice:         r.AverageSellPrice,
		AverageBuyPrice:          r.AverageBuyPrice,
		AverageProfitLossPrice:   r.AverageProfitLossPrice,
		BuyExchangeRate:          r.BuyExchangeRate,
		SellExchangeRate:         r.SellExchangeRate,
		ExchangeProfitLossAmount: r.ExchangeProfitLossAmount,
		BuyTax:                   r.BuyTax,
		SellTax:                  r.SellTax,
		BuyCommission:            r.BuyCommission,
		SellCommission:           r.SellCommission,
	}
}

type profitGroupEnvelope struct {
	Result struct {
		TotalEarningAmount domain.MoneyPair     `json:"totalEarningAmount"`
		TotalEarningRate   domain.MoneyPair     `json:"totalEarningRate"`
		IsLast             bool                 `json:"isLast"`
		NextPage           *domain.ProfitPageCursor `json:"nextPage"`
		Items              []profitGroupItemRaw `json:"items"`
	} `json:"result"`
}

type profitGroupItemRaw struct {
	Key            string           `json:"key"`
	ProductCode    string           `json:"productCode"`
	ISIN           *string          `json:"isin"`
	Symbol         *string          `json:"symbol"`
	Name           string           `json:"name"`
	LogoImageURL   string           `json:"logoImageUrl"`
	ProfitType     string           `json:"profitType"`
	EarningAmount  domain.MoneyPair `json:"earningAmount"`
	EarningRate    domain.MoneyPair `json:"earningRate"`
	PurchaseAmount domain.MoneyPair `json:"purchaseAmount"`
	Commission     struct {
		TotalAmount domain.MoneyPair `json:"totalAmount"`
	} `json:"commission"`
	Tax struct {
		TotalAmount domain.MoneyPair `json:"totalAmount"`
	} `json:"tax"`
	Quantity float64 `json:"quantity"`
}

func (r profitGroupItemRaw) toDomain() domain.ProfitGroupItem {
	out := domain.ProfitGroupItem{
		Key:            r.Key,
		ProductCode:    r.ProductCode,
		Name:           r.Name,
		LogoImageURL:   r.LogoImageURL,
		ProfitType:     r.ProfitType,
		EarningAmount:  r.EarningAmount,
		EarningRate:    r.EarningRate,
		PurchaseAmount: r.PurchaseAmount,
		Commission:     r.Commission.TotalAmount,
		Tax:            r.Tax.TotalAmount,
		Quantity:       r.Quantity,
	}
	if r.ISIN != nil {
		out.ISIN = *r.ISIN
	}
	if r.Symbol != nil {
		out.Symbol = *r.Symbol
	}
	return out
}

// --- Client methods ---

// GetProfitOverview calls /api/v1/profit/overview, the lifetime/period
// aggregate broken down by source (sales/lending/dividend/maturity/interest).
// rangeType defaults to "all" when blank; passing the empty zero-value to
// from/to defaults to "10y ago" → "now in KST" so the caller can ask for the
// lifetime aggregate without supplying any dates.
func (c *Client) GetProfitOverview(ctx context.Context, rangeType string, from, to time.Time) (domain.ProfitOverview, error) {
	if err := c.requireSession(); err != nil {
		return domain.ProfitOverview{}, err
	}
	win, err := normalizeProfitWindow(rangeType, from, to)
	if err != nil {
		return domain.ProfitOverview{}, err
	}

	body, _ := json.Marshal(map[string]any{
		"rangeType": string(win.RangeType),
		"startDate": win.startDate(),
		"endDate":   win.endDate(),
	})
	var envelope profitOverviewEnvelope
	if err := c.postJSON(ctx, c.certBaseURL+"/api/v1/profit/overview", body, &envelope); err != nil {
		return domain.ProfitOverview{}, err
	}
	return domain.ProfitOverview{
		RangeType:        string(win.RangeType),
		StartDate:        win.startDate(),
		EndDate:          win.endDate(),
		TotalAssetAmount: envelope.Result.TotalAssetAmount,
		EarningAmount:    envelope.Result.EarningAmount,
		Sales:            envelope.Result.Sales.toDomain(),
		Lending:          envelope.Result.Lending.toDomain(),
		Dividend:         envelope.Result.Dividend.toDomain(),
		Maturity:         envelope.Result.Maturity.toDomain(),
		Interest:         envelope.Result.Interest,
	}, nil
}

// ListProfitDaily fetches a single page from /api/v1/profit/wts/daily/market —
// the per-symbol daily realized P&L list 수익분석 shows when "일별" view is
// selected. Items span all markets (kr + us) and currency is the display
// hint, not a filter; the response carries both KRW and USD amounts.
// page<0 is normalized to 0; size<=0 falls back to ProfitDefaultDailyPageSize.
func (c *Client) ListProfitDaily(ctx context.Context, currency string, from, to time.Time, page, size int) (domain.ProfitDailyPage, error) {
	if err := c.requireSession(); err != nil {
		return domain.ProfitDailyPage{}, err
	}
	cur, err := parseProfitCurrency(currency)
	if err != nil {
		return domain.ProfitDailyPage{}, err
	}
	// rangeType is not part of the daily/market body; we still validate the
	// window so callers get the same from/to defaults.
	win, err := normalizeProfitWindow("", from, to)
	if err != nil {
		return domain.ProfitDailyPage{}, err
	}
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = ProfitDefaultDailyPageSize
	}

	body, _ := json.Marshal(map[string]any{
		"currency":  string(cur),
		"startDate": win.startDate(),
		"endDate":   win.endDate(),
		"page":      page,
		"size":      size,
	})
	var envelope profitDailyEnvelope
	if err := c.postJSON(ctx, c.certBaseURL+"/api/v1/profit/wts/daily/market", body, &envelope); err != nil {
		return domain.ProfitDailyPage{}, err
	}

	items := make([]domain.ProfitDailyItem, 0, len(envelope.Result.Content))
	for _, it := range envelope.Result.Content {
		items = append(items, it.toDomain())
	}
	return domain.ProfitDailyPage{
		Items:       items,
		CurrentPage: envelope.Result.CurrentPage,
		PageSize:    envelope.Result.PageSize,
		HasNext:     envelope.Result.HasNext,
	}, nil
}

// ListAllProfitDaily pages through /api/v1/profit/wts/daily/market until
// hasNext is false or pageLimit is reached. pageLimit<=0 falls back to
// ProfitDefaultPageLimit to keep a misreported hasNext from spinning forever.
func (c *Client) ListAllProfitDaily(ctx context.Context, currency string, from, to time.Time, size, pageLimit int) ([]domain.ProfitDailyItem, error) {
	if pageLimit <= 0 {
		pageLimit = ProfitDefaultPageLimit
	}
	if size <= 0 {
		size = ProfitDefaultDailyPageSize
	}
	all := make([]domain.ProfitDailyItem, 0)
	for page := 0; page < pageLimit; page++ {
		p, err := c.ListProfitDaily(ctx, currency, from, to, page, size)
		if err != nil {
			return nil, err
		}
		all = append(all, p.Items...)
		if !p.HasNext || len(p.Items) == 0 {
			return all, nil
		}
	}
	return all, nil
}

// ListProfitGroup fetches one page of /api/v2/profit/group/market — the
// "종목별 합계" view that aggregates realized P&L by symbol over the window.
// market and profitType are mandatory because the v2 endpoint scopes results
// to one market and one source at a time. size<=0 falls back to the WTS
// default of 50; page<0 is normalized to 0.
func (c *Client) ListProfitGroup(
	ctx context.Context,
	market string,
	profitType string,
	rangeType string,
	currency string,
	from, to time.Time,
	page, size int,
) (domain.ProfitGroupPage, error) {
	if err := c.requireSession(); err != nil {
		return domain.ProfitGroupPage{}, err
	}
	m, err := parseProfitMarket(market)
	if err != nil {
		return domain.ProfitGroupPage{}, err
	}
	pt, err := parseProfitType(profitType)
	if err != nil {
		return domain.ProfitGroupPage{}, err
	}
	cur, err := parseProfitCurrency(currency)
	if err != nil {
		return domain.ProfitGroupPage{}, err
	}
	win, err := normalizeProfitWindow(rangeType, from, to)
	if err != nil {
		return domain.ProfitGroupPage{}, err
	}
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = ProfitDefaultGroupPageSize
	}

	body, _ := json.Marshal(map[string]any{
		"rangeType":  string(win.RangeType),
		"startDate":  win.startDate(),
		"endDate":    win.endDate(),
		"profitType": string(pt),
		"pagingParam": map[string]int{
			"page": page,
			"size": size,
		},
		"marketType": m,
		"currency":   string(cur),
	})
	var envelope profitGroupEnvelope
	if err := c.postJSON(ctx, c.certBaseURL+"/api/v2/profit/group/market", body, &envelope); err != nil {
		return domain.ProfitGroupPage{}, err
	}

	items := make([]domain.ProfitGroupItem, 0, len(envelope.Result.Items))
	for _, it := range envelope.Result.Items {
		items = append(items, it.toDomain())
	}
	return domain.ProfitGroupPage{
		Market:             m,
		ProfitType:         string(pt),
		Currency:           string(cur),
		TotalEarningAmount: envelope.Result.TotalEarningAmount,
		TotalEarningRate:   envelope.Result.TotalEarningRate,
		Items:              items,
		IsLast:             envelope.Result.IsLast,
		NextPage:           envelope.Result.NextPage,
	}, nil
}

// ListAllProfitGroup pages /api/v2/profit/group/market until isLast is true
// or pageLimit is reached. The accumulator preserves the totals from the
// first page (server-side aggregates do not change across pages of the same
// query).
func (c *Client) ListAllProfitGroup(
	ctx context.Context,
	market string,
	profitType string,
	rangeType string,
	currency string,
	from, to time.Time,
	size, pageLimit int,
) (domain.ProfitGroupPage, error) {
	if pageLimit <= 0 {
		pageLimit = ProfitDefaultPageLimit
	}
	if size <= 0 {
		size = ProfitDefaultGroupPageSize
	}
	var aggregate domain.ProfitGroupPage
	for page := 0; page < pageLimit; page++ {
		p, err := c.ListProfitGroup(ctx, market, profitType, rangeType, currency, from, to, page, size)
		if err != nil {
			return domain.ProfitGroupPage{}, err
		}
		if page == 0 {
			aggregate = p
			aggregate.NextPage = nil
		} else {
			aggregate.Items = append(aggregate.Items, p.Items...)
			aggregate.IsLast = p.IsLast
		}
		if p.IsLast || len(p.Items) == 0 {
			aggregate.IsLast = true
			return aggregate, nil
		}
	}
	return aggregate, nil
}

// normalizeProfitBaseDate converts Toss's "26.5.18" style date into ISO
// YYYY-MM-DD with a 4-digit year ("2026-05-18"). Anything we can't parse is
// returned untouched so the raw value still flows through to the caller via
// BaseDateRaw — there's no upside to dropping data.
func normalizeProfitBaseDate(value string) string {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return value
	}
	y, m, d := parts[0], parts[1], parts[2]
	if len(y) == 2 {
		y = "20" + y
	}
	if len(m) == 1 {
		m = "0" + m
	}
	if len(d) == 1 {
		d = "0" + d
	}
	if len(y) != 4 || len(m) != 2 || len(d) != 2 {
		return value
	}
	return y + "-" + m + "-" + d
}
