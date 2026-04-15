package redditscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	_ "embed"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/scraper/scraperutils"
	"github.com/CodeAfu/go-ducc-api/internal/capsolver"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed stealth.min.js
var stealthJS string

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
		posts[i].Permalink = "https://www.reddit.com" + child.Data.Permalink
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
	slog.Debug("posts", "data", posts)

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

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			chromedp.Flag("headless", false),
		)...,
	)

	results := scraperutils.FanOut(ctx, postStream, 5, func(post RedditPost) ScrapeResult {
		slog.Debug("scraping comments", "url", post.Permalink)
		time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		comments, err := scrapeComments(allocCtx, post.Permalink) // chromedp here
		slog.Debug("post scraped", "comments", comments)
		if err != nil {
			return ScrapeResult{Post: post, Err: err}
		}
		// s.repo.SaveComments(ctx, ...) — your DB stuff
		return ScrapeResult{Post: post, Comments: comments}
	})

	out := make(chan ScrapeResult, cap(results))
	go func() {
		defer allocCancel()
		defer close(out)
		for r := range results {
			out <- r
		}
	}()

	return out, nil
}

func scrapeComments(allocCtx context.Context, url string) ([]string, error) {
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var comments []string
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(stealthJS).Do(ctx)
			return err
		}),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			captchaAttempts := 0
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				var isBotCheck bool
				chromedp.Evaluate(`document.querySelector('iframe[src*="recaptcha"]') !== null`, &isBotCheck).Do(ctx)
				if isBotCheck {
					if captchaAttempts >= 3 {
						return fmt.Errorf("captcha solve failed after %d attempts", captchaAttempts)
					}
					captchaAttempts++

					var siteKey string
					chromedp.Evaluate(`document.querySelector('iframe[src*="recaptcha"]')?.src.match(/k=([^&]+)/)?.[1]`, &siteKey).Do(ctx)

					token, err := capsolver.SolveCaptchaV2Task(siteKey, url)
					if err != nil {
						return fmt.Errorf("captcha solve failed: %w", err)
					}
					slog.Debug("capsolver", "attempt", captchaAttempts, "siteKey", siteKey, "url", url, "token", token)

					time.Sleep(time.Duration(2+rand.Intn(4)) * time.Second)
					chromedp.Evaluate(fmt.Sprintf(`
						document.getElementById('g-recaptcha-response').innerHTML = '%s';
						document.getElementById('g-recaptcha-response-100000').innerHTML = '%s';
						var cb = document.querySelector('[data-callback]')?.getAttribute('data-callback');
						if (cb) window[cb]('%s');
					`, token, token, token), nil).Do(ctx)

					time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
					continue
				}

				var hasComments bool
				chromedp.Evaluate(`document.querySelectorAll('shreddit-comment').length > 0`, &hasComments).Do(ctx)
				if hasComments {
					return nil
				}

				time.Sleep(time.Duration(1500+rand.Intn(2500)) * time.Millisecond)
			}
		}),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('shreddit-comment'))
				.map(el => el.innerText)
		`, &comments),
	)

	return comments, err
}
