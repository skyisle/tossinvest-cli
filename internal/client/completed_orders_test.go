package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestListCompletedOrdersParsesStructuredFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/trading/my-orders/markets/us/by-date/completed" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("range.from"); got != "2026-03-01" {
			t.Fatalf("unexpected range.from: %s", got)
		}
		if got := r.URL.Query().Get("range.to"); got != "2026-03-12" {
			t.Fatalf("unexpected range.to: %s", got)
		}
		_, _ = w.Write([]byte(`{
		  "result": {
		    "body": [
		      {
		        "orderedAt": "2026-03-11 00:00:00.000",
		        "orderNo": 25,
		        "orderId": "opaque-completed-order-id",
		        "stockCode": "US20220809012",
		        "stockName": "TSLL",
		        "symbol": "TSLL",
		        "tradeType": "buy",
		        "status": "취소",
		        "orderQuantity": 1,
		        "executedQuantity": 0,
		        "userOrderDate": "2026-03-11",
		        "orderPrice": {"krw": 500},
		        "averageExecutionPrice": {"krw": 0}
		      }
		    ]
		  }
		}`))
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
			Headers: map[string]string{"App-Version": "v260311.1636", "Browser-Tab-Id": "browser-tab-test"},
		},
	})

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 3, 12, 0, 0, 0, 0, time.Local)
	orders, err := client.ListCompletedOrdersRange(context.Background(), "us", from, to, 20, 1)
	if err != nil {
		t.Fatalf("ListCompletedOrdersRange returned error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 completed order, got %d", len(orders))
	}
	if orders[0].ID != "2026-03-11/25" {
		t.Fatalf("expected parsed order id 2026-03-11/25, got %q", orders[0].ID)
	}
	if orders[0].Symbol != "TSLL" {
		t.Fatalf("expected parsed symbol TSLL, got %q", orders[0].Symbol)
	}
	if orders[0].Market != "us" {
		t.Fatalf("expected market us, got %q", orders[0].Market)
	}
	if orders[0].Status != "취소" {
		t.Fatalf("expected status 취소, got %q", orders[0].Status)
	}
}

func TestFindOrderFallsBackToCompletedHistory(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[]}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = w.Write([]byte(`{
			  "result": {
			    "body": [
			      {
			        "orderedAt": "2026-03-11 00:00:00.000",
			        "orderNo": 24,
			        "orderId": "completed-order-id",
			        "stockCode": "US20220809012",
			        "stockName": "TSLL",
			        "symbol": "TSLL",
			        "tradeType": "buy",
			        "status": "체결완료",
			        "orderQuantity": 1,
			        "executedQuantity": 1,
			        "userOrderDate": "2026-03-11",
			        "orderPrice": {"krw": 21208},
			        "averageExecutionPrice": {"krw": 21208}
			      }
			    ]
			  }
			}`))
		case "/api/v2/trading/my-orders/markets/kr/by-date/completed":
			_, _ = w.Write([]byte(`{"result":{"body":[]}}`))
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
			Headers: map[string]string{"App-Version": "v260311.1636", "Browser-Tab-Id": "browser-tab-test"},
		},
	})

	order, err := client.FindOrder(context.Background(), "2026-03-11/24", "all")
	if err != nil {
		t.Fatalf("FindOrder returned error: %v", err)
	}
	if order.ID != "2026-03-11/24" {
		t.Fatalf("expected order id 2026-03-11/24, got %q", order.ID)
	}
	if order.Status != "체결완료" {
		t.Fatalf("expected completed status, got %q", order.Status)
	}
	if order.FilledQuantity != 1 {
		t.Fatalf("expected filled quantity 1, got %v", order.FilledQuantity)
	}
}

func TestFindOrderWithAliasesMarksResolvedFromID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/trading/orders/histories/all/pending":
			_, _ = w.Write([]byte(`{"result":[]}`))
		case "/api/v2/trading/my-orders/markets/us/by-date/completed":
			_, _ = w.Write([]byte(`{
			  "result": {
			    "body": [
			      {
			        "orderedAt": "2026-03-13 00:00:00.000",
			        "lastExecutedAt": "2026-03-13 00:00:00.000",
			        "orderNo": 2,
			        "orderId": "completed-order-id",
			        "stockCode": "US20220809012",
			        "stockName": "TSLL",
			        "symbol": "TSLL",
			        "tradeType": "buy",
			        "status": "취소",
			        "orderQuantity": 1,
			        "executedQuantity": 0,
			        "userOrderDate": "2026-03-13",
			        "orderPrice": {"krw": 500},
			        "averageExecutionPrice": {"krw": 0}
			      }
			    ]
			  }
			}`))
		case "/api/v2/trading/my-orders/markets/kr/by-date/completed":
			_, _ = w.Write([]byte(`{"result":{"body":[]}}`))
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
			Headers: map[string]string{"App-Version": "v260311.1636", "Browser-Tab-Id": "browser-tab-test"},
		},
	})

	order, err := client.FindOrderWithAliases(context.Background(), "2026-03-13/1", "all", "2026-03-13/2")
	if err != nil {
		t.Fatalf("FindOrderWithAliases returned error: %v", err)
	}
	if order.ID != "2026-03-13/2" {
		t.Fatalf("expected order id 2026-03-13/2, got %q", order.ID)
	}
	if order.ResolvedFromID != "2026-03-13/1" {
		t.Fatalf("expected resolved_from_id 2026-03-13/1, got %q", order.ResolvedFromID)
	}
}
