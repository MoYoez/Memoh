package router

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/chat"
	"github.com/memohai/memoh/internal/contacts"
	"github.com/memohai/memoh/internal/settings"
)

// ChatGateway 抽象聊天能力，避免路由层直接依赖具体实现。
type ChatGateway interface {
	Chat(ctx context.Context, req chat.ChatRequest) (chat.ChatResponse, error)
}

type ContactService interface {
	GetByID(ctx context.Context, contactID string) (contacts.Contact, error)
	GetByUserID(ctx context.Context, botID, userID string) (contacts.Contact, error)
	GetByChannelIdentity(ctx context.Context, botID, platform, externalID string) (contacts.ContactChannel, error)
	Create(ctx context.Context, req contacts.CreateRequest) (contacts.Contact, error)
	CreateGuest(ctx context.Context, botID, displayName string) (contacts.Contact, error)
	UpsertChannel(ctx context.Context, botID, contactID, platform, externalID string, metadata map[string]interface{}) (contacts.ContactChannel, error)
	GetBindToken(ctx context.Context, token string) (contacts.BindToken, error)
	MarkBindTokenUsed(ctx context.Context, id string) (contacts.BindToken, error)
	BindUser(ctx context.Context, contactID, userID string) (contacts.Contact, error)
}

type SettingsService interface {
	GetBot(ctx context.Context, botID string) (settings.Settings, error)
}

// ChannelInboundProcessor 将 channel 入站消息路由到 chat，并返回可发送的回复。
type ChannelInboundProcessor struct {
	store            channel.ConfigStore
	chat             ChatGateway
	contacts         ContactService
	settings         SettingsService
	logger           *slog.Logger
	unboundReply     string
	bindSuccessReply string
	jwtSecret        string
	tokenTTL         time.Duration
}

func NewChannelInboundProcessor(log *slog.Logger, store channel.ConfigStore, chatGateway ChatGateway, contactService ContactService, settingsService SettingsService, jwtSecret string, tokenTTL time.Duration) *ChannelInboundProcessor {
	if log == nil {
		log = slog.Default()
	}
	if tokenTTL <= 0 {
		tokenTTL = 5 * time.Minute
	}
	return &ChannelInboundProcessor{
		store:            store,
		chat:             chatGateway,
		contacts:         contactService,
		settings:         settingsService,
		logger:           log.With(slog.String("component", "channel_router")),
		unboundReply:     "当前不允许陌生人访问，请联系管理员。",
		bindSuccessReply: "绑定成功，感谢确认。",
		jwtSecret:        strings.TrimSpace(jwtSecret),
		tokenTTL:         tokenTTL,
	}
}

