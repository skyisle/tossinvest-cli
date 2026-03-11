package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	tradingflow "github.com/junghoonkye/tossinvest-cli/internal/trading"
)

type pendingOrderDetails struct {
	OrderNo            string
	OrderedDate        string
	StockCode          string
	TradeType          string
	OrderPrice         float64
	OrderUSDPrice      float64
	Quantity           float64
	PendingQuantity    float64
	OrderPriceTypeCode string
	IsFractionalOrder  bool
	IsAfterMarketOrder bool
}

type stockPriceMetadata struct {
	Close        float64
	CloseKRW     float64
	ExchangeRate float64
}

type mutationEnvelope[T any] struct {
	Result T `json:"result"`
}

type cancelPrepareResult struct {
	DelayCancelExchange bool `json:"delayCancelExchange"`
	OrderKey            string `json:"orderKey"`
	AuthRequired        struct {
		Required   bool   `json:"required"`
		SimpleTrade bool  `json:"simpleTrade"`
		Verifier   any    `json:"verifier"`
	} `json:"authRequired"`
}

type cancelResult struct {
	Message  string `json:"message"`
	OrderDate string `json:"orderDate"`
	OrderNo  any    `json:"orderNo"`
	OrderID  string `json:"orderId"`
	UUID     string `json:"uuid"`
	Modulus  string `json:"modulus"`
	Exponent string `json:"exponent"`
	Keyboard string `json:"keyboard"`
}

func (c *Client) PlacePendingOrder(ctx context.Context, intent orderintent.PlaceIntent) error {
	if err := c.ensureTradingMetadata(ctx); err != nil {
		return err
	}
	productCode, err := c.resolveProductCode(ctx, intent.Symbol)
	if err != nil {
		return err
	}

	info, err := c.getStockInfo(ctx, productCode)
	if err != nil {
		return err
	}
	price, err := c.getStockPrice(ctx, productCode)
	if err != nil {
		return err
	}
	usdRate, err := c.getUSDBaseExchangeRate(ctx)
	if err != nil {
		return err
	}

	meta := stockPriceMetadata{
		Close:        price.Close,
		CloseKRW:     math.Round(price.Close * usdRate),
		ExchangeRate: usdRate,
	}
	bodyPrepare, err := buildPlaceBody(productCode, info.Market.Code, intent, meta, true)
	if err != nil {
		return err
	}
	bodyCreate, err := buildPlaceBody(productCode, info.Market.Code, intent, meta, false)
	if err != nil {
		return err
	}

	var prepare mutationEnvelope[cancelPrepareResult]
	if err := c.postTradingJSON(ctx, fmt.Sprintf("%s/api/v2/wts/trading/order/prepare", c.certBaseURL), bodyPrepare, &prepare); err != nil {
		return err
	}
	if prepare.Result.AuthRequired.Required {
		return tradingflow.ErrInteractiveAuthRequired
	}
	if strings.TrimSpace(prepare.Result.OrderKey) == "" {
		return fmt.Errorf("place prepare response did not include order key")
	}

	var create mutationEnvelope[cancelResult]
	if err := c.postTradingJSONWithHeaders(ctx, fmt.Sprintf("%s/api/v2/wts/trading/order/create", c.certBaseURL), bodyCreate, map[string]string{
		"X-Order-Key": prepare.Result.OrderKey,
	}, &create); err != nil {
		return err
	}
	if strings.TrimSpace(create.Result.UUID) != "" &&
		strings.TrimSpace(create.Result.Modulus) != "" &&
		strings.TrimSpace(create.Result.Exponent) != "" {
		return tradingflow.ErrInteractiveAuthRequired
	}

	return c.ensurePendingOrder(ctx, productCode, intent.Price, intent.Quantity)
}

