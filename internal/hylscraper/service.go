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
	s.mu.Lock()
	s.contexts = append(s.contexts, scraperContext{
		id:      session.ID,
		context: jobCtx,
		cancel:  jobCancel,
	})
	s.mu.Unlock()

	go func() {
		defer s.removeContext(session.ID)

		err := s.scrape(jobCtx, jobCancel, session.ID, limit)
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
		send([]byte(notif.Payload))
	}
}

func (s *svc) scrape(jobCtx context.Context, jobCancel context.CancelFunc, sessionId int64, limit int) error {
	start := time.Now()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.headless),
		chromedp.NoSandbox,
	)

	tx, err := s.db.BeginTx(jobCtx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(jobCtx)
	qtx := s.repo.WithTx(tx)

	linksCh := make(chan LinkResult)
	workerCh := make(chan LinkResult, limit)

	abort := func() {
		close(linksCh)
		close(workerCh)
	}

	session, err := qtx.UpdateHylScraperSession(jobCtx, repo.UpdateHylScraperSessionParams{
		ID:          sessionId,
		ScrapeBegin: pgtype.Timestamptz{Valid: true, Time: start},
	})
	if err != nil {
		abort()
		return err
	}

	if err := tx.Commit(jobCtx); err != nil {
		abort()
		return err
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
		defer close(workerCh)
		for link := range linksCh {
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

			post, err := s.persistPost(jobCtx, res)
			if err != nil {
				slog.Error("persist post failed", "err", err)
				continue
			}

			_, err = s.persistComment(jobCtx, res, post.ID)
			if err != nil {
				slog.Error("persist comment failed", "err", err)
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

	return nil
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
	actions = append(actions, chromedp.Sleep(scraperutils.SleepRangeMs(3000, 4200)))
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		// Get Post
		var postContent string
		if err := chromedp.Evaluate(`
			(() => {
				const isImageContent = document.querySelector('.mhy-image-article');
				const postContent = isImageContent
					? (document.querySelector('.mhy-image-article__describe')?.innerText.trim() || '')
					: Array.from(document.querySelectorAll('.mhy-image-text-article__content p'))
						.map(p => p.innerText.trim())
						.join('\n');
				return postContent;
			})()
			`, &postContent).Do(ctx); err != nil {
			return err
		}
		result.Content = postContent
		return nil
	}))
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		// Get Comments
		type comment struct {
			Author  string `json:"author"`
			Content string `json:"content"`
		}
		type pageInfo struct {
			Comments []comment `json:"comments"`
		}
		var pageData pageInfo
		var noMore bool
		seen := map[string]struct{}{}
		var allComments []comment

		nullScrolls := 0
		for {
			// Scrape
			if err := chromedp.Evaluate(`
			(async () => {
				const sleep = async (duration) => {
					await new Promise(r => setTimeout(r, duration));
				}

				const scrollToBottom = async (el, maxAttempts = 20) => {
					let lastHeight = el.scrollHeight;
					for (let i = 0; i < maxAttempts; i++) {
						el.scrollTop = el.scrollHeight;
						await sleep(500);
						if (el.scrollHeight === lastHeight) break; // no new content loaded
						lastHeight = el.scrollHeight;
					}
				};

				const getCommentsFromDialog = async () => {
					comments = [];
					const dialog = document.querySelector('.mhy-dialog__body');
					
					// Parent Comment
					const parentAuthorEl = dialog.querySelector('.mhy-account-title__name');
					const parentCommentEls = dialog.querySelector('.replyContentWrapper pre p');
					const parentAuthor = parentAuthorEl ? parentAuthorEl.innerText.trim() : '';
					const parentComment = Array.from(parentCommentEls)
						.map(p => p.innerText.trim())
						.join('\n');
					comments.push({ author: parentAuthor, content: parentComment });

					// Scroll down
					const loadMore = dialog.querySelector('.mhy-loadmore__btn .mhy-button__button');
					let noMore = false;
					while (!noMore) {
						await scrollToBottom(dialog);
						nomore = dialog.querySelector('.mhy-loadmore__nomore') !== null;
					}
					
					// All other comments
					const cardEls = dialog.querySelectorAll('.s-reply-list__item');
					cardEls.forEach((cardEl) => {
						const a = cardEl.querySelector('.mhy-account-title__name);
						const c = cardEl.querySelector('.reply-card__content-inner-wrapper');
					});
					

					return comments;
				}

				const comments = await Promise.all(
				Array.from(document.querySelectorAll('.reply-card'))
					.map(async (card) => {
						const resComments = [];
						const authorEl = card.querySelector('.mhy-account-title__name');
						const contentsEl = card.querySelectorAll('.replyContentWrapper pre p');
						const author = authorEl ? authorEl.innerText.trim() : '';
						const content = Array.from(contentsEl)
							.map(p => p.innerText.trim())
							.join('\n');

						// Parent
						resComments.push({
							author: author,
							content: content
						});

						const innerRepliesButtonEl = card.querySelector('.reply-card-inner-reply__detail');
						const innerRepliesEls = card.querySelectorAll('.reply-card__replies');

						if (innerRepliesButtonEl) {
							innerRepliesButtonEl.click();
							await sleep(1000);
							const comments = await getCommentsFromDialog();

						} else if (innerRepliesEls.length > 0) {
							innerRepliesEls.forEach((reply) => {
								const innerAuthorEl = reply.querySelector('.mhy-account-title__name');
								const innerContentsEl = reply.querySelectorAll('.reply-card-inner-reply__content p')
								const innerAuthor = innerAuthorEl ? innerAuthorEl.innerText.trim() : '';
								const innerContent = Array.from(innerContentsEl)
									.map(p => p.innerText.trim())
									.join('\n');

								resComments.push({ author: innerAuthor, content: innerContent });
							});
						}

					return resComments;
				});

				return {
					comments: comments.flat()
				};
			})()
			`, &pageData).Do(ctx); err != nil {
				return err
			}
			// slog.Debug("raw page data", "comment_count", len(pageData.Comments), "first", pageData.Comments)

			// Scroll
			chromedp.Evaluate(`document.querySelector('.mhy-loadmore__nomore') !== null`, &noMore).Do(ctx)
			if noMore || nullScrolls >= 3 {
				break
			}
			chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil).Do(ctx)
			time.Sleep(scraperutils.SleepRangeMs(1500, 2500))

			// Check for null scrolls
			prevCount := len(allComments)
			for _, c := range pageData.Comments {
				key := c.Author + "|" + c.Content
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					allComments = append(allComments, c)
				}
			}
			if len(allComments) == prevCount {
				nullScrolls++
			} else {
				nullScrolls = 0
			}

			slog.Debug("scroll iteration",
				"new_comments", len(allComments)-prevCount,
				"total_so_far", len(allComments),
				"null_scrolls", nullScrolls,
			)
		}

		slog.Debug("comments scraped",
			"url", result.Permalink,
			"count", len(allComments),
			"stopped_by", map[bool]string{true: "noMore", false: "nullScrolls"}[noMore],
		)

		for _, c := range allComments {
			comment := ScrapeComment{
				Url:     result.Permalink,
				Author:  c.Author,
				Content: c.Content,
			}
			result.Comments = append(result.Comments, comment)
		}

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
	url := "https://www.hoyolab.com/circles/2/30/feed?page_type=30&page_sort=hot"

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

