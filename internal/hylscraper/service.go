package hylscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"sync"
	"time"

	repo "github.com/CodeAfu/go-ducc-api/internal/adapters/postgresql/sqlc"
	"github.com/CodeAfu/go-ducc-api/internal/adapters/scraper/scraperutils"
	"github.com/chromedp/chromedp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	Init(ctx context.Context, email string, limit int) (repo.HylScrapeSession, error)
	// Stream(ctx context.Context, id int64) (idk) // TODO: MAKE ASAP
	// Scrape(sessionId int64, limit int) (<-chan LinkResult, error)
	// StreamLinks(id int64) (<-chan ScrapeData, error)
	Subscribe(ctx context.Context, sessionID int64, send func([]byte)) error
}

// type session struct {
// 	data       repo.HylScrapeSession
// 	linksCh    chan LinkResult
// 	frontendCh chan LinkResult
// 	workerCh   chan LinkResult
// 	resCh      chan ScrapeData
// }

type svc struct {
	repo        *repo.Queries
	db          *pgxpool.Pool
	headless    bool
	subscribers map[int64][]chan LinkResult
	mu          sync.RWMutex
	contexts    []scraperContext
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, headless bool) Service {
	return &svc{
		repo:        repo,
		db:          db,
		headless:    headless,
		subscribers: make(map[int64][]chan LinkResult),
		mu:          sync.RWMutex{},
	}
}

func (s *svc) Init(ctx context.Context, email string, limit int) (repo.HylScrapeSession, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return repo.HylScrapeSession{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)
	session, err := qtx.CreateHylScrapeSession(ctx, email)
	if err != nil {
		return repo.HylScrapeSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.HylScrapeSession{}, err
	}

	// Use context.Background() because s.scrape() is a fire and forget task
	jobCtx, jobCancel := context.WithTimeout(context.Background(), 60*time.Minute)
	s.contexts = append(s.contexts, scraperContext{
		id:      session.ID,
		context: jobCtx,
		cancel:  jobCancel,
	})

	go func() {
		defer s.removeContext(session.ID)

		_, err := s.scrape(jobCtx, jobCancel, session.ID, limit)
		if err != nil {
			slog.Error("error while scraping", "err", err)
			jobCancel()
		}
	}()

	return session, nil
}

func (s *svc) Subscribe(ctx context.Context, sessionID int64, send func([]byte)) error {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	channel := fmt.Sprintf("hyl_scrape_%d", sessionID)
	if _, err := conn.Exec(ctx, "LISTEN "+channel); err != nil {
		return err
	}

	for {
		notif, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		send([]byte("data: " + notif.Payload + "\n\n"))
	}

}

