package redditscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RedditPost struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	URL         string  `json:"url"`
	Selftext    string  `json:"selftext"`
	Created     float64 `json:"created_utc"`
	NumComments int     `json:"num_comments"`
}

type redditResponse struct {
	Data struct {
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type Service interface {
	GetLinks(context.Context, string, int) ([]RedditPost, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) GetLinks(ctx context.Context, subreddit string, lim int) ([]RedditPost, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*90)
	defer cancel()

	uri := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", subreddit, lim)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ducc/0.1")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit api error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var result redditResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode reddit response: %w", err)
	}

	// Equivalent to jq '[.data.children[].data]'
	posts := make([]RedditPost, len(result.Data.Children))
	for i, child := range result.Data.Children {
		posts[i] = child.Data
	}

	return posts, nil
}