func (p *ChannelInboundProcessor) HandleInbound(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage) (*channel.OutboundMessage, error) {
	if p.store == nil || p.chat == nil || p.contacts == nil {
		return nil, fmt.Errorf("channel inbound processor not configured")
	}
	if strings.TrimSpace(msg.Text) == "" {
		return nil, nil
	}
	if strings.TrimSpace(msg.BotID) == "" {
		msg.BotID = cfg.BotID
	}

	sessionID := msg.SessionID()
	channelConfigID := cfg.ID
	if msg.Channel == channel.ChannelCLI || msg.Channel == channel.ChannelWeb {
		channelConfigID = ""
	}

	session, err := p.store.GetChannelSession(ctx, sessionID)
	if err != nil && p.logger != nil {
		p.logger.Error("get user by session failed", slog.String("session_id", sessionID), slog.Any("error", err))
	}
	userID := strings.TrimSpace(session.UserID)
	contactID := strings.TrimSpace(session.ContactID)
	externalID := extractExternalIdentity(msg)

	if bindReply, handled := p.tryHandleBindToken(ctx, msg, externalID); handled {
		return bindReply, nil
	}

	if userID == "" {
		userID, err = p.store.ResolveUserBinding(ctx, msg.Channel, channel.BindingCriteria{
			Username: msg.Username,
			UserID:   msg.UserID,
			ChatID:   msg.ChatID,
			OpenID:   msg.OpenID,
		})
		if err == nil && userID != "" {
			_ = p.store.UpsertChannelSession(ctx, sessionID, msg.BotID, channelConfigID, userID, contactID, string(msg.Channel))
		}
	}

	var contact contacts.Contact
	if contactID == "" && userID != "" {
		contact, err = p.contacts.GetByUserID(ctx, msg.BotID, userID)
		if err != nil {
			displayName := extractDisplayName(msg)
			contact, err = p.contacts.Create(ctx, contacts.CreateRequest{
				BotID:       msg.BotID,
				UserID:      userID,
				DisplayName: displayName,
				Status:      "active",
			})
		}
		if err == nil {
			contactID = contact.ID
			if externalID != "" {
				_, _ = p.contacts.UpsertChannel(ctx, msg.BotID, contactID, msg.Channel.String(), externalID, nil)
			}
		}
	}

	if contactID == "" && externalID != "" {
		binding, err := p.contacts.GetByChannelIdentity(ctx, msg.BotID, msg.Channel.String(), externalID)
		if err == nil {
			contactID = binding.ContactID
		}
	}

	if contactID == "" {
		allowGuest := false
		if p.settings != nil {
			botSettings, err := p.settings.GetBot(ctx, msg.BotID)
			if err == nil {
				allowGuest = botSettings.AllowGuest
			}
		}
		if allowGuest {
			displayName := extractDisplayName(msg)
			contact, err = p.contacts.CreateGuest(ctx, msg.BotID, displayName)
			if err == nil {
				contactID = contact.ID
				if externalID != "" {
					_, _ = p.contacts.UpsertChannel(ctx, msg.BotID, contactID, msg.Channel.String(), externalID, nil)
				}
			}
		} else {
			return p.buildUnboundReply(msg)
		}
	}

	if contactID != "" && contact.ID == "" {
		loaded, err := p.contacts.GetByID(ctx, contactID)
		if err == nil {
			contact = loaded
		}
	}

	if contactID != "" {
		_ = p.store.UpsertChannelSession(ctx, sessionID, msg.BotID, channelConfigID, userID, contactID, string(msg.Channel))
	}

	sessionToken := ""
	if p.jwtSecret != "" && strings.TrimSpace(msg.ReplyTo) != "" {
		signed, _, err := auth.GenerateSessionToken(auth.SessionToken{
			BotID:       msg.BotID,
			Platform:    msg.Channel.String(),
			ReplyTarget: strings.TrimSpace(msg.ReplyTo),
			SessionID:   sessionID,
			ContactID:   contactID,
		}, p.jwtSecret, p.tokenTTL)
		if err != nil {
			if p.logger != nil {
				p.logger.Warn("issue session token failed", slog.Any("error", err))
			}
		} else {
			sessionToken = signed
		}
	}

	token := ""
	if userID != "" && p.jwtSecret != "" {
		signed, _, err := auth.GenerateToken(userID, p.jwtSecret, p.tokenTTL)
		if err != nil {
			if p.logger != nil {
				p.logger.Warn("issue channel token failed", slog.Any("error", err))
			}
		} else {
			token = "Bearer " + signed
		}
	}
	resp, err := p.chat.Chat(ctx, chat.ChatRequest{
		BotID:           msg.BotID,
		SessionID:       sessionID,
		Token:           token,
		UserID:          userID,
		Query:           msg.Text,
		CurrentPlatform: msg.Channel.String(),
		Platforms:       []string{msg.Channel.String()},
		ToolContext: &chat.ToolContext{
			BotID:           msg.BotID,
			SessionID:       sessionID,
			CurrentPlatform: msg.Channel.String(),
			ReplyTarget:     strings.TrimSpace(msg.ReplyTo),
			SessionToken:    sessionToken,
			ContactID:       contactID,
			ContactAlias:    strings.TrimSpace(contact.Alias),
			ContactName:     strings.TrimSpace(contact.DisplayName),
		},
	})
	if err != nil {
		if p.logger != nil {
			p.logger.Error("chat gateway failed", slog.String("channel", msg.Channel.String()), slog.String("user_id", userID), slog.Any("error", err))
		}
		return nil, err
	}
	if len(resp.Messages) == 0 {
		return nil, nil
	}
	// Extract assistant text as reply
	if reply := extractAssistantReply(resp.Messages); strings.TrimSpace(reply) != "" {
		target := strings.TrimSpace(msg.ReplyTo)
		if target == "" {
			return nil, fmt.Errorf("reply target missing")
		}
		return &channel.OutboundMessage{
			To:   target,
			Text: reply,
		}, nil
	}
	return nil, nil
}

// extractAssistantReply extracts text content from the last assistant message with actual text.
// Skips assistant messages that only contain tool_calls without text content.
func extractAssistantReply(messages []chat.GatewayMessage) string {
	if len(messages) == 0 {
		return ""
	}
	reply := ""
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		if role != "" && role != "assistant" {
			continue
		}
		// Skip if this message only has tool_calls without text content
		if _, hasToolCalls := msg["tool_calls"]; hasToolCalls {
			// Check if there's also text content
			if msg["content"] == nil {
				continue
			}
		}
		if content, ok := msg["content"].(string); ok && strings.TrimSpace(content) != "" {
			reply = content
			continue
		}
		parts, ok := msg["content"].([]interface{})
		if !ok {
			continue
		}
		texts := make([]string, 0, len(parts))
		for _, part := range parts {
			switch value := part.(type) {
			case string:
				if strings.TrimSpace(value) != "" {
					texts = append(texts, value)
				}
			case map[string]interface{}:
				if text, ok := value["text"].(string); ok && strings.TrimSpace(text) != "" {
					texts = append(texts, text)
				}
			}
		}
		if len(texts) > 0 {
			reply = strings.Join(texts, "\n")
		}
	}
	return reply
}

