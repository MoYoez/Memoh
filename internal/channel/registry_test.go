package channel

func init() {
	MustRegisterChannel(ChannelDescriptor{
		Type:                ChannelTelegram,
		DisplayName:         "Telegram",
		NormalizeConfig:     NormalizeTelegramConfig,
		NormalizeUserConfig: NormalizeTelegramUserConfig,
	})
	MustRegisterChannel(ChannelDescriptor{
		Type:                ChannelFeishu,
		DisplayName:         "Feishu",
		NormalizeConfig:     NormalizeFeishuConfig,
		NormalizeUserConfig: NormalizeFeishuUserConfig,
	})
	MustRegisterChannel(ChannelDescriptor{
		Type:                ChannelCLI,
		DisplayName:         "CLI",
		NormalizeConfig:     func(map[string]interface{}) (map[string]interface{}, error) { return map[string]interface{}{}, nil },
		NormalizeUserConfig: func(map[string]interface{}) (map[string]interface{}, error) { return map[string]interface{}{}, nil },
	})
	MustRegisterChannel(ChannelDescriptor{
		Type:                ChannelWeb,
		DisplayName:         "Web",
		NormalizeConfig:     func(map[string]interface{}) (map[string]interface{}, error) { return map[string]interface{}{}, nil },
		NormalizeUserConfig: func(map[string]interface{}) (map[string]interface{}, error) { return map[string]interface{}{}, nil },
	})
}
