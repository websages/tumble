package activitypub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"tumble/internal/data"
)

const maxDeliveryAttempts = 8

// PublishNote fans out a Create(Note) activity to every follower inbox
// (deduped by shared inbox) as queued deliveries.
func (s *Service) PublishNote(ctx context.Context, note *Note) {
	if !s.Enabled() {
		return
	}

	inboxes, err := s.Store.ListActivityPubFollowerInboxes(ctx)
	if err != nil {
		slog.Error("ActivityPub: failed to list follower inboxes", "error", err)
		return
	}
	if len(inboxes) == 0 {
		return
	}

	activity := s.WrapCreate(note)
	payload, err := json.Marshal(activity)
	if err != nil {
		slog.Error("ActivityPub: failed to marshal Create activity", "error", err)
		return
	}

	for _, inbox := range inboxes {
		if err := s.Store.EnqueueActivityPubDelivery(ctx, inbox, string(payload)); err != nil {
			slog.Error("ActivityPub: failed to enqueue delivery", "inbox", inbox, "error", err)
		}
	}
}

// deliverActivity signs and POSTs an arbitrary activity payload to a single
// remote inbox immediately (used for Accept replies to Follow requests).
func (s *Service) deliverActivity(ctx context.Context, inboxURL string, activity any) error {
	payload, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	return s.postSigned(ctx, inboxURL, payload)
}

func (s *Service) postSigned(ctx context.Context, inboxURL string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inboxURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", `application/activity+json`)

	if err := s.signRequest(ctx, req, body); err != nil {
		return fmt.Errorf("sign request: %w", err)
	}

	client := newSafeClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("inbox %s returned status %d", inboxURL, resp.StatusCode)
	}
	return nil
}

// ProcessDueDeliveries sends any pending deliveries whose retry time has
// arrived, marking each as sent or scheduling a backed-off retry on failure.
func (s *Service) ProcessDueDeliveries(ctx context.Context) {
	if !s.Enabled() {
		return
	}

	deliveries, err := s.Store.GetDueActivityPubDeliveries(ctx, 25)
	if err != nil {
		slog.Error("ActivityPub: failed to load due deliveries", "error", err)
		return
	}

	for _, d := range deliveries {
		if err := s.postSigned(ctx, d.InboxURL, []byte(d.Payload)); err != nil {
			s.markFailed(ctx, d, err)
			continue
		}
		if err := s.Store.MarkActivityPubDeliverySucceeded(ctx, d.ID); err != nil {
			slog.Error("ActivityPub: failed to mark delivery succeeded", "id", d.ID, "error", err)
		}
	}
}

func (s *Service) markFailed(ctx context.Context, d data.ActivityPubDelivery, deliveryErr error) {
	attempts := d.Attempts + 1
	giveUp := attempts >= maxDeliveryAttempts
	backoff := time.Duration(math.Pow(2, float64(attempts))) * time.Minute
	if backoff > 24*time.Hour {
		backoff = 24 * time.Hour
	}

	slog.Warn("ActivityPub: delivery failed", "id", d.ID, "inbox", d.InboxURL, "attempts", attempts, "give_up", giveUp, "error", deliveryErr)

	if err := s.Store.MarkActivityPubDeliveryFailed(ctx, d.ID, deliveryErr.Error(), time.Now().Add(backoff), giveUp); err != nil {
		slog.Error("ActivityPub: failed to record delivery failure", "id", d.ID, "error", err)
	}
}
