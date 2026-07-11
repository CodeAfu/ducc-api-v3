package hylscraper

import (
	"context"
	"encoding/json"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
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

type DataType string

const (
	TypePost    DataType = "post"
	TypeComment DataType = "comment"
)

type LinkResult struct {
	Status       ScraperStatus `json:"status"`
	Url          string        `json:"url,omitempty"`
	Title        string        `json:"title,omitempty"`
	Author       string        `json:"author,omitempty"`
	ErrorMessage string        `json:"error,omitempty"`
}

type ScrapeData struct {
	SessionID int64
	Permalink string
	Title     string
	Author    string
	Content   string
	Comments  []ScrapeComment
	ScrapedAt time.Time
	Duration  time.Duration
}

type ScrapeComment struct {
	ParentCommentID int64
	Url             string
	Author          string
	Content         string
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

type NotifyPayload struct {
	SessionID   int64            `json:"session_id"`
	PayloadType DataType         `json:"payload_type"`
	Post        *repo.HylPost    `json:"post,omitempty"`
	Comment     *repo.HylComment `json:"comment,omitempty"`
}
