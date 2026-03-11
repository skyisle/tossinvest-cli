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
