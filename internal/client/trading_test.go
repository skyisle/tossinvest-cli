package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/session"
	tradingflow "github.com/junghoonkye/tossinvest-cli/internal/trading"
)

func TestCancelPendingOrder(t *testing.T) {
	t.Parallel()

	var paths []string
	var bodies []string
	var browserTabIDs []string
	var accountHeaders []string
	var appVersions []string
	var orderKeys []string
	pendingCalls := 0
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05.000")
	preparePath := "/api/v2/wts/trading/order/cancel/prepare/" + today + "/14"
	cancelPath := "/api/v3/wts/trading/order/cancel/" + today + "/14"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/trading/orders/histories/all/pending" {
			pendingCalls++
			if pendingCalls == 1 {
				_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"` + today + `","orderNo":14,"tradeType":"buy","orderPrice":700,"orderUsdPrice":0.4753,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"result":[]}`))
			return
		}
		paths = append(paths, r.URL.Path)
		browserTabIDs = append(browserTabIDs, r.Header.Get("Browser-Tab-Id"))
		accountHeaders = append(accountHeaders, r.Header.Get("X-Tossinvest-Account"))
		appVersions = append(appVersions, r.Header.Get("App-Version"))
		orderKeys = append(orderKeys, r.Header.Get("X-Order-Key"))
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case preparePath:
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::cancel","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case cancelPath:
			_, _ = w.Write([]byte(`{"result":{"message":"취소 되었어요.","orderDate":"` + today + `","orderNo":14,"orderId":"test-order-id"}}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = io.WriteString(w, `{"result":{"body":[{"orderedAt":"`+now+`","lastExecutedAt":"`+now+`","orderNo":15,"orderId":"completed-order-id","stockCode":"US20220809012","stockName":"TSLL","symbol":"TSLL","tradeType":"buy","status":"취소","orderQuantity":1,"executedQuantity":0,"userOrderDate":"`+today+`","orderPrice":{"krw":700},"averageExecutionPrice":{"krw":0}}]}}`)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	intent, err := orderintent.NormalizeCancel("14", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	result, err := client.CancelPendingOrder(context.Background(), intent)
	if err != nil {
		t.Fatalf("CancelPendingOrder returned error: %v", err)
	}
	if result.Status != "canceled" {
		t.Fatalf("expected canceled result, got %q", result.Status)
	}
	if result.CurrentOrderID != today+"/15" {
		t.Fatalf("expected current order id %s/15, got %q", today, result.CurrentOrderID)
	}

	if len(paths) != 3 {
		t.Fatalf("expected 3 non-pending requests, got %d", len(paths))
	}
	if paths[0] != preparePath {
		t.Fatalf("unexpected prepare path: %s", paths[0])
	}
	if paths[1] != cancelPath {
		t.Fatalf("unexpected cancel path: %s", paths[1])
	}
	if browserTabIDs[0] != "browser-tab-test123" || browserTabIDs[1] != "browser-tab-test123" {
		t.Fatalf("unexpected browser-tab-id headers: %#v", browserTabIDs)
	}
	if accountHeaders[0] != "1" || accountHeaders[1] != "1" {
		t.Fatalf("unexpected account headers: %#v", accountHeaders)
	}
	if appVersions[0] != "v260311.1636" || appVersions[1] != "v260311.1636" {
		t.Fatalf("unexpected app-version headers: %#v", appVersions)
	}
	if orderKeys[0] != "" {
		t.Fatalf("prepare request should not include x-order-key: %#v", orderKeys)
	}
	if orderKeys[1] != "trade::session::test::cancel" {
		t.Fatalf("unexpected final x-order-key header: %#v", orderKeys)
	}

	var gotPrepare map[string]any
	if err := json.Unmarshal([]byte(bodies[0]), &gotPrepare); err != nil {
		t.Fatalf("prepare body was not valid json: %v", err)
	}
	if gotPrepare["stockCode"] != "US20220809012" || gotPrepare["tradeType"] != "buy" || gotPrepare["withOrderKey"] != true {
		t.Fatalf("unexpected prepare body: %#v", gotPrepare)
	}

	var gotCancel map[string]any
	if err := json.Unmarshal([]byte(bodies[1]), &gotCancel); err != nil {
		t.Fatalf("cancel body was not valid json: %v", err)
	}
	if _, ok := gotCancel["withOrderKey"]; ok {
		t.Fatalf("cancel body should not include withOrderKey: %#v", gotCancel)
	}
	if gotCancel["stockCode"] != "US20220809012" || gotCancel["tradeType"] != "buy" {
		t.Fatalf("unexpected cancel body: %#v", gotCancel)
	}
}

func TestGetOrderAvailableActionsUsesResolvedPendingOrderID(t *testing.T) {
	t.Parallel()

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/trading/orders/histories/all/pending" {
			_, _ = w.Write([]byte(`{"result":[{"orderId":"broker/order+raw==","stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":14,"tradeType":"buy","orderPrice":700,"orderUsdPrice":0.4753,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
			return
		}

		requestedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"enabled":true}`))
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	if _, err := client.GetOrderAvailableActions(context.Background(), "2026-03-11/14"); err != nil {
		t.Fatalf("GetOrderAvailableActions returned error: %v", err)
	}

	want := "/api/v3/trading/order/broker%2Forder+raw==/available-actions?fractional=false&isReservationOrder=false&orderPriceType=00&stockCode=US20220809012&tradeType=buy"
	if requestedPath != want {
		t.Fatalf("unexpected available-actions path:\nwant: %s\ngot:  %s", want, requestedPath)
	}
}

func TestGetOrderAvailableActionsTreats400AsSoftFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/trading/orders/histories/all/pending" {
			_, _ = w.Write([]byte(`{"result":[{"orderId":"broker/order+raw==","stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":14,"tradeType":"buy","orderPrice":700,"orderUsdPrice":0.4753,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"unsupported"}`))
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	result, err := client.GetOrderAvailableActions(context.Background(), "2026-03-11/14")
	if err != nil {
		t.Fatalf("GetOrderAvailableActions returned error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result on soft failure, got %#v", result)
	}
}

