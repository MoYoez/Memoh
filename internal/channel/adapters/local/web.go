package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

type WebAdapter struct {
	hub *channel.SessionHub
}

func NewWebAdapter(hub *channel.SessionHub) *WebAdapter {
	return &WebAdapter{hub: hub}
}

func (a *WebAdapter) Type() channel.ChannelType {
	return channel.ChannelWeb
}

func (a *WebAdapter) Start(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.AdapterRunner, error) {
	return channel.AdapterRunner{SupportsStop: false}, nil
}

func (a *WebAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	if a.hub == nil {
		return fmt.Errorf("web hub not configured")
	}
	target := strings.TrimSpace(msg.To)
	if target == "" {
		return fmt.Errorf("web target is required")
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return fmt.Errorf("message is required")
	}
	a.hub.Publish(target, msg)
	return nil
}
