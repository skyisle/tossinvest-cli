package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

const (
	TransactionsMaxRangeDays     = 200
	TransactionsDefaultPageSize  = 50
	TransactionsDefaultPageLimit = 10
)

type TransactionFilter int

const (
	TransactionFilterAll     TransactionFilter = 0
	TransactionFilterTrades  TransactionFilter = 1
	TransactionFilterCash    TransactionFilter = 2
	TransactionFilterInOut   TransactionFilter = 3
	TransactionFilterCashAlt TransactionFilter = 6
)

// KoreaLocation is the timezone Toss uses for ledger day boundaries.
//
// Most sibling endpoints (account, completed_orders, trading) rely on `time.Local`
// implicitly because the typical user machine is already KST. Transactions is the
// first endpoint whose date-range semantics are visible to the user (--from/--to
// mapping to Toss's calendar day), so we make the conversion explicit. Siblings
// can migrate to this variable if they ever need day-granularity correctness on
// non-KST runners (e.g. CI). Fixed +09:00 fallback covers images without tzdata.
var KoreaLocation = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Seoul"); err == nil {
		return loc
	}
	return time.FixedZone("KST", 9*3600)
}()

func normalizeTransactionMarket(value string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(value))
	if m != "kr" && m != "us" {
		return "", fmt.Errorf("unsupported market %q; use kr or us", value)
	}
	return m, nil
}

func normalizeTransactionRange(from, to time.Time) (time.Time, time.Time, error) {
	if to.IsZero() {
		to = time.Now().In(KoreaLocation)
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -30)
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("range.from must be on or before range.to")
	}
	if days := int(to.Sub(from).Hours()/24) + 1; days > TransactionsMaxRangeDays {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"date range %d days exceeds Toss limit of %d days; split the query",
			days, TransactionsMaxRangeDays,
		)
	}
	return from, to, nil
}

type transactionsEnvelope struct {
	Result struct {
		PagingParam struct {
			Number  int             `json:"number"`
			Size    int             `json:"size"`
			Key     string          `json:"key"`
			Filters json.RawMessage `json:"filters"`
			Type    json.RawMessage `json:"type"`
		} `json:"pagingParam"`
		Body     []json.RawMessage `json:"body"`
		LastPage bool              `json:"lastPage"`
		AllAsset bool              `json:"allAsset"`
	} `json:"result"`
}

type transactionOverviewEnvelope struct {
	Result struct {
		OrderableAmount         amountPair                     `json:"orderableAmount"`
		WithdrawableAmount      map[string]json.RawMessage     `json:"withdrawableAmount"`
		DisplayWithdrawable     map[string]json.RawMessage     `json:"displayWithdrawableAmount"`
		DepositAmount           map[string]json.RawMessage     `json:"depositAmount"`
		EstimateSettlement      map[string]json.RawMessage     `json:"estimateSettlementAmount"`
		WithdrawableBottomSheet []withdrawableBottomSheetEntry `json:"withdrawableAmountBottomSheet"`
	} `json:"result"`
}

type amountPair struct {
	KRW *float64 `json:"krw"`
	USD *float64 `json:"usd"`
}

type withdrawableBottomSheetEntry struct {
	Title  string     `json:"title"`
	Amount amountPair `json:"amount"`
}

// ListTransactions fetches a single page from the transactions endpoint.
// filter accepts the same strings as parseTransactionFilter ("all"/"trade"/...
// or the numeric form). size<=0 uses TransactionsDefaultPageSize.
func (c *Client) ListTransactions(ctx context.Context, market string, from, to time.Time, filter string, size, number int) (domain.TransactionPage, error) {
	if err := c.requireSession(); err != nil {
		return domain.TransactionPage{}, err
	}
	market, err := normalizeTransactionMarket(market)
	if err != nil {
		return domain.TransactionPage{}, err
	}
	from, to, err = normalizeTransactionRange(from, to)
	if err != nil {
		return domain.TransactionPage{}, err
	}
	filterVal, err := parseTransactionFilter(filter)
	if err != nil {
		return domain.TransactionPage{}, err
	}
	if size <= 0 {
		size = TransactionsDefaultPageSize
	}
	if number < 0 {
		number = 0
	}

	params := url.Values{}
	params.Set("size", strconv.Itoa(size))
	params.Set("filters", strconv.Itoa(int(filterVal)))
	params.Set("range.from", from.Format("2006-01-02"))
	params.Set("range.to", to.Format("2006-01-02"))
	if number > 0 {
		params.Set("number", strconv.Itoa(number))
	}

	endpoint := fmt.Sprintf(
		"%s/api/v3/my-assets/transactions/markets/%s?%s",
		c.apiBaseURL, market, params.Encode(),
	)

	var envelope transactionsEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.TransactionPage{}, err
	}

	// Toss's ledger ignores range.from and always returns up to `size` entries counting
	// backward from range.to. Filter client-side so callers see only the requested window.
	fromDay, toDayExclusive := transactionRangeBounds(from, to)
	page := domain.TransactionPage{
		Market:   market,
		Items:    make([]domain.Transaction, 0, len(envelope.Result.Body)),
		LastPage: envelope.Result.LastPage,
	}
	for _, raw := range envelope.Result.Body {
		tx := parseTransaction(raw, market)
		if !transactionWithinRange(tx, fromDay, toDayExclusive) {
			continue
		}
		page.Items = append(page.Items, tx)
	}
	if !envelope.Result.LastPage {
		page.Next = &domain.PagingParam{
			Number:  envelope.Result.PagingParam.Number + 1,
			Size:    envelope.Result.PagingParam.Size,
			Key:     envelope.Result.PagingParam.Key,
			Filters: rawJSONScalarToString(envelope.Result.PagingParam.Filters),
			Type:    rawJSONScalarToString(envelope.Result.PagingParam.Type),
		}
	}
	return page, nil
}

