package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestGetProfitOverviewParsesAggregate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profit/overview" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if got["rangeType"] != "all" {
			t.Fatalf("rangeType = %v", got["rangeType"])
		}
		if got["startDate"] != "20210315" || got["endDate"] != "20260519" {
			t.Fatalf("dates = %v / %v", got["startDate"], got["endDate"])
		}
		_, _ = w.Write([]byte(`{
			"result": {
				"totalAssetAmount": {"krw": 100, "usd": 0.07},
				"earningAmount":    {"krw": 100, "usd": 0.07},
				"sales": {
					"amount":         {"krw": 90, "usd": 0.06},
					"earningRate":    {"krw": 9.0, "usd": 9.0},
					"purchaseAmount": {"krw": 1000, "usd": 0.7}
				},
				"lending":  {"amount": {"krw": 0, "usd": 0}, "earningRate": {"krw": 0, "usd": 0}, "purchaseAmount": null},
				"dividend": {"amount": {"krw": 10, "usd": 0.01}, "earningRate": {"krw": 0, "usd": 0}, "purchaseAmount": null},
				"maturity": {"amount": {"krw": 0, "usd": 0}, "earningRate": {"krw": 0, "usd": null}, "purchaseAmount": null},
				"interest": 5
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	from := time.Date(2021, 3, 15, 0, 0, 0, 0, KoreaLocation)
	to := time.Date(2026, 5, 19, 0, 0, 0, 0, KoreaLocation)
	got, err := client.GetProfitOverview(context.Background(), "all", from, to)
	if err != nil {
		t.Fatalf("GetProfitOverview: %v", err)
	}
	if got.EarningAmount.KRW != 100 {
		t.Fatalf("earning krw = %v", got.EarningAmount.KRW)
	}
	if got.Sales.Amount.KRW != 90 {
		t.Fatalf("sales krw = %v", got.Sales.Amount.KRW)
	}
	if got.Sales.PurchaseAmount == nil || got.Sales.PurchaseAmount.KRW != 1000 {
		t.Fatalf("sales purchase = %+v", got.Sales.PurchaseAmount)
	}
	if got.Lending.PurchaseAmount != nil {
		t.Fatalf("lending purchase should be nil, got %+v", got.Lending.PurchaseAmount)
	}
	if got.Interest != 5 {
		t.Fatalf("interest = %v", got.Interest)
	}
	if got.StartDate != "20210315" || got.EndDate != "20260519" {
		t.Fatalf("dates = %v / %v", got.StartDate, got.EndDate)
	}
}

func TestListProfitDailyParsesPageAndNormalizesBaseDate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profit/wts/daily/market" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["currency"] != "USD" {
			t.Fatalf("currency = %v", got["currency"])
		}
		if got["page"].(float64) != 0 || got["size"].(float64) != 3 {
			t.Fatalf("page/size = %v / %v", got["page"], got["size"])
		}
		_, _ = w.Write([]byte(`{
			"result": {
				"content": [
					{
						"key": "20260518:KR:A005490:KR7005490008:A005490",
						"marketType": "kr",
						"productCode": "A005490",
						"productName": "POSCO홀딩스",
						"baseDate": "26.5.18",
						"quantity": 19,
						"profitLossAmount": {"krw": 505600, "usd": 505600},
						"profitRate": 6.18,
						"sellAmount": {"krw": 8700000, "usd": 8700000},
						"buyAmount": {"krw": 8176999, "usd": 8176999},
						"commission": {"krw": 0, "usd": 0},
						"tax": {"krw": 17400, "usd": 17400},
						"averageSellPrice": {"krw": 457894, "usd": 457894},
						"averageBuyPrice": {"krw": 430368, "usd": 430368},
						"averageProfitLossPrice": {"krw": 27526, "usd": 27526},
						"buyExchangeRate": null,
						"sellExchangeRate": null,
						"exchangeProfitLossAmount": null,
						"buyTax": {"krw": 0, "usd": 0},
						"sellTax": {"krw": 17400, "usd": 17400},
						"buyCommission": {"krw": 0, "usd": 0},
						"sellCommission": {"krw": 0, "usd": 0}
					}
				],
				"currentPage": 0,
				"pageSize": 3,
				"hasNext": true
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, KoreaLocation)
	to := time.Date(2026, 5, 19, 0, 0, 0, 0, KoreaLocation)
	page, err := client.ListProfitDaily(context.Background(), "USD", from, to, 0, 3)
	if err != nil {
		t.Fatalf("ListProfitDaily: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d", len(page.Items))
	}
	if !page.HasNext {
		t.Fatal("expected hasNext=true")
	}
	got := page.Items[0]
	if got.BaseDate != "2026-05-18" {
		t.Fatalf("baseDate normalized = %q", got.BaseDate)
	}
	if got.BaseDateRaw != "26.5.18" {
		t.Fatalf("baseDateRaw = %q", got.BaseDateRaw)
	}
	if got.ProductName != "POSCO홀딩스" {
		t.Fatalf("productName = %q", got.ProductName)
	}
	if got.ProfitLossAmount.KRW != 505600 {
		t.Fatalf("pnl krw = %v", got.ProfitLossAmount.KRW)
	}
}

func TestListProfitGroupSendsExpectedBodyAndFlattensCommission(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/profit/group/market" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["marketType"] != "kr" || got["profitType"] != "sales" || got["currency"] != "USD" {
			t.Fatalf("body shape mismatch: %+v", got)
		}
		paging, _ := got["pagingParam"].(map[string]any)
		if paging["page"].(float64) != 0 || paging["size"].(float64) != 50 {
			t.Fatalf("pagingParam = %+v", paging)
		}
		_, _ = w.Write([]byte(`{
			"result": {
				"totalEarningAmount": {"krw": 17139655, "usd": 17139654.4},
				"totalEarningRate":   {"krw": 8.3, "usd": 8.3},
				"isLast": false,
				"nextPage": {"size": 50, "page": 1},
				"items": [
					{
						"key": "KR:A005930:DEFAULT_ISIN:DEFAULT_SYMBOL",
						"productCode": "A005930",
						"isin": null,
						"symbol": null,
						"name": "삼성전자",
						"logoImageUrl": "https://example.com/x.png",
						"profitType": "sales",
						"earningAmount":  {"krw": 5285888, "usd": 5285887.4},
						"earningRate":    {"krw": 12.2, "usd": 12.2},
						"purchaseAmount": {"krw": 43239612, "usd": 43239612.2},
						"commission": {"buyAmount": null, "sellAmount": null, "totalAmount": {"krw": 754, "usd": 754.58}, "krw": 754, "usd": 754.58},
						"tax":        {"buyAmount": null, "sellAmount": null, "totalAmount": {"krw": 97246, "usd": 97246}, "krw": 97246, "usd": 97246},
						"quantity": 250
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, KoreaLocation)
	to := time.Date(2026, 5, 19, 0, 0, 0, 0, KoreaLocation)
	page, err := client.ListProfitGroup(context.Background(), "kr", "sales", "year", "USD", from, to, 0, 0)
	if err != nil {
		t.Fatalf("ListProfitGroup: %v", err)
	}
	if page.IsLast {
		t.Fatal("expected isLast=false")
	}
	if page.NextPage == nil || page.NextPage.Page != 1 {
		t.Fatalf("nextPage = %+v", page.NextPage)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d", len(page.Items))
	}
	it := page.Items[0]
	if it.Commission.KRW != 754 || it.Tax.KRW != 97246 {
		t.Fatalf("commission/tax flatten failed: %+v %+v", it.Commission, it.Tax)
	}
	if it.Name != "삼성전자" {
		t.Fatalf("name = %q", it.Name)
	}
}

func TestListAllProfitGroupPagesUntilIsLast(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		paging, _ := got["pagingParam"].(map[string]any)
		page := int(paging["page"].(float64))

		switch page {
		case 0:
			_, _ = w.Write([]byte(`{"result":{
				"totalEarningAmount":{"krw":10,"usd":0},
				"totalEarningRate":{"krw":0,"usd":0},
				"isLast": false,
				"nextPage":{"size":2,"page":1},
				"items":[
					{"key":"k1","productCode":"A","name":"alpha","profitType":"sales","earningAmount":{"krw":5,"usd":0},"earningRate":{"krw":0,"usd":0},"purchaseAmount":{"krw":0,"usd":0},"commission":{"totalAmount":{"krw":0,"usd":0}},"tax":{"totalAmount":{"krw":0,"usd":0}},"quantity":1}
				]
			}}`))
		case 1:
			_, _ = w.Write([]byte(`{"result":{
				"totalEarningAmount":{"krw":10,"usd":0},
				"totalEarningRate":{"krw":0,"usd":0},
				"isLast": true,
				"nextPage": null,
				"items":[
					{"key":"k2","productCode":"B","name":"beta","profitType":"sales","earningAmount":{"krw":5,"usd":0},"earningRate":{"krw":0,"usd":0},"purchaseAmount":{"krw":0,"usd":0},"commission":{"totalAmount":{"krw":0,"usd":0}},"tax":{"totalAmount":{"krw":0,"usd":0}},"quantity":1}
				]
			}}`))
		default:
			t.Fatalf("unexpected extra page request: %d", page)
		}
	}))
	defer server.Close()

	client := newTestClient(server)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, KoreaLocation)
	to := time.Date(2026, 5, 19, 0, 0, 0, 0, KoreaLocation)
	got, err := client.ListAllProfitGroup(context.Background(), "kr", "sales", "year", "USD", from, to, 2, 5)
	if err != nil {
		t.Fatalf("ListAllProfitGroup: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if !got.IsLast {
		t.Fatal("expected isLast=true after paging completes")
	}
	if len(got.Items) != 2 {
		t.Fatalf("items aggregated = %d", len(got.Items))
	}
	if got.Items[0].Name != "alpha" || got.Items[1].Name != "beta" {
		t.Fatalf("items order/content: %+v", got.Items)
	}
}

func TestNormalizeProfitBaseDate(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"26.5.18":    "2026-05-18",
		"25.12.8":    "2025-12-08",
		"2026.05.18": "2026-05-18",
		"":           "",
		"invalid":    "invalid",
		"26-5-18":    "26-5-18",
	}
	for in, want := range cases {
		if got := normalizeProfitBaseDate(in); got != want {
			t.Errorf("normalizeProfitBaseDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProfitInvalidInputs(t *testing.T) {
	t.Parallel()
	c := New(Config{Session: &session.Session{Cookies: map[string]string{"x": "y"}}})
	if _, err := c.ListProfitGroup(context.Background(), "jp", "sales", "all", "USD", time.Time{}, time.Time{}, 0, 0); err == nil {
		t.Fatal("expected error for unsupported market")
	}
	if _, err := c.ListProfitGroup(context.Background(), "kr", "bad", "all", "USD", time.Time{}, time.Time{}, 0, 0); err == nil {
		t.Fatal("expected error for unsupported profit type")
	}
	if _, err := c.ListProfitGroup(context.Background(), "kr", "sales", "all", "EUR", time.Time{}, time.Time{}, 0, 0); err == nil {
		t.Fatal("expected error for unsupported currency")
	}
	if _, err := c.GetProfitOverview(context.Background(), "fortnight", time.Time{}, time.Time{}); err == nil {
		t.Fatal("expected error for unsupported range")
	}
	// from > to
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, KoreaLocation)
	from := to.AddDate(1, 0, 0)
	if _, err := c.GetProfitOverview(context.Background(), "all", from, to); err == nil || !strings.Contains(err.Error(), "from must be on or before to") {
		t.Fatalf("expected from>to error, got %v", err)
	}
}

func newTestClient(server *httptest.Server) *Client {
	return New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
		},
	})
}
