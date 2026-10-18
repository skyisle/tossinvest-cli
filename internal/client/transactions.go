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
	// TransactionsMaxRangeDays mirrors Toss's server-side range cap. Probed
	// 2026-05-19 on both KR and US: `range.to - range.from + 1 <= 365` is
	// accepted; 366+ returns HTTP 400.
	TransactionsMaxRangeDays     = 365
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
			Number     int             `json:"number"`
			Size       int             `json:"size"`
			Key        string          `json:"key"`
			Filters    json.RawMessage `json:"filters"`
			Type       json.RawMessage `json:"type"`
			SettledKey json.RawMessage `json:"settledKey"`
		} `json:"pagingParam"`
		Body     []json.RawMessage `json:"body"`
		LastPage bool              `json:"lastPage"`
		AllAsset bool              `json:"allAsset"`
	} `json:"result"`
}

// transactionCursor carries the follow-up paging params the transactions
// endpoint expects on every page after the first. Both KR and US use the same
// pagingParam shape but with different cursor fields and `allAsset` polarity
// (verified against the live web UI on 2026-05-27):
//
//	market   cursor field          number   allAsset
//	-----    ----------------      ------   --------
//	kr       pagingParam.key       1        true     (e.g. key="2026-04-27::2")
//	us       pagingParam.settledKey 2       false    (e.g. settledKey=3196198223)
//
// We populate whichever fields the server gave us; sending both is harmless on
// either market but only one is meaningful per request.
type transactionCursor struct {
	key        string // KR cursor (date::type form)
	settledKey string // US cursor (numeric transaction id)
	kind       string // server's `pagingParam.type`, echoed back as `type`
	number     int    // server's `pagingParam.number`, echoed back as `number`
	allAsset   bool   // server-expected `allAsset` flag (true on KR, false on US)
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
	page, _, err := c.listTransactionsPage(ctx, market, from, to, filter, size, number, nil)
	return page, err
}

// listTransactionsPage is the cursor-aware single-page fetch. When `cursor` is
// non-nil the request includes the follow-up paging params Toss expects for
// the cursor's market (KR: key/type/number=1/allAsset=true; US: settledKey/
// type/number=2/allAsset=false). The returned *transactionCursor is non-nil
// when the server reports more data, so callers can drive a paging loop
// without parsing the envelope themselves.
func (c *Client) listTransactionsPage(ctx context.Context, market string, from, to time.Time, filter string, size, number int, cursor *transactionCursor) (domain.TransactionPage, *transactionCursor, error) {
	if err := c.requireSession(); err != nil {
		return domain.TransactionPage{}, nil, err
	}
	market, err := normalizeTransactionMarket(market)
	if err != nil {
		return domain.TransactionPage{}, nil, err
	}
	from, to, err = normalizeTransactionRange(from, to)
	if err != nil {
		return domain.TransactionPage{}, nil, err
	}
	filterVal, err := parseTransactionFilter(filter)
	if err != nil {
		return domain.TransactionPage{}, nil, err
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
	if cursor != nil {
		// Cursor params come straight from the previous response's pagingParam.
		// `number` from the cursor wins over the caller-supplied value because
		// the server's own echo is the only reliable next-page hint.
		if cursor.key != "" {
			params.Set("key", cursor.key)
		}
		if cursor.settledKey != "" {
			params.Set("settledKey", cursor.settledKey)
		}
		if cursor.kind != "" {
			params.Set("type", cursor.kind)
		}
		if cursor.allAsset {
			params.Set("allAsset", "true")
		} else {
			params.Set("allAsset", "false")
		}
		if cursor.number > 0 {
			params.Set("number", strconv.Itoa(cursor.number))
		}
	} else if number > 0 {
		params.Set("number", strconv.Itoa(number))
	}

	endpoint := fmt.Sprintf(
		"%s/api/v3/my-assets/transactions/markets/%s?%s",
		c.apiBaseURL, market, params.Encode(),
	)

	var envelope transactionsEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.TransactionPage{}, nil, err
	}

	// Toss's ledger ignores range.from for filtering and instead emits a cursor
	// (key on KR, settledKey on US). Filter client-side so callers see only the
	// requested window.
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
	nextCursor := nextTransactionCursor(market, envelope)
	if !envelope.Result.LastPage {
		page.Next = &domain.PagingParam{
			Number:  envelope.Result.PagingParam.Number + 1,
			Size:    envelope.Result.PagingParam.Size,
			Key:     envelope.Result.PagingParam.Key,
			Filters: rawJSONScalarToString(envelope.Result.PagingParam.Filters),
			Type:    rawJSONScalarToString(envelope.Result.PagingParam.Type),
		}
	}
	return page, nextCursor, nil
}