// ListAllTransactions pages through by narrowing range.to toward from.
//
// Toss's transactions endpoint ignores number/key/cursor and only honours the
// range.to bound. To fetch older items we re-request with range.to set to the
// oldest date seen in the previous page, then deduplicate by SortKey (since the
// boundary date repeats across pages). pageLimit<=0 defaults to
// TransactionsDefaultPageLimit to avoid runaway loops.
func (c *Client) ListAllTransactions(ctx context.Context, market string, from, to time.Time, filter string, size, pageLimit int) ([]domain.Transaction, error) {
	if pageLimit <= 0 {
		pageLimit = TransactionsDefaultPageLimit
	}
	// Normalize once up front so loop comparisons use the canonical range.
	market, err := normalizeTransactionMarket(market)
	if err != nil {
		return nil, err
	}
	from, to, err = normalizeTransactionRange(from, to)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	all := make([]domain.Transaction, 0)
	currentTo := to
	for i := 0; i < pageLimit; i++ {
		page, err := c.ListTransactions(ctx, market, from, currentTo, filter, size, 0)
		if err != nil {
			return nil, err
		}
		// ListTransactions already filters by the range, so we watch the
		// earliest date here to decide when to stop paging.
		newThisPage := 0
		var earliest time.Time
		var earliestSet bool
		for _, tx := range page.Items {
			if tx.SortKey != "" {
				if _, dup := seen[tx.SortKey]; dup {
					continue
				}
				seen[tx.SortKey] = struct{}{}
			}
			all = append(all, tx)
			newThisPage++
			if d := extractTransactionDate(tx); !d.IsZero() {
				if !earliestSet || d.Before(earliest) {
					earliest = d
					earliestSet = true
				}
			}
		}
		if page.LastPage {
			return all, nil
		}
		if newThisPage == 0 || !earliestSet {
			return all, nil
		}
		// snap earliest to midnight so range.to has day granularity
		earliestDay := time.Date(earliest.Year(), earliest.Month(), earliest.Day(), 0, 0, 0, 0, earliest.Location())
		if !earliestDay.Before(currentTo) {
			// would not advance: break to avoid infinite loop when >size entries share one day
			return all, nil
		}
		if earliestDay.Before(from) {
			return all, nil
		}
		currentTo = earliestDay
	}
	return all, nil
}

// transactionRangeBounds returns the inclusive [fromDay, toDayExclusive) window
// used to filter items against the caller-provided range, at day granularity in KST.
func transactionRangeBounds(from, to time.Time) (time.Time, time.Time) {
	fromKST := from.In(KoreaLocation)
	toKST := to.In(KoreaLocation)
	fromDay := time.Date(fromKST.Year(), fromKST.Month(), fromKST.Day(), 0, 0, 0, 0, KoreaLocation)
	toDayExclusive := time.Date(toKST.Year(), toKST.Month(), toKST.Day(), 0, 0, 0, 0, KoreaLocation).AddDate(0, 0, 1)
	return fromDay, toDayExclusive
}

// transactionWithinRange returns true when the tx's event date falls in [fromDay, toDayExclusive).
// Items without a parsable date are kept (we cannot confidently exclude them).
func transactionWithinRange(tx domain.Transaction, fromDay, toDayExclusive time.Time) bool {
	d := extractTransactionDate(tx)
	if d.IsZero() {
		return true
	}
	return !d.Before(fromDay) && d.Before(toDayExclusive)
}

