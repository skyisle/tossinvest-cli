package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type watchlistData struct {
	Groups []struct {
		Name  string `json:"name"`
		Items []struct {
			StockCode string `json:"stockCode"`
			StockName string `json:"stockName"`
			Prices    struct {
				Currency string   `json:"currency"`
				Base     *float64 `json:"base"`
				Close    *float64 `json:"close"`
			} `json:"prices"`
		} `json:"items"`
	} `json:"groups"`
}

func (c *Client) ListWatchlist(ctx context.Context) ([]domain.WatchlistItem, error) {
	if err := c.requireSession(); err != nil {
		return nil, err
	}

	var envelope assetSectionsEnvelope
	if err := c.postJSON(ctx, c.certBaseURL+"/api/v2/dashboard/asset/sections/all", json.RawMessage("{}"), &envelope); err != nil {
		return nil, err
	}

	var watchlist watchlistData
	found := false
	for _, section := range envelope.Result.Sections {
		if section.Type != "WATCHLIST" {
			continue
		}
		if err := json.Unmarshal(section.Data, &watchlist); err != nil {
			return nil, err
		}
		found = true
		break
	}
	if !found {
		return nil, fmt.Errorf("WATCHLIST section not found")
	}

	items := []domain.WatchlistItem{}
	for _, group := range watchlist.Groups {
		for _, item := range group.Items {
			items = append(items, domain.WatchlistItem{
				Group:    group.Name,
				Symbol:   item.StockCode,
				Name:     item.StockName,
				Currency: item.Prices.Currency,
				Base:     coalesceMoney(item.Prices.Base),
				Last:     coalesceMoney(item.Prices.Close),
			})
		}
	}

	return items, nil
}
