package static

import (
	"testing"
	"time"
)

func TestGetTodaysQuote(t *testing.T) {
	tests := []struct {
		name    string
		date    time.Time
		want    *Quote
		wantErr bool
	}{
		{
			name: "existing date",
			date: time.Date(2024, 12, 12, 0, 0, 0, 0, time.UTC),
			want: &Quote{
				Date:        "2024-12-12",
				Author:      "Oscar Wilde",
				Quote:       "Be yourself; everyone else is already taken.",
				Contributor: "@icco",
				Source:      "",
				SourceURL:   "",
			},
			wantErr: false,
		},
		{
			name:    "non-existing date",
			date:    time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTodaysQuote(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTodaysQuote(%q) error = %v, wantErr %v", tt.date, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Date != tt.want.Date ||
				got.Author != tt.want.Author ||
				got.Quote != tt.want.Quote ||
				got.Contributor != tt.want.Contributor ||
				got.Source != tt.want.Source ||
				got.SourceURL != tt.want.SourceURL {
				t.Errorf("GetTodaysQuote(%q) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestGetContribURL(t *testing.T) {
	tests := []struct {
		contributor string
		want        string
	}{
		{
			contributor: "@icco",
			want:        "https://natwelch.com",
		},
		{
			contributor: "Unknown Contributor",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.contributor, func(t *testing.T) {
			if got := GetContribURL(tt.contributor); got != tt.want {
				t.Errorf("GetContribURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
