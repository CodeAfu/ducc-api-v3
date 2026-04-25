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
	Scrape(ctx context.Context, limit int) (<-chan LinkResult, error)
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

func (s *svc) Scrape(ctx context.Context, limit int) (<-chan LinkResult, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx, chromedp.WithDebugf(func(f string, v ...interface{}) {
		slog.Debug(fmt.Sprintf(f, v...))
	}))

	linksCh := make(chan LinkResult)
	frontendCh := make(chan LinkResult)
	workerCh := make(chan LinkResult)

	go func() {
		defer close(linksCh)

		linksCh <- LinkResult{Status: StatusInitializing}

		err := getLinks(browserCtx, linksCh, limit)
		if err != nil {
			slog.Error("getLinks failed", "err", err)
			return
		}
	}()

	go func() {
		defer close(frontendCh)
		defer close(workerCh)
		for link := range linksCh {
			select {
			case <-ctx.Done():
				return
			case frontendCh <- link:
			}

			select {
			case <-ctx.Done():
				return
			case workerCh <- link:
			}
		}
	}()

	resCh := scraperutils.FanOut(ctx, workerCh, 5, func(link LinkResult) ScrapeData {
		return scrapeTab(browserCtx, link)
	})

	go func() {
		defer cancel()
		defer cancelBrowser()
		for res := range resCh {
			slog.Info("successfully scraped tab", "url", res.Permalink)
		}
	}()

	return frontendCh, nil
}

func scrapeTab(parentCtx context.Context, linkResult LinkResult) ScrapeData {
	result := ScrapeData{
		Permalink: linkResult.Url,
		Title:     linkResult.Title,
		Author:    linkResult.Author,
	}
	if linkResult.Url == "" {
		return result
	}

	tabCtx, cancelTab := chromedp.NewContext(parentCtx)
	defer cancelTab()

	var actions []chromedp.Action
	actions = append(actions, chromedp.Navigate(linkResult.Url))
	actions = append(actions, chromedp.Sleep(scraperutils.SleepRangeMs(3000, 5000)))

	if result.Title == "" {
		actions = append(actions, chromedp.Title(&result.Title))
	}

	if err := chromedp.Run(tabCtx, actions...); err != nil {
		slog.Error("failed to scrape tab", "url", linkResult.Url, "err", err)
	}

	return result
}

func getLinks(ctx context.Context, linksCh chan<- LinkResult, limit int) error {
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
				type result struct {
					URL    string `json:"url"`
					Title  string `json:"title"`
					Author string `json:"author"`
				}
				var cur []result
				// Get url, title and author
				if err := chromedp.Evaluate(`
					Array.from(document.querySelectorAll('.mhy-article-card'))
						.map(card => {
							const link = card.querySelector('a.mhy-article-card__link');
							const title = card.querySelector('.mhy-article-card__text');
							const author = card.querySelector('.mhy-account-title__name');
							return {
								url: link ? 'https://www.hoyolab.com' + link.getAttribute('href') : '',
								title: title ? title.innerText.trim() : '',
								author: author ? author.innerText.trim() : ''
							};
						})
						.filter(res => res.url && res.url.includes('/article/'))
				`, &cur).Do(ctx); err != nil {
					return err
				}

				for _, r := range cur {
					if !seen[r.URL] {
						seen[r.URL] = true
						linksCh <- LinkResult{
							Status: StatusFetchingLinks,
							Url:    r.URL,
							Title:  r.Title,
							Author: r.Author,
						}
					}
					if len(seen) >= limit {
						linksCh <- LinkResult{
							Status: StatusFetchComplete,
						}
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
		linksCh <- LinkResult{
			Status:       StatusError,
			ErrorMessage: err.Error(),
		}
	}

	time.Sleep(time.Second * 5)
	slog.Debug("retrieved links", "url", url, "limit", limit)

	return nil
}
