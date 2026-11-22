package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"tumble/internal/data"
)

// recordChannelName stores the display name a client reported for a
// channel. Failures are logged but never fail the request.
func (h *Handler) recordChannelName(ctx context.Context, clientType, clientNetwork, clientChannel, name *string) {
	if clientType == nil || clientNetwork == nil || clientChannel == nil || name == nil {
		return
	}
	n := strings.TrimPrefix(strings.TrimSpace(*name), "#")
	if *clientType == "" || *clientNetwork == "" || *clientChannel == "" || n == "" {
		return
	}
	err := h.Store.UpsertChannel(ctx, &data.Channel{
		ClientType:    *clientType,
		ClientNetwork: *clientNetwork,
		ClientChannel: *clientChannel,
		Name:          n,
		UpdatedAt:     time.Now(),
	})
	if err != nil {
		slog.Error("Failed to record channel name", "channel", *clientChannel, "error", err)
	}
}

// channelNames loads the channel display name lookup for rendering.
// On error it returns an empty lookup so pages still render.
func (h *Handler) channelNames(ctx context.Context) data.ChannelNames {
	names, err := h.Store.GetChannelNames(ctx)
	if err != nil {
		slog.Error("Failed to load channel names", "error", err)
		return data.ChannelNames{}
	}
	return names
}