func (s *svc) scrape(jobCtx context.Context, jobCancel context.CancelFunc, sessionId int64, limit int) (<-chan LinkResult, error) {
	start := time.Now()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.headless),
		chromedp.NoSandbox,
	)

	tx, err := s.db.BeginTx(jobCtx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(jobCtx)
	qtx := s.repo.WithTx(tx)

	linksCh := make(chan LinkResult)
	frontendCh := make(chan LinkResult, 100)
	workerCh := make(chan LinkResult, limit)

	abort := func() {
		close(linksCh)
		close(frontendCh)
		close(workerCh)
	}

	session, err := qtx.UpdateHylScraperSession(jobCtx, repo.UpdateHylScraperSessionParams{
		ID:          sessionId,
		ScrapeBegin: pgtype.Timestamptz{Valid: true, Time: start},
	})
	if err != nil {
		abort()
		return nil, err
	}

	if err := tx.Commit(jobCtx); err != nil {
		abort()
		return nil, err
	}

	slog.Info("session", "data", session)

	allocCtx, allocCancel := chromedp.NewExecAllocator(jobCtx, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

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
	numTabs := int(math.Min(float64(runtime.NumCPU()/2+1), 5))
	resCh := scraperutils.FanOut(jobCtx, workerCh, numTabs, func(link LinkResult) ScrapeData {
		res := s.scrapeTab(browserCtx, link)
		res.SessionID = sessionId
		return res
	})

	// 4. Persistence & Cleanup
	go func() {
		defer jobCancel()
		defer browserCancel()
		defer allocCancel()

		idx := 0
		for res := range resCh {
			if res.Permalink == "" {
				continue
			}

			if err := s.persistResult(jobCtx, res); err != nil {
				slog.Error("persist failed", "err", err)
				continue
			}

			slog.Debug("hyl scrape data",
				"idx", idx,
				"url", res.Permalink,
				"author", res.Author,
				"title", res.Title,
				"duration", res.Duration.String(),
			)
			idx++
		}
		elapsed := float64(time.Since(start).Milliseconds()) / 1000.0
		slog.Info("scrape job completed and chrome process killed", "duration_s", elapsed)
	}()

	return frontendCh, nil
}

// func (s *svc) StreamLinks(id int64) (<-chan ScrapeData, error) {
// 	return s.subscribers[id], nil
// }

func (s *svc) scrapeTab(browserCtx context.Context, linkResult LinkResult) ScrapeData {
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
	actions = append(actions, chromedp.Sleep(scraperutils.SleepRangeMs(6000, 10000)))
	// actions = append(actions, chromedp.WaitReady(`div.mhy-reply-list`, chromedp.ByQuery))
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		// var comments []ScrapeComment
		type cardInfo struct {
			Author  string `json:"author"`
			Content string `json:"content"`
		}
		var cardInfoList []cardInfo
		if err := chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.reply-card'))
				.map((card) => {
					const author = document.querySelector('.mhy-account-title__name');
					const content = document.querySelector('replyContentWrapper pre p');
					return {
						author: author ? author.innerText.trim() : '',
						content: content ? content.innerText.trim() : ''
					}
				})
			`, &cardInfoList).Do(ctx); err != nil {
			return err
		}

		// for _, card := range cardInfoList {
		//
		// }
		return nil
	}))

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

func (s *svc) persistResult(jobCtx context.Context, res ScrapeData) error {
	tx, err := s.db.BeginTx(jobCtx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(jobCtx)

	qtx := s.repo.WithTx(tx)

	post, err := qtx.AddHylPost(jobCtx, repo.AddHylPostParams{
		SessionID: res.SessionID,
		Url:       res.Permalink,
		Author:    res.Author,
		Title:     res.Title,
		Content:   res.Content,
	})
	if err != nil {
		return fmt.Errorf("AddHylPost: %w", err)
	}

	for _, c := range res.Comments {
		_, err := qtx.AddHylComment(jobCtx, repo.AddHylCommentParams{
			SessionID:       res.SessionID,
			PostID:          post.ID,
			ParentCommentID: pgtype.Int8{Valid: false}, // TODO: impl properly
			Url:             res.Permalink,
			Author:          c.Author,
			Content:         c.Content,
		})
		if err != nil {
			// tx.Rollback(jobCtx)
			slog.Error("AddHylComment failed", "err", err)
			// continue
		}
	}

	channel := fmt.Sprintf("hyl_scrape_%d", res.SessionID)

	payload, err := json.Marshal(post)
	if err != nil {
		slog.Error("Failed to marshal json for post", "err", err, "postId", post.ID)
	}
	_, err = tx.Exec(jobCtx,
		"SELECT pg_notify($1, $2)", channel, string(payload),
	)
	if err != nil {
		slog.Error("pg_notify failed", "err", err)
	}

	return tx.Commit(jobCtx)
}

// func (s *svc) publish(id int64, res LinkResult) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	for _, ch := range s.subscribers[id] {
// 		select {
// 		case ch <- res:
// 		default:
// 		}
// 	}
// }

func (s *svc) removeContext(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contexts := s.contexts[:0]
	for _, c := range s.contexts {
		if c.id != id {
			contexts = append(contexts, c)
		}
	}
	s.contexts = contexts
}
