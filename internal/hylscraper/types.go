package hylscraper

type ScraperStatus string

const (
	StatusInitializing  ScraperStatus = "initializing"
	StatusFetchingLinks ScraperStatus = "fetching"
	StatusFetchComplete ScraperStatus = "done"
	StatusError         ScraperStatus = "error"
)

type ScrapeData struct {
	Permalink string `json:"permalink"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Content   string `json:"content"`
}

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