func (s *svc) persistPost(jobCtx context.Context, res ScrapeData) (repo.HylPost, error) {
	tx, err := s.db.BeginTx(jobCtx, pgx.TxOptions{})
	if err != nil {
		return repo.HylPost{}, err
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
		return repo.HylPost{}, fmt.Errorf("AddHylPost: %w", err)
	}

	channel := fmt.Sprintf("hyl_scrape_%d", res.SessionID)

	payload, err := json.Marshal(NotifyPayload{
		SessionID:   res.SessionID,
		PayloadType: TypePost,
		Post:        &post,
	})
	if err != nil {
		slog.Error("Failed to marshal json for post", "err", err, "postId", post.ID)
		return repo.HylPost{}, err
	}
	_, err = tx.Exec(jobCtx,
		"SELECT pg_notify($1, $2)", channel, string(payload),
	)
	if err != nil {
		slog.Error("pg_notify failed", "err", err)
		return repo.HylPost{}, fmt.Errorf("hylscraper post pg_notify failed: %w", err)
	}

	err = tx.Commit(jobCtx)
	if err != nil {
		slog.Error("failed to commit hylscraper post", "err", err)
		return repo.HylPost{}, fmt.Errorf("hylscraper AddHylPost commit failed: %w", err)
	}

	return post, nil
}

func (s *svc) persistComment(jobCtx context.Context, res ScrapeData, postId int64) ([]repo.HylComment, error) {
	if len(res.Comments) <= 0 {
		return []repo.HylComment{}, nil
	}

	tx, err := s.db.BeginTx(jobCtx, pgx.TxOptions{})
	if err != nil {
		return []repo.HylComment{}, err
	}
	defer tx.Rollback(jobCtx)

	qtx := s.repo.WithTx(tx)

	params := repo.AddHylCommentsParams{
		SessionID: make([]int64, len(res.Comments)),
		PostID:    make([]int64, len(res.Comments)),
		// ParentCommentID: make([]int64, len(res.Comments)), // TODO: its empty
		Url:     make([]string, len(res.Comments)),
		Author:  make([]string, len(res.Comments)),
		Content: make([]string, len(res.Comments)),
	}

	for i, c := range res.Comments {
		params.SessionID[i] = res.SessionID
		params.PostID[i] = postId
		// params.ParentCommentID[i] = c.ParentCommentID // TODO: nothing is here
		params.Url[i] = c.Url
		params.Author[i] = c.Author
		params.Content[i] = c.Content
	}
	comments, err := qtx.AddHylComments(jobCtx, params)
	if err != nil {
		slog.Error("AddHylComment failed", "err", err)
		return []repo.HylComment{}, err
	}

	channel := fmt.Sprintf("hyl_scrape_%d", res.SessionID)

	payload, err := json.Marshal(NotifyPayload{
		SessionID:   res.SessionID,
		PayloadType: TypeComment,
		Comments:    comments,
	})
	if err != nil {
		slog.Error("Failed to marshal json for comment", "err", err, "postId", postId)
		return []repo.HylComment{}, err
	}
	_, err = tx.Exec(jobCtx,
		"SELECT pg_notify($1, $2)", channel, string(payload),
	)
	if err != nil {
		slog.Error("pg_notify failed", "err", err)
		return []repo.HylComment{}, fmt.Errorf("hylscraper comment pg_notify failed: %w", err)
	}

	err = tx.Commit(jobCtx)
	if err != nil {
		slog.Error("failed to commit hylscraper comment", "err", err)
		return []repo.HylComment{}, fmt.Errorf("hylscraper AddHylComment commit failed: %w", err)
	}

	return comments, nil
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
