package local

import "github.com/memohai/memoh/internal/channel"

func init() {
	channel.MustRegisterChannel(channel.ChannelDescriptor{
		Type:                channel.ChannelCLI,
		DisplayName:         "CLI",
		NormalizeConfig:     normalizeEmpty,
		NormalizeUserConfig: normalizeEmpty,
	})
	channel.MustRegisterChannel(channel.ChannelDescriptor{
		Type:                channel.ChannelWeb,
		DisplayName:         "Web",
		NormalizeConfig:     normalizeEmpty,
		NormalizeUserConfig: normalizeEmpty,
	})
}

func normalizeEmpty(map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
