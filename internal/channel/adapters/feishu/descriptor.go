package feishu

import "github.com/memohai/memoh/internal/channel"

func init() {
	channel.MustRegisterChannel(channel.ChannelDescriptor{
		Type:                channel.ChannelFeishu,
		DisplayName:         "Feishu",
		NormalizeConfig:     channel.NormalizeFeishuConfig,
		NormalizeUserConfig: channel.NormalizeFeishuUserConfig,
	})
}
