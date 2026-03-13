package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
	IP          string  `json:"ip"`      // Some APIs use 'ip' instead of 'query'
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
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}

		for _, url := range apis {
			info, err := func() (*IPInfo, error) {
				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					return nil, err
				}

				// Some APIs require a User-Agent
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

				resp, err := client.Do(req)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("API %s returned status %d", url, resp.StatusCode)
				}

				var info IPInfo
				if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
					return nil, err
				}
				return &info, nil
			}()

			if err != nil {
				lastErr = err
				continue
			}

			// Standardize query field for different APIs
			if info.Query == "" && info.IP != "" {
				info.Query = info.IP
			}

			if info.Query == "" {
				lastErr = fmt.Errorf("API %s returned no IP address", url)
				continue
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch IP info from all APIs after retries: %v", lastErr)
}
