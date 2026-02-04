package channel

import (
	"context"
	"strings"
)

type InboundMessage struct {
	Channel    ChannelType
	Text       string
	Username   string
	UserID     string
	OpenID     string
	ChatID     string
	ChatType   string
	ReplyTo    string
	BotID      string // 增加 BotID 以支持多 Bot 隔离
	SessionKey string
}

// SessionID 结构: platform:bot_id:chat_id[:sender_id]
func (m InboundMessage) SessionID() string {
	if strings.TrimSpace(m.SessionKey) != "" {
		return strings.TrimSpace(m.SessionKey)
	}
	return GenerateSessionID(string(m.Channel), m.BotID, m.ChatID, m.ChatType, m.OpenID, m.UserID, m.Username)
}

// GenerateSessionID 统一生成 SessionID 的逻辑
func GenerateSessionID(platform, botID, chatID, chatType, openID, userID, username string) string {
	parts := []string{platform, botID, chatID}
	// 如果是群聊，增加发送者 ID 以支持个人上下文
	ct := strings.ToLower(strings.TrimSpace(chatType))
	if ct != "" && ct != "p2p" && ct != "private" {
		senderID := strings.TrimSpace(openID)
		if senderID == "" {
			senderID = strings.TrimSpace(userID)
		}
		if senderID == "" {
			senderID = strings.TrimSpace(username)
		}
		if senderID != "" {
			parts = append(parts, senderID)
		}
	}
	return strings.Join(parts, ":")
}

type OutboundMessage struct {
	To   string
	Text string
}

type AdapterRunner struct {
	Stop         func()
	SupportsStop bool
}

type InboundHandler func(ctx context.Context, cfg ChannelConfig, msg InboundMessage) error

type Adapter interface {
	Type() ChannelType
	Start(ctx context.Context, cfg ChannelConfig, handler InboundHandler) (AdapterRunner, error)
	Send(ctx context.Context, cfg ChannelConfig, msg OutboundMessage) error
}
