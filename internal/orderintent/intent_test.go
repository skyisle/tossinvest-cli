package orderintent

import "testing"

func TestNormalizePlace(t *testing.T) {
	intent, err := NormalizePlace(PlaceInput{
		Symbol:       "tsll",
		Market:       "US",
		Side:         "BUY",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "krw",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	if intent.Symbol != "TSLL" {
		t.Fatalf("expected uppercase symbol, got %q", intent.Symbol)
	}
	if intent.Side != "buy" {
		t.Fatalf("expected normalized side, got %q", intent.Side)
	}
	if intent.Market != "us" {
		t.Fatalf("expected normalized market, got %q", intent.Market)
	}
	if intent.CurrencyMode != "KRW" {
		t.Fatalf("expected normalized currency mode, got %q", intent.CurrencyMode)
	}
}

func TestNormalizeAmendRequiresMutationField(t *testing.T) {
	if _, err := NormalizeAmend("5", nil, nil); err == nil {
		t.Fatal("expected error when amend does not change quantity or price")
	}
}

func TestNormalizeCancelRequiresSymbol(t *testing.T) {
	intent, err := NormalizeCancel("5", "tsll")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}
	if intent.Symbol != "TSLL" {
		t.Fatalf("expected uppercase symbol, got %q", intent.Symbol)
	}
}

func TestNormalizePlaceKRSymbolWithUSMarketErrors(t *testing.T) {
	_, err := NormalizePlace(PlaceInput{
		Symbol:       "005930",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        200000,
		CurrencyMode: "KRW",
	})
	if err == nil {
		t.Fatal("expected error for KR symbol with us market")
	}
}

func TestNormalizePlaceKRSymbolWithKRMarketSucceeds(t *testing.T) {
	intent, err := NormalizePlace(PlaceInput{
		Symbol:       "005930",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        200000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}
	if intent.Market != "kr" {
		t.Fatalf("expected market kr, got %q", intent.Market)
	}
}

func TestNormalizePlaceFractionalAutoMarketOrder(t *testing.T) {
	intent, err := NormalizePlace(PlaceInput{Symbol: "TSLL", Side: "buy", Amount: 18000, Fractional: true})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if intent.OrderType != "market" {
		t.Fatalf("expected market, got %q", intent.OrderType)
	}
	if intent.Quantity != 0 {
		t.Fatalf("expected quantity 0, got %v", intent.Quantity)
	}
}

func TestNormalizePlaceFractionalKRRejects(t *testing.T) {
	_, err := NormalizePlace(PlaceInput{Symbol: "290080", Market: "kr", Side: "buy", Amount: 8000, Fractional: true})
	if err == nil {
		t.Fatal("expected error for KR fractional")
	}
}

func TestNormalizePlaceFractionalRequiresAmount(t *testing.T) {
	_, err := NormalizePlace(PlaceInput{Symbol: "TSLL", Side: "buy", Fractional: true})
	if err == nil {
		t.Fatal("expected error when amount is zero")
	}
}

func TestInferMarketFromStockCode(t *testing.T) {
	cases := []struct {
		code   string
		expect string
	}{
		{"A290080", "kr"},
		{"A005930", "kr"},
		{"US20220809012", "us"},
		{"AMX0240221001", "us"},
		{"NAS0241211006", "us"},
	}
	for _, tc := range cases {
		got := InferMarketFromStockCode(tc.code)
		if got != tc.expect {
			t.Errorf("InferMarketFromStockCode(%q) = %q, want %q", tc.code, got, tc.expect)
		}
	}
}

func TestConfirmTokenIsDeterministic(t *testing.T) {
	canonical := CanonicalPlace(PlaceIntent{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})

	first := ConfirmToken(canonical)
	second := ConfirmToken(canonical)
	if first != second {
		t.Fatalf("expected stable token, got %q and %q", first, second)
	}
}