// extractTransactionDate returns the event date to use for range filtering.
// Priority: DateTime → Date → OrderDate → SettlementDate.
// OrderDate is preferred over SettlementDate because US type=1 trade records
// only populate SettlementDate (usually T+2), which would mis-filter a trade
// executed on 2026-04-17 out of a `--to 2026-04-19` window.
func extractTransactionDate(tx domain.Transaction) time.Time {
	if tx.DateTime != "" {
		for _, layout := range []string{"2006-01-02 15:04:05.000", "2006-01-02 15:04:05.999", "2006-01-02 15:04:05"} {
			if t, err := time.ParseInLocation(layout, tx.DateTime, KoreaLocation); err == nil {
				return t
			}
		}
	}
	if tx.Date != "" {
		if t, err := time.ParseInLocation("2006-01-02", tx.Date, KoreaLocation); err == nil {
			return t
		}
	}
	if tx.OrderDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", tx.OrderDate, KoreaLocation); err == nil {
			return t
		}
	}
	if tx.SettlementDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", tx.SettlementDate, KoreaLocation); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetTransactionsOverview fetches the `/overview` summary for a market.
func (c *Client) GetTransactionsOverview(ctx context.Context, market string) (domain.TransactionOverview, error) {
	if err := c.requireSession(); err != nil {
		return domain.TransactionOverview{}, err
	}
	m := strings.ToLower(strings.TrimSpace(market))
	if m != "kr" && m != "us" {
		return domain.TransactionOverview{}, fmt.Errorf("unsupported market %q; use kr or us", market)
	}

	endpoint := fmt.Sprintf("%s/api/v3/my-assets/transactions/markets/%s/overview", c.apiBaseURL, m)
	var envelope transactionOverviewEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.TransactionOverview{}, err
	}

	overview := domain.TransactionOverview{Market: m}
	overview.OrderableKRW = pointerFloat(envelope.Result.OrderableAmount.KRW)
	overview.OrderableUSD = pointerFloat(envelope.Result.OrderableAmount.USD)
	overview.Withdrawable = parseSettlementBuckets(envelope.Result.WithdrawableAmount)
	overview.DisplayWithdrawable = parseSettlementBuckets(envelope.Result.DisplayWithdrawable)
	overview.Deposit = parseSettlementBuckets(envelope.Result.DepositAmount)
	overview.EstimateSettlement = parseSettlementEstimates(envelope.Result.EstimateSettlement)
	for _, item := range envelope.Result.WithdrawableBottomSheet {
		overview.WithdrawableBottomSheet = append(overview.WithdrawableBottomSheet, domain.WithdrawableBottomSheetEntry{
			Title: item.Title,
			KRW:   pointerFloat(item.Amount.KRW),
			USD:   pointerFloat(item.Amount.USD),
		})
	}
	return overview, nil
}

func parseSettlementBuckets(flat map[string]json.RawMessage) []domain.SettlementBucket {
	if len(flat) == 0 {
		return nil
	}
	buckets := make([]domain.SettlementBucket, 0, 4)
	for i := 0; i < 4; i++ {
		amountKey := fmt.Sprintf("amount%d", i)
		dateKey := fmt.Sprintf("date%d", i)
		amountRaw, hasAmount := flat[amountKey]
		dateRaw, hasDate := flat[dateKey]
		if !hasAmount && !hasDate {
			continue
		}
		b := domain.SettlementBucket{}
		if hasDate {
			_ = json.Unmarshal(dateRaw, &b.Date)
		}
		if hasAmount {
			var pair amountPair
			if err := json.Unmarshal(amountRaw, &pair); err == nil {
				b.KRW = pointerFloat(pair.KRW)
				b.USD = pointerFloat(pair.USD)
			}
		}
		if b.Date == "" && b.KRW == 0 && b.USD == 0 {
			continue
		}
		buckets = append(buckets, b)
	}
	return buckets
}

