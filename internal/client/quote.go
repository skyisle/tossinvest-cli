package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type quoteEnvelope[T any] struct {
	Result T `json:"result"`
}

type stockInfoResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	Market   struct {
		Code        string `json:"code"`
		DisplayName string `json:"displayName"`
	} `json:"market"`
}

type stockDetailCommonResult struct {
	Badges  []json.RawMessage `json:"badges"`
	Notices []json.RawMessage `json:"notices"`
}

type stockPriceResult struct {
	ProductCode string  `json:"productCode"`
	Exchange    string  `json:"exchange"`
	Currency    string  `json:"currency"`
	Base        float64 `json:"base"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
}

type stockSearchEnvelope struct {
	Result struct {
		Stocks []struct {
			StockCode string `json:"stockCode"`
			StockName string `json:"stockName"`
			MatchType string `json:"matchType"`
		} `json:"stocks"`
	} `json:"result"`
}

func (c *Client) GetQuote(ctx context.Context, symbol string) (domain.Quote, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.Quote{}, err
	}

	info, err := c.getStockInfo(ctx, productCode)
	if err != nil {
		return domain.Quote{}, err
	}

	price, err := c.getStockPrice(ctx, productCode)
	if err != nil {
		return domain.Quote{}, err
	}

	detail, err := c.getStockDetailCommon(ctx, productCode)
	if err != nil {
		detail = nil
	}

	quote := domain.Quote{
		ProductCode:    price.ProductCode,
		Symbol:         info.Symbol,
		Name:           info.Name,
		MarketCode:     info.Market.Code,
		Market:         info.Market.DisplayName,
		Currency:       firstNonEmpty(price.Currency, info.Currency),
		ReferencePrice: price.Base,
		Last:           price.Close,
		Change:         price.Close - price.Base,
		Volume:         price.Volume,
		Status:         info.Status,
		FetchedAt:      time.Now().UTC(),
	}

	if price.Base != 0 {
		quote.ChangeRate = quote.Change / price.Base
	}

	if detail != nil {
		quote.BadgeCount = len(detail.Badges)
		quote.NoticeCount = len(detail.Notices)
	}

	return quote, nil
}

func (c *Client) resolveProductCode(ctx context.Context, symbol string) (string, error) {
	normalized := normalizeProductCode(symbol)
	if normalized == "" {
		return "", fmt.Errorf("symbol is required")
	}
	if looksLikeProductCode(normalized) {
		return normalized, nil
	}

	var envelope stockSearchEnvelope
	body := []byte(fmt.Sprintf(`{"query":%q}`, normalized))
	if err := c.postJSON(ctx, fmt.Sprintf("%s/api/v2/search/stocks", c.infoBaseURL), body, &envelope); err != nil {
		return "", err
	}
	if len(envelope.Result.Stocks) == 0 {
		return "", fmt.Errorf("no product code result returned for %s", normalized)
	}
	return envelope.Result.Stocks[0].StockCode, nil
}

func (c *Client) getStockInfo(ctx context.Context, productCode string) (stockInfoResult, error) {
	var envelope quoteEnvelope[stockInfoResult]
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/v2/stock-infos/%s", c.infoBaseURL, productCode), &envelope); err != nil {
		return stockInfoResult{}, err
	}
	return envelope.Result, nil
}

func (c *Client) getStockDetailCommon(ctx context.Context, productCode string) (*stockDetailCommonResult, error) {
	var envelope quoteEnvelope[stockDetailCommonResult]
	if err := c.getJSON(
		ctx,
		fmt.Sprintf("%s/api/v1/stock-detail/ui/%s/common", c.infoBaseURL, productCode),
		&envelope,
	); err != nil {
		return nil, err
	}
	return &envelope.Result, nil
}

func (c *Client) getStockPrice(ctx context.Context, productCode string) (stockPriceResult, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/product/stock-prices", c.infoBaseURL))
	if err != nil {
		return stockPriceResult{}, err
	}

	query := endpoint.Query()
	query.Set("meta", "true")
	query.Set("productCodes", productCode)
	endpoint.RawQuery = query.Encode()

	var envelope quoteEnvelope[[]stockPriceResult]
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return stockPriceResult{}, err
	}

	if len(envelope.Result) == 0 {
		return stockPriceResult{}, fmt.Errorf("no price result returned for %s", productCode)
	}

	return envelope.Result[0], nil
}

func normalizeProductCode(symbol string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(symbol))
	if trimmed == "" {
		return trimmed
	}

	if len(trimmed) == 6 && trimmed[0] >= '0' && trimmed[0] <= '9' {
		return "A" + trimmed
	}

	return trimmed
}

func looksLikeProductCode(value string) bool {
	if len(value) == 7 && value[0] == 'A' {
		return true
	}
	if len(value) >= 8 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z' {
		hasDigit := false
		for i := 2; i < len(value); i++ {
			if value[i] >= '0' && value[i] <= '9' {
				hasDigit = true
				continue
			}
			return false
		}
		return hasDigit
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
