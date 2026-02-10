package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IPInfo holds the response from the IP geolocation API
type IPInfo struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
	Message     string  `json:"message"` // For error messages
}

// FetchIPInfo fetches IP and geolocation information using the provided http.Client
func FetchIPInfo(ctx context.Context, client *http.Client) (*IPInfo, error) {
	// Try multiple APIs in case one is blocked or down
	apis := []string{
		"http://ip-api.com/json/?lang=zh-CN",
		"https://api.ip.sb/geoip",
		"https://ipapi.co/json/",
	}

	var lastErr error
	for _, url := range apis {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		// Some APIs require a User-Agent
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API %s returned status %d", url, resp.StatusCode)
			continue
		}

		var info IPInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			lastErr = err
			continue
		}

		// Standardize query field for different APIs if necessary
		if info.Query == "" && info.Status == "" {
			// Some APIs use different fields, handle them if needed
			// For simplicity, we assume ip-api format or similar
		}

		return &info, nil
	}

	return nil, fmt.Errorf("failed to fetch IP info from all APIs: %v", lastErr)
}
