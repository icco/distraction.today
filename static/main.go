package static

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed *.json
var fs embed.FS

// Quote represents a daily quote with its metadata.
type Quote struct {
	Date        string `json:"date"`        // Date in YYYY-MM-DD format
	Author      string `json:"author"`      // Author of the quote
	Quote       string `json:"quote"`       // The actual quote text
	Contributor string `json:"contributor"` // Person who contributed the quote
	Source      string `json:"source"`      // Source of the quote
	SourceURL   string `json:"source_url"`  // URL to the quote source
}

// GetTodaysQuote returns the quote for the specified date.
// It returns an error if no quote is found for the given date or if there's an issue reading the quotes file.
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

// GetContribURL returns the URL associated with a contributor.
// It returns an empty string if the contributor is not found or if there's an issue reading the contributors file.
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
