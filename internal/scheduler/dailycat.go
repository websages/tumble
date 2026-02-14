package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"tumble/internal/data"
)

const (
	cataasJSONURL = "https://cataas.com/cat?json=true"
	catAASUser    = "cat AAS"
	kittenTitle   = "Daily Kitten"
)

// catResponse represents the JSON response from cataas.com
type catResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// fetchCatURL fetches a cat from the given JSON API URL and returns a stable image URL.
// Uses the JSON API to get the cat ID, then constructs a permanent URL.
func fetchCatURL(apiURL string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch cat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var cat catResponse
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return "", fmt.Errorf("failed to decode cat response: %w", err)
	}

	// Prefer the URL field if provided, otherwise construct from ID
	if cat.URL != "" {
		return cat.URL, nil
	}

	if cat.ID == "" {
		return "", fmt.Errorf("no cat ID in response")
	}

	return fmt.Sprintf("https://cataas.com/cat/%s", cat.ID), nil
}

// FetchAndStoreDailyCat fetches a cat image and stores it in the database.
// Returns true if a new cat was stored, false if today's cat already exists.
func FetchAndStoreDailyCat(ctx context.Context, store data.Store) (bool, error) {
	// Check if today's cat already exists
	existing, err := store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing cat: %w", err)
	}
	if existing != nil {
		slog.Info("Daily kitten already exists for today", "imageID", existing.ID)
		return false, nil
	}

	// Fetch the cat URL
	catURL, err := fetchCatURL(cataasJSONURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch cat URL: %w", err)
	}

	// Store the image
	id, err := store.InsertImage(ctx, &data.Image{Title: kittenTitle, Link: catAASUser, URL: catURL})
	if err != nil {
		return false, fmt.Errorf("failed to insert cat image: %w", err)
	}

	slog.Info("Daily kitten fetched and stored", "imageID", id, "url", catURL)
	return true, nil
}

// ForceFetchDailyCat deletes today's cat (if any) and fetches a new one.
// Returns the new cat's URL or an error.
func ForceFetchDailyCat(ctx context.Context, store data.Store) (string, error) {
	// Delete today's cat if it exists
	if err := store.DeleteTodayImageByLink(ctx, catAASUser); err != nil {
		return "", fmt.Errorf("failed to delete existing cat: %w", err)
	}

	// Fetch a new cat URL
	catURL, err := fetchCatURL(cataasJSONURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch cat URL: %w", err)
	}

	// Store the image
	id, err := store.InsertImage(ctx, &data.Image{Title: kittenTitle, Link: catAASUser, URL: catURL})
	if err != nil {
		return "", fmt.Errorf("failed to insert cat image: %w", err)
	}

	slog.Info("Daily kitten force-fetched and stored", "imageID", id, "url", catURL)
	return catURL, nil
}
