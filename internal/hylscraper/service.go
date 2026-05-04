package hylscraper

import (
	"context"
	"log/slog"
	"math"
	"runtime"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/scraper/scraperutils"
	"github.com/chromedp/chromedp"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	Scrape(limit int) (<-chan LinkResult, error)
	StreamLinks(id int64) (<-chan ScrapeData, error)
}

type channels struct {
	linksCh    chan LinkResult
	frontendCh chan LinkResult
	workerCh   chan LinkResult
	resCh      chan ScrapeData
}

type svc struct {
	repo     *repo.Queries
	db       *pgxpool.Pool
	headless bool
	sessions map[int64]channels
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, headless bool) Service {
	return &svc{
		repo:     repo,
		db:       db,
		headless: headless,
	}
}

func (s *svc) Scrape(limit int) (<-chan LinkResult, error) {
	start := time.Now()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.headless),
		chromedp.NoSandbox,
	)

	// Fire and forget
	jobCtx, jobCancel := context.WithTimeout(context.Background(), 60*time.Minute)

	allocCtx, allocCancel := chromedp.NewExecAllocator(jobCtx, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	linksCh := make(chan LinkResult)
	frontendCh := make(chan LinkResult, 100)
	workerCh := make(chan LinkResult, limit)

	// 1. Link Fetcher (Opens 1 Tab)
	go func() {
		defer close(linksCh)
		linksCh <- LinkResult{Status: StatusInitializing}
		if err := getLinks(browserCtx, linksCh, limit); err != nil {
			slog.Error("getLinks failed", "err", err)
		}
	}()

	// 2. Distributor
	go func() {
		defer close(frontendCh)
		defer close(workerCh)
		for link := range linksCh {
			select {
			case frontendCh <- link:
			default:
			}
			select {
			case <-jobCtx.Done():
				return
			case workerCh <- link:
			}
		}
	}()

	// 3. Tab Workers
	numTabs := int(math.Min(float64(runtime.NumCPU()), 5))
	resCh := scraperutils.FanOut(jobCtx, workerCh, numTabs, func(link LinkResult) ScrapeData {
		return scrapeTab(browserCtx, link)
	})

	// 4. Persistence & Cleanup
	go func() {
		defer jobCancel()
		defer browserCancel()
		defer allocCancel()

		idx := 0
		for res := range resCh {
			if res.Permalink != "" {
				slog.Info("hyl scrape data", "id", idx, "url", res.Permalink, "author", res.Author, "title", res.Title, "duration", res.Duration.String())
				idx++
			}
		}
		elapsed := float64(time.Since(start).Milliseconds()) / 1000.0
		slog.Info("scrape job completed and chrome process killed", "duration_s", elapsed)
	}()

	return frontendCh, nil
}

func (s *svc) StreamLinks(id int64) (<-chan ScrapeData, error) {
	return s.sessions[id].resCh, nil
}

func scrapeTab(browserCtx context.Context, linkResult LinkResult) ScrapeData {
	start := time.Now()

	result := ScrapeData{
		Permalink: linkResult.Url,
		Title:     linkResult.Title,
		Author:    linkResult.Author,
	}

	if linkResult.Url == "" {
		return result
	}

	tabCtx, cancelTab := chromedp.NewContext(browserCtx)
	tabCtx, timeoutCancel := context.WithTimeout(tabCtx, time.Second*30)
	defer cancelTab()
	defer timeoutCancel()

	var actions []chromedp.Action
	actions = append(actions, chromedp.Navigate(linkResult.Url))
	actions = append(actions, chromedp.WaitVisible(`body`, chromedp.ByQuery))
	actions = append(actions, chromedp.Sleep(scraperutils.SleepRangeMs(3000, 7000)))

	if result.Title == "" {
		actions = append(actions, chromedp.Title(&result.Title))
	}

	if err := chromedp.Run(tabCtx, actions...); err != nil {
		slog.Error("failed to scrape tab", "url", linkResult.Url, "err", err)
	}

	result.ScrapedAt = time.Now()
	result.Duration = time.Since(start)

	return result
}

func getLinks(browserCtx context.Context, linksCh chan<- LinkResult, limit int) error {
	url := "https://www.hoyolab.com/circles/2/30/feed?page_type=30&page_sort=new"

	if err := chromedp.Run(browserCtx,
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
						linksCh <- LinkResult{Status: StatusFetchComplete}
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
		slog.Error("error while executing chromedp task", "err", err)
		linksCh <- LinkResult{
			Status:       StatusError,
			ErrorMessage: err.Error(),
		}
	}

	time.Sleep(time.Second * 5)
	return nil
}
