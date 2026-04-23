package hylscraper

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	// _ "embed"

	"github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/scraper/scraperutils"
	// "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/jackc/pgx/v5/pgxpool"
)

// //go:embed stealth.min.js
// var stealthJS string

type HylService interface {
	Scrape(ctx context.Context, limit int) (<-chan string, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewService(repo *repo.Queries, db *pgxpool.Pool) HylService {
	return &svc{
		repo: repo,
		db:   db,
	}
}

func (s *svc) Scrape(ctx context.Context, limit int) (<-chan string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx, chromedp.WithDebugf(func(f string, v ...interface{}) {
		slog.Debug(fmt.Sprintf(f, v...))
	}))

	linksCh := make(chan string)
	go func() {
		defer cancel()
		defer cancelBrowser()
		defer close(linksCh)

		err := getLinks(browserCtx, linksCh, limit)
		if err != nil {
			slog.Error("getLinks failed", "err", err)
			return
		}
	}()

	return linksCh, nil
}

func getLinks(ctx context.Context, linksCh chan<- string, limit int) error {
	url := "https://www.hoyolab.com/circles/2/30/feed?page_type=30&page_sort=new"
	taskCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	if err := chromedp.Run(taskCtx,
		// chromedp.ActionFunc(func(ctx context.Context) error {
		// 	_, err := page.AddScriptToEvaluateOnNewDocument(stealthJS).Do(ctx)
		// 	return err
		// }),
		chromedp.Navigate(url),
		chromedp.WaitVisible(`button.el-dialog__headerbtn`),
		chromedp.Sleep(scraperutils.SleepRangeMs(300, 700)),
		chromedp.Click(`button.el-dialog__headerbtn`),
		chromedp.Sleep(scraperutils.SleepRangeMs(1000, 1200)),
		chromedp.WaitVisible(`//button[.//span[text()="Skip"]]`, chromedp.BySearch),
		chromedp.Evaluate(`document.querySelector('button.normal__quaternary').click()`, nil),
		chromedp.WaitVisible(`div.mhy-article-list__body`),
		chromedp.Sleep(scraperutils.SleepRangeMs(1000, 2000)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			seen := map[string]bool{}
			for {
				var current []string
				if err := chromedp.Evaluate(`
					Array.from(document.querySelectorAll('a.mhy-article-card__link'))
						.map(a => a.getAttribute('href'))
						.filter(href => href && href.startsWith('/article/'))
						.map(href => 'https://www.hoyolab.com' + href)
				`, &current).Do(ctx); err != nil {
					return err
				}
				for _, link := range current {
					if !seen[link] {
						seen[link] = true
						linksCh <- link
					}
					if len(seen) >= limit {
						return nil
					}
				}
				if len(seen) >= limit {
					break
				}
				chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil).Do(ctx)
				time.Sleep(scraperutils.SleepRangeMs(1500, 2500))
			}
			return nil
		}),
	); err != nil {
		slog.Error(fmt.Sprintf("error while executing chromedp task: %v", err))
	}

	time.Sleep(time.Second * 5)
	slog.Debug("retrieved links", "url", url, "limit", limit)

	return nil
}