// nextTransactionCursor pulls the next-page cursor out of a transactions
// envelope, selecting the field the given market uses (`key` on KR,
// `settledKey` on US). Returns nil when the server reports lastPage=true or
// when the chosen field is missing/empty, signalling end of stream. JSON
// RawMessage parsing accepts either a bare number or a quoted string — Toss
// has shipped both shapes historically.
func nextTransactionCursor(market string, envelope transactionsEnvelope) *transactionCursor {
	if envelope.Result.LastPage {
		return nil
	}
	kind := rawJSONScalarToString(envelope.Result.PagingParam.Type)
	number := envelope.Result.PagingParam.Number
	switch market {
	case "us":
		settledKey := rawJSONScalarToString(envelope.Result.PagingParam.SettledKey)
		if settledKey == "" {
			return nil
		}
		return &transactionCursor{settledKey: settledKey, kind: kind, number: number, allAsset: false}
	case "kr":
		if envelope.Result.PagingParam.Key == "" {
			return nil
		}
		return &transactionCursor{key: envelope.Result.PagingParam.Key, kind: kind, number: number, allAsset: true}
	}
	return nil
}

// ListAllTransactions returns the full transaction window the server will expose
// within `[from, to]`. pageLimit<=0 defaults to TransactionsDefaultPageLimit to
// avoid runaway loops.
//
// The two markets paginate differently:
//   - KR: server honours range.to and emits no settledKey, so the client narrows
//     range.to toward `from` between pages and dedupes by SortKey.
//   - US: server ignores range.to but returns `pagingParam.settledKey` plus the
//     paging type/number. The client echoes those back on the next request to
//     walk older entries until lastPage=true (verified against the live
//     ledger via the Toss web UI on 2026-05-27).
func (c *Client) ListAllTransactions(ctx context.Context, market string, from, to time.Time, filter string, size, pageLimit int) ([]domain.Transaction, error) {
	if pageLimit <= 0 {
		pageLimit = TransactionsDefaultPageLimit
	}
	market, err := normalizeTransactionMarket(market)
	if err != nil {
		return nil, err
	}
	from, to, err = normalizeTransactionRange(from, to)
	if err != nil {
		return nil, err
	}
	return c.listAllTransactionsByCursor(ctx, market, from, to, filter, size, pageLimit)
}

// listAllTransactionsByCursor drives the paging loop for both markets. The
// first request goes out without a cursor; every subsequent request reuses the
// `pagingParam` cursor the server returned for the previous page. The server
// ignores range.from for filtering, so we stop on lastPage=true, a missing
// cursor, or two consecutive empty pages (a sign the cursor has walked past
// the caller's window).
func (c *Client) listAllTransactionsByCursor(ctx context.Context, market string, from, to time.Time, filter string, size, pageLimit int) ([]domain.Transaction, error) {
	seen := make(map[string]struct{})
	all := make([]domain.Transaction, 0)
	emptyStreak := 0
	var cursor *transactionCursor
	for i := 0; i < pageLimit; i++ {
		page, next, err := c.listTransactionsPage(ctx, market, from, to, filter, size, 0, cursor)
		if err != nil {
			return nil, err
		}
		newThisPage := 0
		for _, tx := range page.Items {
			if tx.SortKey != "" {
				if _, dup := seen[tx.SortKey]; dup {
					continue
				}
				seen[tx.SortKey] = struct{}{}
			}
			all = append(all, tx)
			newThisPage++
		}
		if page.LastPage || next == nil {
			return all, nil
		}
		if newThisPage == 0 {
			emptyStreak++
			if emptyStreak >= 2 {
				return all, nil
			}
		} else {
			emptyStreak = 0
		}
		cursor = next
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
