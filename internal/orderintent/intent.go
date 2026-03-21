package orderintent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type PlaceIntent struct {
	Symbol       string  `json:"symbol"`
	Market       string  `json:"market"`
	Side         string  `json:"side"`
	OrderType    string  `json:"order_type"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price,omitempty"`
	Amount       float64 `json:"amount,omitempty"`
	CurrencyMode string  `json:"currency_mode"`
	Fractional   bool    `json:"fractional"`
}

type CancelIntent struct {
	OrderID string `json:"order_id"`
	Symbol  string `json:"symbol"`
}

type AmendIntent struct {
	OrderID  string   `json:"order_id"`
	Quantity *float64 `json:"quantity,omitempty"`
	Price    *float64 `json:"price,omitempty"`
}

func (i AmendIntent) GetOrderID() string    { return i.OrderID }
func (i AmendIntent) GetQuantity() *float64 { return i.Quantity }
func (i AmendIntent) GetPrice() *float64    { return i.Price }

type PlaceInput struct {
	Symbol       string
	Market       string
	Side         string
	OrderType    string
	Quantity     float64
	Price        float64
	Amount       float64
	CurrencyMode string
	Fractional   bool
}

func NormalizePlace(input PlaceInput) (PlaceIntent, error) {
	symbol := strings.TrimSpace(strings.ToUpper(input.Symbol))
	if symbol == "" {
		return PlaceIntent{}, fmt.Errorf("symbol is required")
	}
	side := strings.TrimSpace(strings.ToLower(input.Side))
	if side == "" {
		return PlaceIntent{}, fmt.Errorf("side is required")
	}

	orderType := normalizeDefault(input.OrderType, "limit")
	fractional := input.Fractional

	// Fractional orders must be market orders
	if fractional && orderType == "limit" {
		orderType = "market"
	}

	intent := PlaceIntent{
		Symbol:       symbol,
		Market:       normalizeDefault(input.Market, "us"),
		Side:         side,
		OrderType:    orderType,
		Quantity:     input.Quantity,
		Price:        input.Price,
		Amount:       input.Amount,
		CurrencyMode: normalizeCurrencyMode(input.CurrencyMode),
		Fractional:   fractional,
	}

	if intent.Market == "us" && looksLikeKRSymbol(intent.Symbol) {
		return PlaceIntent{}, fmt.Errorf("symbol %q looks like a Korean stock code; use --market kr", input.Symbol)
	}
	if intent.Fractional && intent.Market != "us" {
		return PlaceIntent{}, fmt.Errorf("fractional orders are only supported for US stocks")
	}
	if intent.Side != "buy" && intent.Side != "sell" {
		return PlaceIntent{}, fmt.Errorf("unsupported side %q; expected buy or sell", input.Side)
	}
	if intent.OrderType != "limit" && intent.OrderType != "market" {
		return PlaceIntent{}, fmt.Errorf("unsupported order type %q; expected limit or market", input.OrderType)
	}
	if intent.Fractional {
		// Fractional: amount-based, no quantity/price required
		if intent.Amount <= 0 {
			return PlaceIntent{}, fmt.Errorf("amount must be greater than zero for fractional orders")
		}
		intent.Quantity = 0
		intent.Price = 0
	} else {
		if intent.Quantity <= 0 {
			return PlaceIntent{}, fmt.Errorf("quantity must be greater than zero")
		}
		if intent.OrderType == "limit" && intent.Price <= 0 {
			return PlaceIntent{}, fmt.Errorf("price must be greater than zero for limit orders")
		}
		if intent.OrderType == "market" {
			intent.Price = 0
		}
	}

	return intent, nil
}

func NormalizeCancel(orderID, symbol string) (CancelIntent, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return CancelIntent{}, fmt.Errorf("order-id is required")
	}
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return CancelIntent{}, fmt.Errorf("symbol is required")
	}

	return CancelIntent{OrderID: orderID, Symbol: symbol}, nil
}

func NormalizeAmend(orderID string, quantity *float64, price *float64) (AmendIntent, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return AmendIntent{}, fmt.Errorf("order-id is required")
	}
	intent := CancelIntent{OrderID: orderID}

	if quantity == nil && price == nil {
		return AmendIntent{}, fmt.Errorf("at least one of quantity or price must be set")
	}
	if quantity != nil && *quantity <= 0 {
		return AmendIntent{}, fmt.Errorf("quantity must be greater than zero when provided")
	}
	if price != nil && *price <= 0 {
		return AmendIntent{}, fmt.Errorf("price must be greater than zero when provided")
	}

	return AmendIntent{
		OrderID:  intent.OrderID,
		Quantity: quantity,
		Price:    price,
	}, nil
}

func CanonicalPlace(intent PlaceIntent) string {
	fields := map[string]string{
		"currency_mode": intent.CurrencyMode,
		"fractional":    strconv.FormatBool(intent.Fractional),
		"market":        intent.Market,
		"order_type":    intent.OrderType,
		"price":         formatFloat(intent.Price),
		"amount":        formatFloat(intent.Amount),
		"quantity":      formatFloat(intent.Quantity),
		"side":          intent.Side,
		"symbol":        intent.Symbol,
	}
	return canonicalString("place", fields)
}

func CanonicalCancel(intent CancelIntent) string {
	return canonicalString("cancel", map[string]string{
		"order_id": intent.OrderID,
		"symbol":   intent.Symbol,
	})
}

func CanonicalAmend(intent AmendIntent) string {
	fields := map[string]string{"order_id": intent.OrderID}
	if intent.Quantity != nil {
		fields["quantity"] = formatFloat(*intent.Quantity)
	}
	if intent.Price != nil {
		fields["price"] = formatFloat(*intent.Price)
	}
	return canonicalString("amend", fields)
}

func ConfirmToken(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:12]
}

func normalizeDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	return strings.ToLower(value)
}

func normalizeCurrencyMode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "KRW"
	}
	return strings.ToUpper(value)
}

func canonicalString(kind string, fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)+1)
	parts = append(parts, "kind="+kind)
	for _, key := range keys {
		parts = append(parts, key+"="+fields[key])
	}
	return strings.Join(parts, "|")
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

var krSymbolPattern = regexp.MustCompile(`^\d{6}$`)

func looksLikeKRSymbol(symbol string) bool {
	return krSymbolPattern.MatchString(symbol)
}

// InferMarketFromStockCode infers the market from a stock code pattern.
// Korean stock codes start with "A" followed by digits.
func InferMarketFromStockCode(code string) string {
	if strings.HasPrefix(code, "A") && len(code) == 7 {
		rest := code[1:]
		if krSymbolPattern.MatchString(rest) {
			return "kr"
		}
	}
	return "us"
}
