package hylscraper

import (
	"context"
	"encoding/json"
	"time"
)

type scraperContext struct {
	id      int64
	context context.Context
	cancel  context.CancelFunc
}

type ScraperStatus string

const (
	StatusInitializing  ScraperStatus = "initializing"
	StatusFetchingLinks ScraperStatus = "fetching"
	StatusFetchComplete ScraperStatus = "done"
	StatusError         ScraperStatus = "error"
)

type ScrapeComment struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

type LinkResult struct {
	Status       ScraperStatus `json:"status"`
	Url          string        `json:"url,omitempty"`
	Title        string        `json:"title,omitempty"`
	Author       string        `json:"author,omitempty"`
	ErrorMessage string        `json:"error,omitempty"`
}

type ScrapeData struct {
	Id        int64         `json:"id"`
	Permalink string        `json:"permalink"`
	Title     string        `json:"title"`
	Author    string        `json:"author"`
	Content   string        `json:"content"`
	ScrapedAt time.Time     `json:"scraped_at"`
	Duration  time.Duration `json:"-"`
}

func (s ScrapeData) MarshalJSON() ([]byte, error) {
	type Alias ScrapeData
	return json.Marshal(&struct {
		*Alias
		Duration string `json:"duration"`
	}{
		Alias:    (*Alias)(&s),
		Duration: s.Duration.String(),
	})
}
