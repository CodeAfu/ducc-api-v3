package hylscraper

type ScraperStatus string

const (
	Initializing    ScraperStatus = "Initializing Scraper..."
	Loading         ScraperStatus = "Loading..."
	FetchingLinks   ScraperStatus = "Fetching Links..."
	LoadingContents ScraperStatus = "Loading Post Contents..."
)

type ScrapeData struct {
	Permalink string `json:"permalink"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Content   string `json:"content"`
}

type ScrapeResult struct {
	Status ScraperStatus
	Data   ScrapeData
	Err    error
}
