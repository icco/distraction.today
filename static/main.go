package static

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed *.json
var fs embed.FS

type Quote struct {
	Date        string `json:"date"`
	Author      string `json:"author"`
	Quote       string `json:"quote"`
	Contributor string `json:"contributor"`
}

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