func parseSettlementEstimates(flat map[string]json.RawMessage) []domain.SettlementEstimate {
	if len(flat) == 0 {
		return nil
	}
	out := make([]domain.SettlementEstimate, 0, len(flat))
	for i := 1; i <= 4; i++ {
		raw, ok := flat[fmt.Sprintf("day%d", i)]
		if !ok {
			continue
		}
		var payload struct {
			SettlementKorDate string  `json:"settlementKorDate"`
			BuyAmount         float64 `json:"buyAmount"`
			SellAmount        float64 `json:"sellAmount"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if payload.SettlementKorDate == "" && payload.BuyAmount == 0 && payload.SellAmount == 0 {
			continue
		}
		out = append(out, domain.SettlementEstimate{
			Date:       payload.SettlementKorDate,
			BuyAmount:  payload.BuyAmount,
			SellAmount: payload.SellAmount,
		})
	}
	return out
}

func parseTransaction(raw json.RawMessage, market string) domain.Transaction {
	tx := domain.Transaction{Market: market, Raw: raw}
	switch market {
	case "us":
		tx.Currency = "USD"
	default:
		tx.Currency = "KRW"
	}

	var payload struct {
		Type            string `json:"type"`
		TransactionType struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"transactionType"`
		DisplayType      string  `json:"displayType"`
		Summary          *string `json:"summary"`
		StockCode        string  `json:"stockCode"`
		StockName        string  `json:"stockName"`
		Quantity         float64 `json:"quantity"`
		Amount           float64 `json:"amount"`
		AdjustedAmount   float64 `json:"adjustedAmount"`
		CommissionAmount float64 `json:"commissionAmount"`
		TotalTaxAmount   float64 `json:"totalTaxAmount"`
		BalanceAmount    float64 `json:"balanceAmount"`
		Date             *string `json:"date"`
		DateTime         *string `json:"dateTime"`
		SettlementDate   *string `json:"settlementDate"`
		ReferenceType    *string `json:"referenceType"`
		ReferenceID      *string `json:"referenceId"`
		CompositeKey     struct {
			OrderDate string `json:"orderDate"`
			Date      string `json:"date"`
			TradeType string `json:"tradeType"`
			StockCode string `json:"stockCode"`
			ID        any    `json:"id"`
			No        any    `json:"no"`
		} `json:"compositeKey"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return tx
	}

	tx.Type = payload.Type
	tx.Code = payload.TransactionType.Code
	tx.DisplayName = payload.TransactionType.DisplayName
	tx.DisplayType = payload.DisplayType
	if payload.Summary != nil {
		tx.Summary = *payload.Summary
	}
	tx.StockCode = payload.StockCode
	tx.StockName = payload.StockName
	tx.Quantity = payload.Quantity
	tx.Amount = payload.Amount
	tx.AdjustedAmount = payload.AdjustedAmount
	tx.CommissionAmount = payload.CommissionAmount
	tx.TaxAmount = payload.TotalTaxAmount
	tx.BalanceAmount = payload.BalanceAmount
	if payload.Date != nil {
		tx.Date = *payload.Date
	}
	if payload.DateTime != nil {
		tx.DateTime = *payload.DateTime
	}
	if payload.SettlementDate != nil {
		tx.SettlementDate = *payload.SettlementDate
	}
	if payload.ReferenceType != nil {
		tx.ReferenceType = *payload.ReferenceType
	}
	if payload.ReferenceID != nil {
		tx.ReferenceID = *payload.ReferenceID
	}
	tx.TradeType = payload.CompositeKey.TradeType
	tx.OrderDate = payload.CompositeKey.OrderDate

	switch payload.Type {
	case "1":
		tx.Category = "trade"
	case "2":
		tx.Category = "cash"
	default:
		tx.Category = "other"
	}

	tx.SortKey = buildTransactionSortKey(tx, payload.CompositeKey.ID, payload.CompositeKey.No, payload.CompositeKey.OrderDate, payload.CompositeKey.Date)
	return tx
}

func buildTransactionSortKey(tx domain.Transaction, id, no any, orderDate, date string) string {
	var builder strings.Builder
	switch {
	case tx.DateTime != "":
		builder.WriteString(tx.DateTime)
	case tx.Date != "":
		builder.WriteString(tx.Date)
	case orderDate != "":
		builder.WriteString(orderDate)
	case date != "":
		builder.WriteString(date)
	}
	if id != nil {
		builder.WriteString("|")
		builder.WriteString(fmt.Sprint(id))
	}
	if no != nil {
		builder.WriteString("|")
		builder.WriteString(fmt.Sprint(no))
	}
	return builder.String()
}

// rawJSONScalarToString unwraps a json.RawMessage scalar (string or number) into its
// bare string form. This avoids leaking JSON quoting into output fields when Toss sends
// e.g. "0" instead of 0 in pagingParam.filters/type.
func rawJSONScalarToString(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return trimmed
}

// parseTransactionFilter accepts numeric ("0"/"1"/...) or named
// ("all"/"trade"/"cash"/"inout"/"cash-alt") filter values.
func parseTransactionFilter(value string) (TransactionFilter, error) {
	if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		switch TransactionFilter(n) {
		case TransactionFilterAll, TransactionFilterTrades, TransactionFilterCash,
			TransactionFilterInOut, TransactionFilterCashAlt:
			return TransactionFilter(n), nil
		}
		return 0, fmt.Errorf("unsupported filter %d; use 0|1|2|3|6", n)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return TransactionFilterAll, nil
	case "trade", "trades":
		return TransactionFilterTrades, nil
	case "cash", "deposit", "withdrawal":
		return TransactionFilterCash, nil
	case "inout", "stock-io", "transfer":
		return TransactionFilterInOut, nil
	case "cash-alt", "cashalt":
		return TransactionFilterCashAlt, nil
	}
	return 0, fmt.Errorf("unsupported filter %q; use all|trade|cash|inout|cash-alt or 0|1|2|3|6", value)
}
