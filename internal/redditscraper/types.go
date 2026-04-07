package redditscraper

type RedditPost struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	URL         string  `json:"url"`
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
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}