func (p *ChannelInboundProcessor) buildUnboundReply(msg channel.InboundMessage) (*channel.OutboundMessage, error) {
	target := strings.TrimSpace(msg.ReplyTo)
	if target == "" {
		return nil, fmt.Errorf("reply target missing")
	}
	return &channel.OutboundMessage{
		To:   target,
		Text: p.unboundReply,
	}, nil
}

func extractExternalIdentity(msg channel.InboundMessage) string {
	if strings.TrimSpace(msg.OpenID) != "" {
		return strings.TrimSpace(msg.OpenID)
	}
	if strings.TrimSpace(msg.UserID) != "" {
		return strings.TrimSpace(msg.UserID)
	}
	if strings.TrimSpace(msg.Username) != "" {
		return strings.TrimSpace(msg.Username)
	}
	if strings.TrimSpace(msg.ChatID) != "" {
		return strings.TrimSpace(msg.ChatID)
	}
	return ""
}

func extractDisplayName(msg channel.InboundMessage) string {
	if strings.TrimSpace(msg.Username) != "" {
		return strings.TrimSpace(msg.Username)
	}
	if strings.TrimSpace(msg.UserID) != "" {
		return strings.TrimSpace(msg.UserID)
	}
	if strings.TrimSpace(msg.OpenID) != "" {
		return strings.TrimSpace(msg.OpenID)
	}
	if strings.TrimSpace(msg.ChatID) != "" {
		return strings.TrimSpace(msg.ChatID)
	}
	return ""
}

func buildUserBindingConfig(msg channel.InboundMessage) map[string]interface{} {
	config := map[string]interface{}{}
	switch msg.Channel {
	case channel.ChannelFeishu:
		if strings.TrimSpace(msg.OpenID) != "" {
			config["open_id"] = strings.TrimSpace(msg.OpenID)
		}
		if strings.TrimSpace(msg.UserID) != "" {
			config["user_id"] = strings.TrimSpace(msg.UserID)
		}
	case channel.ChannelTelegram:
		if strings.TrimSpace(msg.Username) != "" {
			config["username"] = strings.TrimSpace(msg.Username)
		}
		if strings.TrimSpace(msg.UserID) != "" {
			config["user_id"] = strings.TrimSpace(msg.UserID)
		}
		if strings.TrimSpace(msg.ChatID) != "" {
			config["chat_id"] = strings.TrimSpace(msg.ChatID)
		}
	}
	return config
}

func (p *ChannelInboundProcessor) tryHandleBindToken(ctx context.Context, msg channel.InboundMessage, externalID string) (*channel.OutboundMessage, bool) {
	tokenText := strings.TrimSpace(msg.Text)
	if tokenText == "" {
		return nil, false
	}
	token, err := p.contacts.GetBindToken(ctx, tokenText)
	if err != nil {
		return nil, false
	}
	replyTarget := strings.TrimSpace(msg.ReplyTo)
	if replyTarget == "" {
		return nil, true
	}
	now := time.Now().UTC()
	if !token.UsedAt.IsZero() {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定码已被使用。"}, true
	}
	if now.After(token.ExpiresAt) {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定码已过期，请重新获取。"}, true
	}
	if token.BotID != msg.BotID {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定码不匹配。"}, true
	}
	if token.TargetPlatform != "" && token.TargetPlatform != msg.Channel.String() {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定码平台不匹配。"}, true
	}
	if token.TargetExternalID != "" && token.TargetExternalID != externalID {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定码目标不匹配。"}, true
	}
	if externalID == "" {
		return &channel.OutboundMessage{To: replyTarget, Text: "无法识别当前账号，绑定失败。"}, true
	}
	if _, err := p.contacts.UpsertChannel(ctx, msg.BotID, token.ContactID, msg.Channel.String(), externalID, nil); err != nil {
		return &channel.OutboundMessage{To: replyTarget, Text: "绑定失败，请稍后重试。"}, true
	}
	if strings.TrimSpace(token.IssuedByUserID) != "" {
		if boundContact, err := p.contacts.GetByID(ctx, token.ContactID); err == nil {
			if strings.TrimSpace(boundContact.UserID) != "" && boundContact.UserID != token.IssuedByUserID {
				return &channel.OutboundMessage{To: replyTarget, Text: "该绑定码已关联其他账号。"}, true
			}
		}
		_, _ = p.contacts.BindUser(ctx, token.ContactID, token.IssuedByUserID)
		if config := buildUserBindingConfig(msg); len(config) > 0 {
			_, _ = p.store.UpsertUserConfig(ctx, token.IssuedByUserID, msg.Channel, channel.UpsertUserConfigRequest{
				Config: config,
			})
		}
		_ = p.store.UpsertChannelSession(ctx, msg.SessionID(), msg.BotID, "", token.IssuedByUserID, token.ContactID, msg.Channel.String())
	}
	_, _ = p.contacts.MarkBindTokenUsed(ctx, token.ID)
	return &channel.OutboundMessage{To: replyTarget, Text: p.bindSuccessReply}, true
}
