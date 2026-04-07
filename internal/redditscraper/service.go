package redditscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/scraper/scraperutils"
	"github.com/chromedp/chromedp"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	GetLinks(ctx context.Context, subreddit string, limit int) ([]RedditPost, error)
	ScrapeAndStore(ctx context.Context, subreddit string, limit int) (<-chan ScrapeResult, error)
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

	slog.Debug("scraper session started")
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

	slog.Debug(fmt.Sprintf("retrieved %d links", lim))
	return posts, nil
}

func (s *svc) ScrapeAndStore(ctx context.Context, subreddit string, limit int) (<-chan ScrapeResult, error) {
	slog.Debug("scraping post", "subreddit", subreddit)
	posts, err := s.GetLinks(ctx, subreddit, limit)
	if err != nil {
		return nil, err
	}

	// source channel
	postStream := make(chan RedditPost, len(posts))
	go func() {
		defer close(postStream)
		for _, p := range posts {
			select {
			case <-ctx.Done():
				return
			case postStream <- p:
			}
		}
	}()

	results := scraperutils.FanOut(ctx, postStream, 5, func(post RedditPost) ScrapeResult {
		comments, err := scrapeComments(ctx, post.URL) // chromedp here
		if err != nil {
			return ScrapeResult{Post: post, Err: err}
		}
		// s.repo.SaveComments(ctx, ...) — your DB stuff
		return ScrapeResult{Post: post, Comments: comments}
	})

	return results, nil
}

func scrapeComments(ctx context.Context, url string) ([]string, error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var comments []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`[data-testid="comment"]`, chromedp.ByQuery),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid="comment"]'))
				.map(el => el.innerText)
		`, &comments),
	)
	return comments, err
}
