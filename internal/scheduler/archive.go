package scheduler

import (
	"context"
	"log/slog"
	"time"

	"tumble/internal/archive"
	"tumble/internal/data"
)

const (
	archiveRecheckNotFound = 30 * 24 * time.Hour
	archiveRecheckError    = 24 * time.Hour
)

func runArchiveBatch(ctx context.Context, store data.Store, client *archive.Client) {
	// 1. Get unchecked dead link URLs
	unchecked, err := store.GetUncheckedDeadLinkURLs(ctx)
	if err != nil {
		slog.Error("Archive batch: failed to get unchecked URLs", "error", err)
		return
	}

	// 2. Get stale not_found/error URLs needing recheck
	staleNotFound, err := store.GetStaleArchiveLookups(ctx, "not_found", archiveRecheckNotFound)
	if err != nil {
		slog.Error("Archive batch: failed to get stale not_found URLs", "error", err)
	}
	staleError, err := store.GetStaleArchiveLookups(ctx, "error", archiveRecheckError)
	if err != nil {
		slog.Error("Archive batch: failed to get stale error URLs", "error", err)
	}

	// Combine and deduplicate
	seen := make(map[string]bool)
	var urls []string
	for _, u := range unchecked {
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	for _, u := range staleNotFound {
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	for _, u := range staleError {
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}

	if len(urls) == 0 {
		slog.Info("Archive batch: no URLs to check")
		return
	}

	slog.Info("Archive batch: starting", "total", len(urls))

	for i, u := range urls {
		if ctx.Err() != nil {
			slog.Info("Archive batch: context cancelled, stopping", "checked", i)
			return
		}

		result, err := client.Check(ctx, u)
		if err != nil {
			slog.Warn("Archive batch: API error", "url", u, "error", err)
			store.UpsertArchiveLookup(ctx, &data.ArchiveLookup{
				URL:       u,
				Status:    "error",
				CheckedAt: time.Now(),
			})
			continue
		}

		lookup := &data.ArchiveLookup{
			URL:       u,
			CheckedAt: time.Now(),
		}
		if result.Found {
			lookup.Status = "found"
			lookup.ArchiveURL = &result.ArchiveURL
			if !result.SnapshotAt.IsZero() {
				lookup.SnapshotAt = &result.SnapshotAt
			}
		} else {
			lookup.Status = "not_found"
		}

		if err := store.UpsertArchiveLookup(ctx, lookup); err != nil {
			slog.Warn("Archive batch: failed to store result", "url", u, "error", err)
		}

		if (i+1)%50 == 0 {
			slog.Info("Archive batch: progress", "checked", i+1, "total", len(urls))
		}
	}

	slog.Info("Archive batch: complete", "total", len(urls))
}
