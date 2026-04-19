package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var testTransactions = []domain.Transaction{
	{
		Type:           "1",
		Category:       "trade",
		Code:           "5",
		DisplayName:    "매수",
		DisplayType:    "50",
		Market:         "kr",
		Currency:       "KRW",
		StockCode:      "A999001",
		StockName:      "샘플종목1",
		Quantity:       10,
		Amount:         1000000,
		AdjustedAmount: -1000000,
		Date:           "2024-01-15",
		SettlementDate: "2024-01-17",
		TradeType:      "buy",
		SortKey:        "2024-01-15|A999001",
	},
	{
		Type:           "2",
		Category:       "cash",
		Code:           "1",
		DisplayName:    "입금",
		DisplayType:    "13",
		Summary:        "배당금입금",
		Market:         "kr",
		Currency:       "KRW",
		StockCode:      "A999002",
		StockName:      "샘플종목2",
		Amount:         10000,
		AdjustedAmount: 8000,
		TaxAmount:      2000,
		BalanceAmount:  500000,
		DateTime:       "2024-01-15 10:00:00.000",
		SortKey:        "2024-01-15 10:00:00.000|0001",
	},
}

var testOverview = domain.TransactionOverview{
	Market:       "kr",
	OrderableKRW: 100000,
	Withdrawable: []domain.SettlementBucket{
		{Date: "2024-01-15", KRW: 100000},
		{Date: "2024-01-16", KRW: 100000},
	},
	DisplayWithdrawable: []domain.SettlementBucket{
		{Date: "2024-01-15", KRW: 100000},
	},
	Deposit: []domain.SettlementBucket{
		{Date: "2024-01-17", KRW: 1000},
	},
	EstimateSettlement: []domain.SettlementEstimate{
		{Date: "2024-01-16", BuyAmount: 1000},
		{Date: "2024-01-17", SellAmount: 2000},
	},
	WithdrawableBottomSheet: []domain.WithdrawableBottomSheetEntry{
		{Title: "출금가능금액", KRW: 100000},
	},
}

func TestWriteTransactionsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactions(&buf, FormatJSON, testTransactions); err != nil {
		t.Fatalf("WriteTransactions JSON error: %v", err)
	}
	var parsed []domain.Transaction
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 items, got %d", len(parsed))
	}
	if parsed[0].DisplayName != "매수" {
		t.Fatalf("parsed[0].DisplayName = %q", parsed[0].DisplayName)
	}
}

func TestWriteTransactionsCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactions(&buf, FormatCSV, testTransactions); err != nil {
		t.Fatalf("WriteTransactions CSV error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines (header + 2), got %d", len(lines))
	}
	header := lines[0]
	for _, col := range []string{"display_type", "display_name", "adjusted_amount", "sort_key"} {
		if !strings.Contains(header, col) {
			t.Fatalf("CSV header missing column %q: %s", col, header)
		}
	}
	if !strings.Contains(lines[1], "샘플종목1") {
		t.Fatalf("expected 샘플종목1 row, got %s", lines[1])
	}
	if !strings.Contains(lines[2], "배당금입금") {
		t.Fatalf("expected 배당 row, got %s", lines[2])
	}
}

func TestWriteTransactionsTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactions(&buf, FormatTable, testTransactions); err != nil {
		t.Fatalf("WriteTransactions Table error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Transactions: 2") {
		t.Fatalf("expected count header, got %s", output)
	}
	if !strings.Contains(output, "샘플종목1") {
		t.Fatalf("expected 샘플종목1 in table output")
	}
	if !strings.Contains(output, "매수") {
		t.Fatalf("expected 매수 in table output")
	}
	if !strings.Contains(output, "배당금입금") {
		t.Fatalf("expected summary annotation in table output")
	}
}

func TestWriteTransactionsTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactions(&buf, FormatTable, nil); err != nil {
		t.Fatalf("WriteTransactions empty error: %v", err)
	}
	if !strings.Contains(buf.String(), "No transactions in range") {
		t.Fatalf("expected empty-state message, got %s", buf.String())
	}
}

func TestWriteTransactionsUnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactions(&buf, "yaml", testTransactions); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestWriteTransactionsOverviewCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactionsOverview(&buf, FormatCSV, testOverview); err != nil {
		t.Fatalf("WriteTransactionsOverview CSV error: %v", err)
	}
	out := buf.String()
	for _, section := range []string{
		"orderable", "withdrawable", "display_withdrawable", "deposit", "settlement", "withdrawable_breakdown",
	} {
		if !strings.Contains(out, section) {
			t.Fatalf("CSV missing section %q:\n%s", section, out)
		}
	}
}

func TestWriteTransactionsOverviewTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactionsOverview(&buf, FormatTable, testOverview); err != nil {
		t.Fatalf("WriteTransactionsOverview Table error: %v", err)
	}
	out := buf.String()
	for _, heading := range []string{
		"Market: KR", "Orderable:", "Withdrawable:", "Display withdrawable:",
		"Deposit schedule:", "Estimated settlement:", "Withdrawable breakdown:",
	} {
		if !strings.Contains(out, heading) {
			t.Fatalf("Table missing %q:\n%s", heading, out)
		}
	}
	if !strings.Contains(out, "출금가능금액") {
		t.Fatalf("bottom sheet title missing in table:\n%s", out)
	}
}

func TestWriteTransactionsOverviewJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactionsOverview(&buf, FormatJSON, testOverview); err != nil {
		t.Fatalf("WriteTransactionsOverview JSON error: %v", err)
	}
	var parsed domain.TransactionOverview
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if parsed.Market != "kr" || parsed.OrderableKRW != 100000 {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
	if len(parsed.DisplayWithdrawable) != 1 {
		t.Fatalf("display_withdrawable round-trip failed: %+v", parsed.DisplayWithdrawable)
	}
}

func TestWriteTransactionsOverviewUnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTransactionsOverview(&buf, "yaml", testOverview); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
