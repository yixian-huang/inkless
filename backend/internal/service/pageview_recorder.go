package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

const (
	defaultPageViewBuffer  = 256
	defaultPageViewBatch   = 50
	defaultPageViewFlush   = 500 * time.Millisecond
	defaultPageViewWriteTO = 5 * time.Second
)

// PageViewRecorder enqueues page views and persists them asynchronously in batches
// so public read handlers never block on DB inserts.
type PageViewRecorder struct {
	repo       repository.PageViewRepository
	ch         chan *model.PageView
	batchSize  int
	flushEvery time.Duration

	wg     sync.WaitGroup
	stopCh chan struct{}
	once   sync.Once
}

// NewPageViewRecorder creates a buffered async recorder. Call Start before Enqueue.
func NewPageViewRecorder(repo repository.PageViewRepository) *PageViewRecorder {
	return &PageViewRecorder{
		repo:       repo,
		ch:         make(chan *model.PageView, defaultPageViewBuffer),
		batchSize:  defaultPageViewBatch,
		flushEvery: defaultPageViewFlush,
		stopCh:     make(chan struct{}),
	}
}

// Start launches the background flush loop.
func (r *PageViewRecorder) Start() {
	if r == nil || r.repo == nil {
		return
	}
	r.wg.Add(1)
	go r.loop()
}

// Stop drains the queue and waits for the worker to exit (bounded).
func (r *PageViewRecorder) Stop(timeout time.Duration) {
	if r == nil {
		return
	}
	r.once.Do(func() {
		close(r.stopCh)
	})
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	select {
	case <-done:
	case <-time.After(timeout):
		slog.Warn("page view recorder stop timed out")
	}
}

// Track enqueues a page view. Drops the event if the buffer is full (best-effort).
func (r *PageViewRecorder) Track(pageKey, locale, visitorID, referer string) {
	if r == nil || r.repo == nil || pageKey == "" {
		return
	}
	if locale == "" {
		locale = "zh"
	}
	pv := &model.PageView{
		PageKey:   pageKey,
		Locale:    locale,
		VisitorID: visitorID,
		Referer:   referer,
		ViewedAt:  time.Now(),
	}
	select {
	case r.ch <- pv:
	default:
		// Prefer not blocking public responses under load.
		slog.Warn("page view queue full; dropping event", "pageKey", pageKey)
	}
}

func (r *PageViewRecorder) loop() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.flushEvery)
	defer ticker.Stop()

	batch := make([]*model.PageView, 0, r.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toWrite := batch
		batch = make([]*model.PageView, 0, r.batchSize)
		ctx, cancel := context.WithTimeout(context.Background(), defaultPageViewWriteTO)
		defer cancel()
		if err := r.repo.CreateBatch(ctx, toWrite); err != nil {
			// Fallback: try single inserts so one bad row doesn't lose the batch.
			slog.Error("page view batch insert failed; falling back to singles", "count", len(toWrite), "error", err)
			for _, pv := range toWrite {
				if err := r.repo.Create(ctx, pv); err != nil {
					slog.Error("page view insert failed", "pageKey", pv.PageKey, "error", err)
				}
			}
		}
	}

	for {
		select {
		case <-r.stopCh:
			// Drain remaining
			for {
				select {
				case pv := <-r.ch:
					batch = append(batch, pv)
					if len(batch) >= r.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case pv := <-r.ch:
			batch = append(batch, pv)
			if len(batch) >= r.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
