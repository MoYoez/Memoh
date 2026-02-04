package telegram

import "github.com/memohai/memoh/internal/channel"

func init() {
	channel.MustRegisterChannel(channel.ChannelDescriptor{
		Type:                channel.ChannelTelegram,
		DisplayName:         "Telegram",
		NormalizeConfig:     channel.NormalizeTelegramConfig,
		NormalizeUserConfig: channel.NormalizeTelegramUserConfig,
	})
}
