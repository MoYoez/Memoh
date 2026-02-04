package channel

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TelegramConfig struct {
	BotToken string
}

type TelegramUserConfig struct {
	Username string
	UserID   string
	ChatID   string
}

type FeishuConfig struct {
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
}

type FeishuUserConfig struct {
	OpenID string
	UserID string
}

func NormalizeChannelConfig(channelType ChannelType, raw map[string]interface{}) (map[string]interface{}, error) {
	if raw == nil {
		raw = map[string]interface{}{}
	}
	desc, ok := GetChannelDescriptor(channelType)
	if !ok {
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
	if desc.NormalizeConfig == nil {
		return raw, nil
	}
	return desc.NormalizeConfig(raw)
}

func NormalizeChannelUserConfig(channelType ChannelType, raw map[string]interface{}) (map[string]interface{}, error) {
	if raw == nil {
		raw = map[string]interface{}{}
	}
	desc, ok := GetChannelDescriptor(channelType)
	if !ok {
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
	if desc.NormalizeUserConfig == nil {
		return raw, nil
	}
	return desc.NormalizeUserConfig(raw)
}

func NormalizeTelegramConfig(raw map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := parseTelegramConfig(raw)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"botToken": cfg.BotToken,
	}, nil
}

func NormalizeTelegramUserConfig(raw map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := parseTelegramUserConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	if cfg.Username != "" {
		result["username"] = cfg.Username
	}
	if cfg.UserID != "" {
		result["user_id"] = cfg.UserID
	}
	if cfg.ChatID != "" {
		result["chat_id"] = cfg.ChatID
	}
	return result, nil
}

func NormalizeFeishuConfig(raw map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := parseFeishuConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"appId":     cfg.AppID,
		"appSecret": cfg.AppSecret,
	}
	if cfg.EncryptKey != "" {
		result["encryptKey"] = cfg.EncryptKey
	}
	if cfg.VerificationToken != "" {
		result["verificationToken"] = cfg.VerificationToken
	}
	return result, nil
}

func NormalizeFeishuUserConfig(raw map[string]interface{}) (map[string]interface{}, error) {
	cfg, err := parseFeishuUserConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	if cfg.OpenID != "" {
		result["open_id"] = cfg.OpenID
	}
	if cfg.UserID != "" {
		result["user_id"] = cfg.UserID
	}
	return result, nil
}

func DecodeTelegramConfig(raw []byte) (TelegramConfig, error) {
	payload, err := decodeConfigMap(raw)
	if err != nil {
		return TelegramConfig{}, err
	}
	return parseTelegramConfig(payload)
}

func DecodeTelegramUserConfig(raw []byte) (TelegramUserConfig, error) {
	payload, err := decodeConfigMap(raw)
	if err != nil {
		return TelegramUserConfig{}, err
	}
	return parseTelegramUserConfig(payload)
}

func DecodeFeishuConfig(raw []byte) (FeishuConfig, error) {
	payload, err := decodeConfigMap(raw)
	if err != nil {
		return FeishuConfig{}, err
	}
	return parseFeishuConfig(payload)
}

func DecodeFeishuUserConfig(raw []byte) (FeishuUserConfig, error) {
	payload, err := decodeConfigMap(raw)
	if err != nil {
		return FeishuUserConfig{}, err
	}
	return parseFeishuUserConfig(payload)
}

func parseTelegramConfig(raw map[string]interface{}) (TelegramConfig, error) {
	token := readString(raw, "botToken", "bot_token")
	token = strings.TrimSpace(token)
	if token == "" {
		return TelegramConfig{}, fmt.Errorf("telegram botToken is required")
	}
	return TelegramConfig{BotToken: token}, nil
}

func parseTelegramUserConfig(raw map[string]interface{}) (TelegramUserConfig, error) {
	username := strings.TrimSpace(readString(raw, "username"))
	userID := strings.TrimSpace(readString(raw, "userId", "user_id"))
	chatID := strings.TrimSpace(readString(raw, "chatId", "chat_id"))
	if username == "" && userID == "" && chatID == "" {
		return TelegramUserConfig{}, fmt.Errorf("telegram user config requires username, user_id, or chat_id")
	}
	return TelegramUserConfig{
		Username: username,
		UserID:   userID,
		ChatID:   chatID,
	}, nil
}

func parseFeishuConfig(raw map[string]interface{}) (FeishuConfig, error) {
	appID := strings.TrimSpace(readString(raw, "appId", "app_id"))
	appSecret := strings.TrimSpace(readString(raw, "appSecret", "app_secret"))
	encryptKey := strings.TrimSpace(readString(raw, "encryptKey", "encrypt_key"))
	verificationToken := strings.TrimSpace(readString(raw, "verificationToken", "verification_token"))
	if appID == "" || appSecret == "" {
		return FeishuConfig{}, fmt.Errorf("feishu appId and appSecret are required")
	}
	return FeishuConfig{
		AppID:             appID,
		AppSecret:         appSecret,
		EncryptKey:        encryptKey,
		VerificationToken: verificationToken,
	}, nil
}

func parseFeishuUserConfig(raw map[string]interface{}) (FeishuUserConfig, error) {
	openID := strings.TrimSpace(readString(raw, "openId", "open_id"))
	userID := strings.TrimSpace(readString(raw, "userId", "user_id"))
	if openID == "" && userID == "" {
		return FeishuUserConfig{}, fmt.Errorf("feishu user config requires open_id or user_id")
	}
	return FeishuUserConfig{OpenID: openID, UserID: userID}, nil
}

func decodeConfigMap(raw []byte) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	return payload, nil
}

func readString(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			switch v := value.(type) {
			case string:
				return v
			default:
				encoded, err := json.Marshal(v)
				if err == nil {
					return strings.Trim(string(encoded), "\"")
				}
			}
		}
	}
	return ""
}
