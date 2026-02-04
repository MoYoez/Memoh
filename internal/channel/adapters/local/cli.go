package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

type CLIAdapter struct {
	hub *channel.SessionHub
}

func NewCLIAdapter(hub *channel.SessionHub) *CLIAdapter {
	return &CLIAdapter{hub: hub}
}

func (a *CLIAdapter) Type() channel.ChannelType {
	return channel.ChannelCLI
}

func (a *CLIAdapter) Start(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.AdapterRunner, error) {
	return channel.AdapterRunner{SupportsStop: false}, nil
}

func (a *CLIAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	if a.hub == nil {
		return fmt.Errorf("cli hub not configured")
	}
	target := strings.TrimSpace(msg.To)
	if target == "" {
		return fmt.Errorf("cli target is required")
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return fmt.Errorf("message is required")
	}
	a.hub.Publish(target, msg)
	return nil
}