func (c *Client) GetOrderAvailableActions(ctx context.Context, orderID string) (map[string]any, error) {
	if err := c.ensureTradingMetadata(ctx); err != nil {
		return nil, err
	}
	if err := c.requireSession(); err != nil {
		return nil, err
	}

	order, err := c.lookupPendingOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("stockCode", order.StockCode)
	query.Set("tradeType", order.TradeType)
	query.Set("orderPriceType", order.OrderPriceTypeCode)
	query.Set("fractional", strconv.FormatBool(order.IsFractionalOrder))
	query.Set("isReservationOrder", strconv.FormatBool(order.IsAfterMarketOrder))

	endpoint := fmt.Sprintf("%s/api/v3/trading/order/%s/available-actions?%s", c.certBaseURL, orderID, query.Encode())

	result := map[string]any{}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CancelPendingOrder(ctx context.Context, orderID string) error {
	if err := c.ensureTradingMetadata(ctx); err != nil {
		return err
	}
	order, err := c.lookupPendingOrder(ctx, orderID)
	if err != nil {
		return err
	}

	bodyPrepare, bodyCancel, err := buildCancelBodies(order)
	if err != nil {
		return err
	}

	var prepare mutationEnvelope[cancelPrepareResult]
	if err := c.postTradingJSON(ctx, fmt.Sprintf("%s/api/v2/wts/trading/order/cancel/prepare/%s/%s", c.certBaseURL, order.OrderedDate, order.OrderNo), bodyPrepare, &prepare); err != nil {
		return err
	}
	if prepare.Result.AuthRequired.Required {
		return fmt.Errorf("cancel requires additional authentication")
	}
	if strings.TrimSpace(prepare.Result.OrderKey) == "" {
		return fmt.Errorf("cancel prepare response did not include order key")
	}

	var cancel mutationEnvelope[cancelResult]
	if err := c.postTradingJSONWithHeaders(ctx, fmt.Sprintf("%s/api/v3/wts/trading/order/cancel/%s/%s", c.certBaseURL, order.OrderedDate, order.OrderNo), bodyCancel, map[string]string{
		"X-Order-Key": prepare.Result.OrderKey,
	}, &cancel); err != nil {
		return err
	}
	if strings.TrimSpace(cancel.Result.UUID) != "" &&
		strings.TrimSpace(cancel.Result.Modulus) != "" &&
		strings.TrimSpace(cancel.Result.Exponent) != "" {
		return tradingflow.ErrInteractiveAuthRequired
	}
	if strings.TrimSpace(cancel.Result.Message) == "" {
		return fmt.Errorf("cancel response did not include confirmation message")
	}
	return nil
}

func (c *Client) AmendPendingOrder(ctx context.Context, intent orderintent.AmendIntent) error {
	if err := c.ensureTradingMetadata(ctx); err != nil {
		return err
	}
	order, err := c.lookupPendingOrder(ctx, intent.OrderID)
	if err != nil {
		return err
	}

	info, err := c.getStockInfo(ctx, order.StockCode)
	if err != nil {
		return err
	}

	rate, err := c.getUSDBaseExchangeRate(ctx)
	if err != nil {
		return err
	}

	bodyPrepare, expectedPriceKRW, expectedQty, err := buildAmendBody(order, info.Market.Code, rate, intent.Quantity, intent.Price, true)
	if err != nil {
		return err
	}
	bodyCorrect, _, _, err := buildAmendBody(order, info.Market.Code, rate, intent.Quantity, intent.Price, false)
	if err != nil {
		return err
	}

	if err := c.postRawJSON(ctx, fmt.Sprintf("%s/api/v2/wts/trading/order/correct/prepare/%s/%s", c.certBaseURL, order.OrderedDate, order.OrderNo), bodyPrepare); err != nil {
		return err
	}
	if err := c.postRawJSON(ctx, fmt.Sprintf("%s/api/v2/wts/trading/order/correct/%s/%s", c.certBaseURL, order.OrderedDate, order.OrderNo), bodyCorrect); err != nil {
		return err
	}

	return c.ensureAmendedPendingOrder(ctx, order.StockCode, expectedPriceKRW, expectedQty)
}

func (c *Client) HasPendingOrder(ctx context.Context, orderID string) (bool, error) {
	orders, err := c.ListPendingOrders(ctx)
	if err != nil {
		return false, err
	}

	for _, order := range orders {
		if orderMatchesID(order.Raw, orderID) {
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) getUSDBaseExchangeRate(ctx context.Context) (float64, error) {
	var envelope struct {
		Result struct {
			Rate float64 `json:"rate"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, c.apiBaseURL+"/api/v1/exchange/usd/base-exchange-rate", &envelope); err != nil {
		return 0, err
	}
	if envelope.Result.Rate <= 0 {
		return 0, fmt.Errorf("usd base exchange rate was empty")
	}
	return envelope.Result.Rate, nil
}

func (c *Client) postEmptyJSON(ctx context.Context, endpoint string) error {
	return c.postRawJSON(ctx, endpoint, []byte("{}"))
}

func (c *Client) postRawJSON(ctx context.Context, endpoint string, body []byte) error {
	_, err := c.postJSONBytes(ctx, endpoint, body, nil)
	return err
}

func (c *Client) postTradingJSON(ctx context.Context, endpoint string, body []byte, target any) error {
	return c.postTradingJSONWithHeaders(ctx, endpoint, body, nil, target)
}

func (c *Client) postTradingJSONWithHeaders(ctx context.Context, endpoint string, body []byte, extraHeaders map[string]string, target any) error {
	data, err := c.postJSONBytes(ctx, endpoint, body, extraHeaders)
	if err != nil {
		return err
	}
	if target == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func (c *Client) postJSONBytes(ctx context.Context, endpoint string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	if len(body) == 0 {
		body = []byte("{}")
	}
	req, err := httpNewRequestWithBody(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	c.applySession(req)
	c.applyTradingHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	for name, value := range extraHeaders {
		req.Header.Set(name, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newStatusError(resp.StatusCode, endpoint)
	}
	return data, nil
}

func (c *Client) lookupPendingOrder(ctx context.Context, orderID string) (pendingOrderDetails, error) {
	orders, err := c.ListPendingOrders(ctx)
	if err != nil {
		return pendingOrderDetails{}, err
	}

	for _, order := range orders {
		if order.ID != orderID && !orderMatchesID(order.Raw, orderID) {
			continue
		}
		return decodePendingOrderDetails(order.Raw)
	}

	return pendingOrderDetails{}, fmt.Errorf("pending order %s was not found", orderID)
}

func (c *Client) ensureAmendedPendingOrder(ctx context.Context, stockCode string, expectedPriceKRW, expectedQty float64) error {
	orders, err := c.ListPendingOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if order.Symbol != stockCode {
			continue
		}
		if equalFloat(order.Price, expectedPriceKRW) && equalFloat(order.Quantity, expectedQty) {
			return nil
		}
	}

	return fmt.Errorf("amended pending order was not found after reconciliation")
}

func orderMatchesID(raw json.RawMessage, orderID string) bool {
	var envelope struct {
		OrderNo any    `json:"orderNo"`
		OrderID string `json:"orderId"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false
	}

	if envelope.OrderID == orderID || envelope.ID == orderID {
		return true
	}

	switch value := envelope.OrderNo.(type) {
	case string:
		return strings.TrimSpace(value) == orderID
	case float64:
		return strconv.FormatInt(int64(value), 10) == orderID
	}

	return false
}

func decodePendingOrderDetails(raw json.RawMessage) (pendingOrderDetails, error) {
	var payload struct {
		OrderNo            any     `json:"orderNo"`
		OrderedDate        string  `json:"orderedDate"`
		StockCode          string  `json:"stockCode"`
		TradeType          string  `json:"tradeType"`
		OrderPrice         float64 `json:"orderPrice"`
		OrderUSDPrice      float64 `json:"orderUsdPrice"`
		Quantity           float64 `json:"quantity"`
		PendingQuantity    float64 `json:"pendingQuantity"`
		OrderPriceTypeCode string  `json:"orderPriceTypeCode"`
		IsFractionalOrder  bool    `json:"isFractionalOrder"`
		IsAfterMarketOrder bool    `json:"isAfterMarketOrder"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return pendingOrderDetails{}, err
	}

	return pendingOrderDetails{
		OrderNo:            normalizeOrderIdentifier(payload.OrderNo, ""),
		OrderedDate:        payload.OrderedDate,
		StockCode:          payload.StockCode,
		TradeType:          payload.TradeType,
		OrderPrice:         payload.OrderPrice,
		OrderUSDPrice:      payload.OrderUSDPrice,
		Quantity:           payload.Quantity,
		PendingQuantity:    payload.PendingQuantity,
		OrderPriceTypeCode: payload.OrderPriceTypeCode,
		IsFractionalOrder:  payload.IsFractionalOrder,
		IsAfterMarketOrder: payload.IsAfterMarketOrder,
	}, nil
}

func buildAmendBody(order pendingOrderDetails, marketCode string, usdRate float64, quantity *float64, priceKRW *float64, withOrderKey bool) ([]byte, float64, float64, error) {
	targetQty := order.Quantity
	if quantity != nil {
		targetQty = *quantity
	}

	targetPriceKRW := order.OrderPrice
	targetPriceUSD := order.OrderUSDPrice
	if priceKRW != nil {
		targetPriceKRW = *priceKRW
		targetPriceUSD = round4(targetPriceKRW / usdRate)
	}

	payload := map[string]any{
		"stockCode":              order.StockCode,
		"market":                 marketCode,
		"currencyMode":           "KRW",
		"tradeType":              order.TradeType,
		"price":                  targetPriceUSD,
		"quantity":               targetQty,
		"orderAmount":            0,
		"orderPriceType":         order.OrderPriceTypeCode,
		"agreedOver100Million":   false,
		"max":                    false,
		"isReservationOrder":     order.IsAfterMarketOrder,
		"openPriceSinglePriceYn": false,
	}
	if withOrderKey {
		payload["withOrderKey"] = true
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, 0, err
	}
	return body, targetPriceKRW, targetQty, nil
}

func buildCancelBodies(order pendingOrderDetails) ([]byte, []byte, error) {
	base := map[string]any{
		"isAfterMarketOrder": order.IsAfterMarketOrder,
		"quantity":           order.PendingQuantity,
		"stockCode":          order.StockCode,
		"tradeType":          order.TradeType,
		"isReservationOrder": order.IsAfterMarketOrder,
	}

	preparePayload := make(map[string]any, len(base)+1)
	for key, value := range base {
		preparePayload[key] = value
	}
	preparePayload["withOrderKey"] = true

	bodyPrepare, err := json.Marshal(preparePayload)
	if err != nil {
		return nil, nil, err
	}
	bodyCancel, err := json.Marshal(base)
	if err != nil {
		return nil, nil, err
	}
	return bodyPrepare, bodyCancel, nil
}

func buildPlaceBody(productCode, marketCode string, intent orderintent.PlaceIntent, meta stockPriceMetadata, withOrderKey bool) ([]byte, error) {
	payload := map[string]any{
		"stockCode":              productCode,
		"market":                 marketCode,
		"currencyMode":           intent.CurrencyMode,
		"tradeType":              intent.Side,
		"price":                  round4(intent.Price / meta.ExchangeRate),
		"quantity":               intent.Quantity,
		"orderAmount":            0,
		"orderPriceType":         "00",
		"agreedOver100Million":   false,
		"marginTrading":          false,
		"max":                    false,
		"allowAutoExchange":      true,
		"isReservationOrder":     false,
		"openPriceSinglePriceYn": false,
	}
	if withOrderKey {
		payload["withOrderKey"] = true
	} else {
		payload["extra"] = map[string]any{
			"close":        meta.Close,
			"closeKrw":     meta.CloseKRW,
			"exchangeRate": meta.ExchangeRate,
			"orderMethod":  "종목상세__주문하기",
		}
	}

	return json.Marshal(payload)
}

func (c *Client) ensurePendingOrder(ctx context.Context, stockCode string, expectedPriceKRW, expectedQty float64) error {
	orders, err := c.ListPendingOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if order.Symbol != stockCode {
			continue
		}
		if equalFloat(order.Price, expectedPriceKRW) && equalFloat(order.Quantity, expectedQty) {
			return nil
		}
	}

	return fmt.Errorf("%w: stock=%s price=%.4f qty=%.4f", tradingflow.ErrPlaceNotReconciled, stockCode, expectedPriceKRW, expectedQty)
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func equalFloat(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
