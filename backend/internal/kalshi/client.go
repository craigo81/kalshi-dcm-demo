// Package kalshi provides integration with Kalshi's public API.
// Core Principle 3: Uses economic binaries (Fed rates, etc.) which are
// not readily susceptible to manipulation.
package kalshi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// CLIENT CONFIGURATION
// =============================================================================

const (
	// Production API (all markets)
	DefaultBaseURL = "https://api.elections.kalshi.com/trade-api/v2"
	// Alternative URL
	TradingBaseURL = "https://trading-api.kalshi.com/trade-api/v2"
)

// Client handles communication with Kalshi's public API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Kalshi API client.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// =============================================================================
// API RESPONSE TYPES
// =============================================================================

type MarketsResponse struct {
	Markets []KalshiMarketResponse `json:"markets"`
	Cursor  string                 `json:"cursor"`
}

type KalshiMarketResponse struct {
	Ticker         string `json:"ticker"`
	EventTicker    string `json:"event_ticker"`
	SeriesTicker   string `json:"series_ticker"`
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	Status         string `json:"status"`
	Category       string `json:"category"`
	YesBid         int    `json:"yes_bid"`
	YesAsk         int    `json:"yes_ask"`
	NoBid          int    `json:"no_bid"`
	NoAsk          int    `json:"no_ask"`
	LastPrice      int    `json:"last_price"`
	Volume         int64  `json:"volume"`
	Volume24H      int64  `json:"volume_24h"`
	OpenInterest   int64  `json:"open_interest"`
	OpenTime       string `json:"open_time"`
	CloseTime      string `json:"close_time"`
	ExpirationTime string `json:"expiration_time"`
	SettlementValue *int  `json:"settlement_value,omitempty"`
	Result         string `json:"result,omitempty"`
}

type EventsResponse struct {
	Events []EventResponse `json:"events"`
	Cursor string          `json:"cursor"`
}

type EventResponse struct {
	EventTicker       string `json:"event_ticker"`
	SeriesTicker      string `json:"series_ticker"`
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	Category          string `json:"category"`
	MutuallyExclusive bool   `json:"mutually_exclusive"`
}

type OrderbookResponse struct {
	Orderbook struct {
		Ticker  string           `json:"ticker"`
		YesBids []OrderbookLevel `json:"yes"`
		NoBids  []OrderbookLevel `json:"no"`
	} `json:"orderbook"`
}

type OrderbookLevel struct {
	Price    int `json:"price"`
	Quantity int `json:"quantity"`
}

type SeriesResponse struct {
	Series []SeriesItem `json:"series"`
	Cursor string       `json:"cursor"`
}

