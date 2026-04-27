package static

import (
	"embed"
	"encoding/json"
	"fmt"
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

// GetTodaysQuote returns the quote dated date, or an error if none.
func GetTodaysQuote(date time.Time) (*Quote, error) {
	today := date.Format("2006-01-02")
	file, err := fs.Open("quotes.json")
	if err != nil {
		return nil, err
	}

	quotes := []Quote{}
	if err := json.NewDecoder(file).Decode(&quotes); err != nil {
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

func GetQuotes() ([]*Quote, error) {
	file, err := fs.Open("quotes.json")
	if err != nil {
		return nil, err
	}

	quotes := []Quote{}
	if err := json.NewDecoder(file).Decode(&quotes); err != nil {
		return nil, err
	}

	var ret []*Quote
	for _, quote := range quotes {
		date, err := time.Parse("2006-01-02", quote.Date)
		if err != nil {
			continue
		}

		if date.IsZero() {
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
	file, err := fs.Open("contributors.json")
	if err != nil {
		return ""
	}

	contributors := map[string]string{}
	if err := json.NewDecoder(file).Decode(&contributors); err != nil {
		return ""
	}

	return contributors[contributor]
}