func TestCancelPendingOrderReturnsCompletedHistoryRollover(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05.000")
	preparePath := "/api/v2/wts/trading/order/cancel/prepare/" + today + "/14"
	cancelPath := "/api/v3/wts/trading/order/cancel/" + today + "/14"
	pendingCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			pendingCalls++
			if pendingCalls == 1 {
				_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"` + today + `","orderNo":14,"tradeType":"buy","orderPrice":700,"orderUsdPrice":0.4753,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"result":[]}`))
		case preparePath:
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::cancel","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case cancelPath:
			_, _ = w.Write([]byte(`{"result":{"message":"취소 되었어요.","orderDate":"` + today + `","orderNo":14,"orderId":"test-order-id"}}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = io.WriteString(w, `{"result":{"body":[{"orderedAt":"`+now+`","lastExecutedAt":"`+now+`","orderNo":15,"orderId":"completed-cancel-order-id","stockCode":"US20220809012","stockName":"TSLL","symbol":"TSLL","tradeType":"buy","status":"취소","orderQuantity":1,"executedQuantity":0,"userOrderDate":"`+today+`","orderPrice":{"krw":700},"averageExecutionPrice":{"krw":0}}]}}`)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	intent, err := orderintent.NormalizeCancel(today+"/14", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	result, err := client.CancelPendingOrder(context.Background(), intent)
	if err != nil {
		t.Fatalf("CancelPendingOrder returned error: %v", err)
	}
	if result.Status != "canceled" {
		t.Fatalf("expected canceled result, got %q", result.Status)
	}
	if result.OriginalOrderID != today+"/14" {
		t.Fatalf("expected original order id %s/14, got %q", today, result.OriginalOrderID)
	}
	if result.CurrentOrderID != today+"/15" {
		t.Fatalf("expected current order id %s/15, got %q", today, result.CurrentOrderID)
	}
	if result.OrderID != today+"/15" {
		t.Fatalf("expected order id %s/15, got %q", today, result.OrderID)
	}
}

func TestBuildAmendBodyMatchesCapturedShape(t *testing.T) {
	order := pendingOrderDetails{
		OrderNo:            "13",
		OrderedDate:        "2026-03-11",
		StockCode:          "US20220809012",
		TradeType:          "buy",
		OrderPrice:         600,
		OrderUSDPrice:      0.4074,
		Quantity:           1,
		PendingQuantity:    1,
		OrderPriceTypeCode: "00",
	}
	price := 700.0

	body, expectedPriceKRW, expectedQty, err := buildAmendBody(order, "NSQ", 1472.8, nil, &price, true)
	if err != nil {
		t.Fatalf("buildAmendBody returned error: %v", err)
	}
	if expectedPriceKRW != 700 {
		t.Fatalf("expected KRW price 700, got %v", expectedPriceKRW)
	}
	if expectedQty != 1 {
		t.Fatalf("expected qty 1, got %v", expectedQty)
	}

	expected := `{"agreedOver100Million":false,"currencyMode":"KRW","isReservationOrder":false,"market":"NSQ","max":false,"openPriceSinglePriceYn":false,"orderAmount":0,"orderPriceType":"00","price":0.4753,"quantity":1,"stockCode":"US20220809012","tradeType":"buy","withOrderKey":true}`
	if string(body) != expected {
		t.Fatalf("unexpected amend prepare body:\nwant: %s\ngot:  %s", expected, string(body))
	}

	body, _, _, err = buildAmendBody(order, "NSQ", 1472.8, nil, &price, false)
	if err != nil {
		t.Fatalf("buildAmendBody returned error: %v", err)
	}
	expected = `{"agreedOver100Million":false,"currencyMode":"KRW","isReservationOrder":false,"market":"NSQ","max":false,"openPriceSinglePriceYn":false,"orderAmount":0,"orderPriceType":"00","price":0.4753,"quantity":1,"stockCode":"US20220809012","tradeType":"buy"}`
	if string(body) != expected {
		t.Fatalf("unexpected amend body:\nwant: %s\ngot:  %s", expected, string(body))
	}
}

func TestCancelPendingOrderReturnsInteractiveAuthRequired(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/trading/orders/histories/all/pending" {
			_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":14,"tradeType":"buy","orderPrice":700,"orderUsdPrice":0.4753,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/api/v2/wts/trading/order/cancel/prepare/2026-03-11/14":
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::cancel","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case "/api/v3/wts/trading/order/cancel/2026-03-11/14":
			_, _ = w.Write([]byte(`{"result":{"uuid":"challenge","modulus":"abc","exponent":"10001","keyboard":"<svg/>"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	intent, err := orderintent.NormalizeCancel("14", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	_, err = client.CancelPendingOrder(context.Background(), intent)
	if err == nil {
		t.Fatal("expected auth challenge error")
	}
	if err != tradingflow.ErrInteractiveAuthRequired {
		t.Fatalf("expected ErrInteractiveAuthRequired, got %v", err)
	}
}

func TestBuildPlaceBodyMatchesCapturedShape(t *testing.T) {
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	meta := stockPriceMetadata{
		Close:        14.4,
		CloseKRW:     21208,
		ExchangeRate: 1472.8,
	}

	body, err := buildPlaceBody("US20220809012", "NSQ", intent, meta, true)
	if err != nil {
		t.Fatalf("buildPlaceBody returned error: %v", err)
	}
	expected := `{"agreedOver100Million":false,"allowAutoExchange":true,"currencyMode":"KRW","isReservationOrder":false,"marginTrading":false,"market":"NSQ","max":false,"openPriceSinglePriceYn":false,"orderAmount":0,"orderPriceType":"00","price":0.3395,"quantity":1,"stockCode":"US20220809012","tradeType":"buy","withOrderKey":true}`
	if string(body) != expected {
		t.Fatalf("unexpected place prepare body:\nwant: %s\ngot:  %s", expected, string(body))
	}

	body, err = buildPlaceBody("US20220809012", "NSQ", intent, meta, false)
	if err != nil {
		t.Fatalf("buildPlaceBody returned error: %v", err)
	}
	expected = `{"agreedOver100Million":false,"allowAutoExchange":true,"currencyMode":"KRW","extra":{"close":14.4,"closeKrw":21208,"exchangeRate":1472.8,"orderMethod":"종목상세__주문하기"},"isReservationOrder":false,"marginTrading":false,"market":"NSQ","max":false,"openPriceSinglePriceYn":false,"orderAmount":0,"orderPriceType":"00","price":0.3395,"quantity":1,"stockCode":"US20220809012","tradeType":"buy"}`
	if string(body) != expected {
		t.Fatalf("unexpected place create body:\nwant: %s\ngot:  %s", expected, string(body))
	}
}

func TestPlacePendingOrderSendsXOrderKeyOnCreate(t *testing.T) {
	t.Parallel()

	var paths []string
	var orderKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/search/stocks":
			_, _ = w.Write([]byte(`{"result":{"stocks":[{"stockCode":"US20220809012","stockName":"TSLL","matchType":"EXACT"}]}}`))
		case "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case "/api/v1/product/stock-prices":
			_, _ = w.Write([]byte(`{"result":[{"productCode":"US20220809012","currency":"USD","base":14.38,"close":14.4,"closeKrw":21208,"volume":13409779}]}`))
		case "/api/v1/exchange/usd/base-exchange-rate":
			_, _ = w.Write([]byte(`{"result":{"rate":1472.8}}`))
		case "/api/v2/wts/trading/order/prepare":
			paths = append(paths, r.URL.Path)
			orderKeys = append(orderKeys, r.Header.Get("X-Order-Key"))
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::place","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case "/api/v2/wts/trading/order/create":
			paths = append(paths, r.URL.Path)
			orderKeys = append(orderKeys, r.Header.Get("X-Order-Key"))
			_, _ = w.Write([]byte(`{"result":{"message":"주문 접수 되었어요."}}`))
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":16,"tradeType":"buy","orderPrice":500,"orderUsdPrice":0.3395,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.2121"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	result, err := client.PlacePendingOrder(context.Background(), intent)
	if err != nil {
		t.Fatalf("PlacePendingOrder returned error: %v", err)
	}
	if result.Status != "accepted_pending" {
		t.Fatalf("expected accepted_pending result, got %q", result.Status)
	}
	if result.OrderID != "2026-03-11/16" {
		t.Fatalf("expected reconciled order id 2026-03-11/16, got %q", result.OrderID)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 mutation requests, got %d", len(paths))
	}
	if orderKeys[0] != "" {
		t.Fatalf("prepare request should not include x-order-key: %#v", orderKeys)
	}
	if orderKeys[1] != "trade::session::test::place" {
		t.Fatalf("unexpected create x-order-key header: %#v", orderKeys)
	}
}

func TestPlacePendingOrderReturnsFilledCompletedFromHistory(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05.000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/search/stocks":
			_, _ = w.Write([]byte(`{"result":{"stocks":[{"stockCode":"US20220809012","stockName":"TSLL","matchType":"EXACT"}]}}`))
		case "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case "/api/v1/product/stock-prices":
			_, _ = w.Write([]byte(`{"result":[{"productCode":"US20220809012","currency":"USD","base":14.38,"close":14.4,"closeKrw":21208,"volume":13409779}]}`))
		case "/api/v1/exchange/usd/base-exchange-rate":
			_, _ = w.Write([]byte(`{"result":{"rate":1472.8}}`))
		case "/api/v2/wts/trading/order/prepare":
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::place","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case "/api/v2/wts/trading/order/create":
			_, _ = w.Write([]byte(`{"result":{"message":"주문 접수 되었어요."}}`))
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[]}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = io.WriteString(w, `{"result":{"body":[{"orderedAt":"`+now+`","lastExecutedAt":"`+now+`","orderNo":1,"orderId":"completed-order-id","stockCode":"US20220809012","stockName":"TSLL","symbol":"TSLL","tradeType":"buy","status":"체결완료","orderQuantity":1,"executedQuantity":1,"userOrderDate":"`+today+`","orderPrice":{"krw":21208},"averageExecutionPrice":{"krw":21208}}]}}`)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.2121"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        21208,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	result, err := client.PlacePendingOrder(context.Background(), intent)
	if err != nil {
		t.Fatalf("PlacePendingOrder returned error: %v", err)
	}
	if result.Status != "filled_completed" {
		t.Fatalf("expected filled_completed, got %q", result.Status)
	}
	if result.OrderID != today+"/1" {
		t.Fatalf("expected completed order id, got %q", result.OrderID)
	}
	if result.FilledQuantity != 1 {
		t.Fatalf("expected filled quantity 1, got %v", result.FilledQuantity)
	}
}

func TestAmendPendingOrderReturnsInteractiveAuthRequiredFromPrepare(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":13,"tradeType":"buy","orderPrice":600,"orderUsdPrice":0.4074,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
		case "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case "/api/v1/exchange/usd/base-exchange-rate":
			_, _ = w.Write([]byte(`{"result":{"rate":1472.8}}`))
		case "/api/v2/wts/trading/order/correct/prepare/2026-03-11/13":
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::correct","authRequired":{"required":true,"simpleTrade":false,"verifier":{"type":"interactive"}}}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.2121"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	price := 700.0
	intent, err := orderintent.NormalizeAmend("13", nil, &price)
	if err != nil {
		t.Fatalf("NormalizeAmend returned error: %v", err)
	}

	_, err = client.AmendPendingOrder(context.Background(), intent)
	if err != tradingflow.ErrInteractiveAuthRequired {
		t.Fatalf("expected ErrInteractiveAuthRequired, got %v", err)
	}
}

func TestAmendPendingOrderReturnsInteractiveAuthRequiredFromCorrect(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":13,"tradeType":"buy","orderPrice":600,"orderUsdPrice":0.4074,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
		case "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case "/api/v1/exchange/usd/base-exchange-rate":
			_, _ = w.Write([]byte(`{"result":{"rate":1472.8}}`))
		case "/api/v2/wts/trading/order/correct/prepare/2026-03-11/13":
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::correct","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case "/api/v2/wts/trading/order/correct/2026-03-11/13":
			_, _ = w.Write([]byte(`{"result":{"uuid":"challenge","modulus":"abc","exponent":"10001","keyboard":"<svg/>"}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.2121"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	price := 700.0
	intent, err := orderintent.NormalizeAmend("13", nil, &price)
	if err != nil {
		t.Fatalf("NormalizeAmend returned error: %v", err)
	}

	_, err = client.AmendPendingOrder(context.Background(), intent)
	if err != tradingflow.ErrInteractiveAuthRequired {
		t.Fatalf("expected ErrInteractiveAuthRequired, got %v", err)
	}
}

func TestAmendPendingOrderReturnsCompletedOrderFromHistory(t *testing.T) {
	t.Parallel()

	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05.000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[{"stockCode":"US20220809012","orderedDate":"2026-03-11","orderNo":13,"tradeType":"buy","orderPrice":600,"orderUsdPrice":0.4074,"quantity":1,"pendingQuantity":1,"orderPriceTypeCode":"00","isFractionalOrder":false,"isAfterMarketOrder":false,"status":"체결대기"}]}`))
		case "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case "/api/v1/exchange/usd/base-exchange-rate":
			_, _ = w.Write([]byte(`{"result":{"rate":1472.8}}`))
		case "/api/v2/wts/trading/order/correct/prepare/2026-03-11/13":
			_, _ = w.Write([]byte(`{"result":{"delayCancelExchange":false,"orderKey":"trade::session::test::correct","authRequired":{"required":false,"simpleTrade":true,"verifier":null}}}`))
		case "/api/v2/wts/trading/order/correct/2026-03-11/13":
			_, _ = w.Write([]byte(`{"result":{"message":"주문 수정 되었어요."}}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = io.WriteString(w, `{"result":{"body":[{"orderedAt":"`+now+`","lastExecutedAt":"`+now+`","orderNo":14,"orderId":"completed-amend-order-id","stockCode":"US20220809012","stockName":"TSLL","symbol":"TSLL","tradeType":"buy","status":"체결완료","orderQuantity":1,"executedQuantity":1,"userOrderDate":"`+today+`","orderPrice":{"krw":700},"averageExecutionPrice":{"krw":700}}]}}`)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.2121"},
			Storage: map[string]string{"localStorage:qr-tabId": "browser-tab-test123"},
		},
	})

	price := 700.0
	intent, err := orderintent.NormalizeAmend("13", nil, &price)
	if err != nil {
		t.Fatalf("NormalizeAmend returned error: %v", err)
	}

	result, err := client.AmendPendingOrder(context.Background(), intent)
	if err != nil {
		t.Fatalf("AmendPendingOrder returned error: %v", err)
	}
	if result.Status != "amended_completed" {
		t.Fatalf("expected amended_completed, got %q", result.Status)
	}
	if result.OriginalOrderID != "13" {
		t.Fatalf("expected original order id 13, got %q", result.OriginalOrderID)
	}
	if result.CurrentOrderID != today+"/14" {
		t.Fatalf("expected current order id %s/14, got %q", today, result.CurrentOrderID)
	}
}
