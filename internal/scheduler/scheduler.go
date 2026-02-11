package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"tumble/internal/archive"
	"tumble/internal/data"
)

const (
	maxRetries    = 3
	retryInterval = 1 * time.Hour
)

// Scheduler manages scheduled tasks for the application.
type Scheduler struct {
	cron          *cron.Cron
	store         data.Store
	archiveClient *archive.Client
	retryCount    int
	retryMu       sync.Mutex
	stopRetry     chan struct{}
}

// New creates a new Scheduler with the given store.
func New(store data.Store) *Scheduler {
	// Use America/Chicago for Central Time
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		slog.Warn("Failed to load America/Chicago timezone, using local time", "error", err)
		loc = time.Local
	}

	return &Scheduler{
		cron:          cron.New(cron.WithLocation(loc)),
		store:         store,
		archiveClient: archive.NewClient(5),
		stopRetry:     make(chan struct{}),
	}
}

// Start begins the scheduler and runs any startup tasks.
func (s *Scheduler) Start(ctx context.Context) error {
	// Schedule daily cat at 10 AM Central
	_, err := s.cron.AddFunc("0 10 * * *", func() {
		s.fetchDailyCatWithRetry(ctx)
	})
	if err != nil {
		return err
	}

	// Schedule archive batch at 3 AM Central
	_, err = s.cron.AddFunc("0 3 * * *", func() {
		runArchiveBatch(ctx, s.store, s.archiveClient)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	slog.Info("Scheduler started", "nextRun", s.cron.Entries()[0].Next)

	// Check if we need to fetch today's kitten on startup
	go s.checkStartupKitten(ctx)

	// Run archive batch on startup (in background)
	go runArchiveBatch(ctx, s.store, s.archiveClient)

	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopRetry)
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("Scheduler stopped")
}

// checkStartupKitten checks if today's kitten needs to be fetched on startup.
// Only fetches if it's past 10 AM Central and no kitten exists for today.
func (s *Scheduler) checkStartupKitten(ctx context.Context) {
	// Check if it's past 10 AM Central Time
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		slog.Warn("Failed to load America/Chicago timezone for startup check", "error", err)
		return
	}

	now := time.Now().In(loc)
	tenAM := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)

	if now.Before(tenAM) {
		slog.Info("Before 10 AM Central, skipping startup kitten check",
			"currentTime", now.Format("15:04:05 MST"))
		return
	}

	existing, err := s.store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		slog.Warn("Failed to check for existing daily cat on startup", "error", err)
		return
	}

	if existing == nil {
		slog.Info("No daily kitten for today and past 10 AM, fetching now")
		s.fetchDailyCatWithRetry(ctx)
	} else {
		slog.Info("Daily kitten already exists", "imageID", existing.ID)
	}
}

// fetchDailyCatWithRetry attempts to fetch the daily cat with retry logic.
func (s *Scheduler) fetchDailyCatWithRetry(ctx context.Context) {
	s.retryMu.Lock()
	s.retryCount = 0
	s.retryMu.Unlock()

	s.attemptFetch(ctx)
}

func (s *Scheduler) attemptFetch(ctx context.Context) {
	stored, err := FetchAndStoreDailyCat(ctx, s.store)
	if err != nil {
		s.retryMu.Lock()
		s.retryCount++
		count := s.retryCount
		s.retryMu.Unlock()

		if count < maxRetries {
			slog.Warn("Failed to fetch daily kitten, will retry",
				"error", err,
				"attempt", count,
				"nextRetry", time.Now().Add(retryInterval))

			// Schedule retry
			go func() {
				select {
				case <-time.After(retryInterval):
					s.attemptFetch(ctx)
				case <-s.stopRetry:
					return
				case <-ctx.Done():
					return
				}
			}()
		} else {
			slog.Warn("Failed to fetch daily kitten after all retries",
				"error", err,
				"attempts", count)
		}
		return
	}

	if stored {
		slog.Info("Daily kitten successfully fetched")
	}
}
