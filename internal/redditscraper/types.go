package redditscraper

type RedditPost struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Permalink   string  `json:"permalink"`
	Selftext    string  `json:"selftext"`
	Created     float64 `json:"created_utc"`
	NumComments int     `json:"num_comments"`
}

type ScrapeResult struct {
	Post     RedditPost
	Comments []string // TODO: adjust
	Err      error
}

type redditResponse struct {
	Data struct {
		After    string `json:"after"`
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
		Before string `json:"before"`
	} `json:"data"`
}
