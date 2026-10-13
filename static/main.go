// Package static provides embedded daily quotes and helpers for accessing them.
package static

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

//go:embed *.json
var fs embed.FS

// Quote is a single daily quote.
type Quote struct {
	Date        string `json:"date"`
	Author      string `json:"author"`
	Quote       string `json:"quote"`
	Contributor string `json:"contributor"`
	Source      string `json:"source"`
	SourceURL   string `json:"source_url"`
}

var loadQuotes = sync.OnceValues(func() ([]Quote, error) {
	data, err := fs.ReadFile("quotes.json")
	if err != nil {
		return nil, err
	}

	quotes := []Quote{}
	if err := json.Unmarshal(data, &quotes); err != nil {
		return nil, err
	}

	sort.SliceStable(quotes, func(i, j int) bool { return quotes[i].Date < quotes[j].Date })
	return quotes, nil
})

var loadContributors = sync.OnceValues(func() (map[string]string, error) {
	data, err := fs.ReadFile("contributors.json")
	if err != nil {
		return nil, err
	}

	contributors := map[string]string{}
	if err := json.Unmarshal(data, &contributors); err != nil {
		return nil, err
	}
	return contributors, nil
})

// GetTodaysQuote returns the quote dated date, or an error if none.
func GetTodaysQuote(date time.Time) (*Quote, error) {
	today := date.Format("2006-01-02")
	quotes, err := loadQuotes()
	if err != nil {
		return nil, err
	}

	for _, quote := range quotes {
		if quote.Date == today {
			return &quote, nil
		}
	}

	return nil, fmt.Errorf("no quote found for date %q", today)
}

// GetLatestQuote returns the most recent quote available.
func GetLatestQuote() (*Quote, error) {
	quotes, err := GetQuotes()
	if err != nil {
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no quotes available")
	}
	return quotes[len(quotes)-1], nil
}

// GetQuotes returns all embedded quotes dated before now, sorted by date ascending.
func GetQuotes() ([]*Quote, error) {
	quotes, err := loadQuotes()
	if err != nil {
		return nil, err
	}

	var ret []*Quote
	for _, quote := range quotes {
		date, err := time.Parse("2006-01-02", quote.Date)
		if err != nil {
			continue
		}

		if date.Before(time.Now()) {
			ret = append(ret, &quote)
		}
	}

	return ret, nil
}

// GetContribURL returns the URL for a contributor, or "" if unknown.
func GetContribURL(contributor string) string {
	contributors, err := loadContributors()
	if err != nil {
		return ""
	}

	return contributors[contributor]
}