type SeriesItem struct {
	SeriesTicker string `json:"series_ticker"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	Frequency    string `json:"frequency"`
}

// =============================================================================
// PUBLIC API METHODS
// =============================================================================

// GetMarkets fetches markets with optional filters.
// Core Principle 3: Focus on economic binaries for low manipulation risk.
func (c *Client) GetMarkets(params MarketParams) (*MarketsResponse, error) {
	endpoint := "/markets"
	queryParams := params.ToQueryParams()
	if queryParams != "" {
		endpoint += "?" + queryParams
	}

	var response MarketsResponse
	if err := c.doRequest("GET", endpoint, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetMarket fetches a single market by ticker.
func (c *Client) GetMarket(ticker string) (*KalshiMarketResponse, error) {
	endpoint := fmt.Sprintf("/markets/%s", url.PathEscape(ticker))

	var response struct {
		Market KalshiMarketResponse `json:"market"`
	}
	if err := c.doRequest("GET", endpoint, &response); err != nil {
		return nil, err
	}

	return &response.Market, nil
}

// GetEvents fetches events with optional filters.
func (c *Client) GetEvents(status string, limit int, cursor string) (*EventsResponse, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	endpoint := "/events"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var response EventsResponse
	if err := c.doRequest("GET", endpoint, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetOrderbook fetches the orderbook for a market.
// Core Principle 9: Transparency in order execution.
func (c *Client) GetOrderbook(ticker string, depth int) (*OrderbookResponse, error) {
	endpoint := fmt.Sprintf("/markets/%s/orderbook", url.PathEscape(ticker))
	if depth > 0 {
		endpoint += fmt.Sprintf("?depth=%d", depth)
	}

	var response OrderbookResponse
	if err := c.doRequest("GET", endpoint, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetSeries fetches series list.
func (c *Client) GetSeries(cursor string, limit int) (*SeriesResponse, error) {
	params := url.Values{}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	endpoint := "/series"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var response SeriesResponse
	if err := c.doRequest("GET", endpoint, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (c *Client) doRequest(method, endpoint string, result interface{}) error {
	reqURL := c.baseURL + endpoint

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// ToMarket converts API response to internal model.
// Core Principle 3: Classify risk category for economic binaries.
func (m *KalshiMarketResponse) ToMarket() models.KalshiMarket {
	market := models.KalshiMarket{
		Ticker:          m.Ticker,
		EventTicker:     m.EventTicker,
		SeriesTicker:    m.SeriesTicker,
		Title:           m.Title,
		Subtitle:        m.Subtitle,
		Status:          models.MarketStatus(m.Status),
		Category:        m.Category,
		YesBid:          m.YesBid,
		YesAsk:          m.YesAsk,
		NoBid:           m.NoBid,
		NoAsk:           m.NoAsk,
		LastPrice:       m.LastPrice,
		Volume:          m.Volume,
		Volume24H:       m.Volume24H,
		OpenInterest:    m.OpenInterest,
		SettlementValue: m.SettlementValue,
		Result:          m.Result,
	}

	// Parse times
	if t, err := time.Parse(time.RFC3339, m.OpenTime); err == nil {
		market.OpenTime = t
	}
	if t, err := time.Parse(time.RFC3339, m.CloseTime); err == nil {
		market.CloseTime = t
	}
	if t, err := time.Parse(time.RFC3339, m.ExpirationTime); err == nil {
		market.ExpirationTime = t
	}

	// Core Principle 3: Classify risk based on category
	market.RiskCategory = classifyRisk(m.Category, m.SeriesTicker)

	return market
}

// classifyRisk determines manipulation risk for a market.
// Core Principle 3: Economic binaries (Fed, GDP, CPI) are low-risk.
func classifyRisk(category, seriesTicker string) string {
	lowRiskCategories := map[string]bool{
		"Economics":    true,
		"Fed":          true,
		"Interest":     true,
		"Inflation":    true,
		"GDP":          true,
		"Unemployment": true,
		"CPI":          true,
	}

	lowRiskSeries := map[string]bool{
		"FED":       true,
		"FOMC":      true,
		"CPI":       true,
		"GDP":       true,
		"UNEMP":     true,
		"INFLATION": true,
	}

	if lowRiskCategories[category] || lowRiskSeries[seriesTicker] {
		return "low"
	}

	// Political events have medium risk
	if category == "Politics" || category == "Elections" {
		return "medium"
	}

	// Entertainment, sports have higher risk
	return "high"
}

// =============================================================================
// QUERY PARAMETERS
// =============================================================================

type MarketParams struct {
	Limit        int
	Cursor       string
	Status       string
	SeriesTicker string
	EventTicker  string
	Category     string
}

func (p MarketParams) ToQueryParams() string {
	params := url.Values{}

	if p.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", p.Limit))
	}
	if p.Cursor != "" {
		params.Set("cursor", p.Cursor)
	}
	if p.Status != "" {
		params.Set("status", p.Status)
	}
	if p.SeriesTicker != "" {
		params.Set("series_ticker", p.SeriesTicker)
	}
	if p.EventTicker != "" {
		params.Set("event_ticker", p.EventTicker)
	}

	return params.Encode()
}
