package router

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/chat"
	"github.com/memohai/memoh/internal/contacts"
)

type fakeConfigStore struct {
	session     channel.ChannelSession
	boundUserID string
}

func (f *fakeConfigStore) ResolveEffectiveConfig(ctx context.Context, botID string, channelType channel.ChannelType) (channel.ChannelConfig, error) {
	return channel.ChannelConfig{}, nil
}

func (f *fakeConfigStore) GetUserConfig(ctx context.Context, actorUserID string, channelType channel.ChannelType) (channel.ChannelUserBinding, error) {
	return channel.ChannelUserBinding{}, fmt.Errorf("not implemented")
}

func (f *fakeConfigStore) UpsertUserConfig(ctx context.Context, actorUserID string, channelType channel.ChannelType, req channel.UpsertUserConfigRequest) (channel.ChannelUserBinding, error) {
	return channel.ChannelUserBinding{}, nil
}

func (f *fakeConfigStore) ListConfigsByType(ctx context.Context, channelType channel.ChannelType) ([]channel.ChannelConfig, error) {
	return nil, nil
}

func (f *fakeConfigStore) ResolveUserBinding(ctx context.Context, channelType channel.ChannelType, criteria channel.BindingCriteria) (string, error) {
	if f.boundUserID == "" {
		return "", fmt.Errorf("channel user binding not found")
	}
	return f.boundUserID, nil
}

func (f *fakeConfigStore) GetChannelSession(ctx context.Context, sessionID string) (channel.ChannelSession, error) {
	if f.session.SessionID == sessionID {
		return f.session, nil
	}
	return channel.ChannelSession{}, nil
}

func (f *fakeConfigStore) UpsertChannelSession(ctx context.Context, sessionID string, botID string, channelConfigID string, userID string, contactID string, platform string) error {
	return nil
}

type fakeChatGateway struct {
	resp   chat.ChatResponse
	err    error
	gotReq chat.ChatRequest
}

func (f *fakeChatGateway) Chat(ctx context.Context, req chat.ChatRequest) (chat.ChatResponse, error) {
	f.gotReq = req
	return f.resp, f.err
}

type fakeContactService struct {
	contactID string
}

func (f *fakeContactService) GetByID(ctx context.Context, contactID string) (contacts.Contact, error) {
	return contacts.Contact{}, fmt.Errorf("not found")
}

func (f *fakeContactService) GetByUserID(ctx context.Context, botID, userID string) (contacts.Contact, error) {
	return contacts.Contact{}, fmt.Errorf("not found")
}

func (f *fakeContactService) GetByChannelIdentity(ctx context.Context, botID, platform, externalID string) (contacts.ContactChannel, error) {
	return contacts.ContactChannel{}, fmt.Errorf("not found")
}

func (f *fakeContactService) Create(ctx context.Context, req contacts.CreateRequest) (contacts.Contact, error) {
	return contacts.Contact{ID: "contact-1", BotID: req.BotID, UserID: req.UserID}, nil
}

func (f *fakeContactService) CreateGuest(ctx context.Context, botID, displayName string) (contacts.Contact, error) {
	return contacts.Contact{ID: "contact-guest", BotID: botID}, nil
}

func (f *fakeContactService) UpsertChannel(ctx context.Context, botID, contactID, platform, externalID string, metadata map[string]interface{}) (contacts.ContactChannel, error) {
	return contacts.ContactChannel{ID: "channel-1", ContactID: contactID}, nil
}

func (f *fakeContactService) GetBindToken(ctx context.Context, token string) (contacts.BindToken, error) {
	return contacts.BindToken{}, fmt.Errorf("not found")
}

func (f *fakeContactService) MarkBindTokenUsed(ctx context.Context, id string) (contacts.BindToken, error) {
	return contacts.BindToken{}, nil
}

func (f *fakeContactService) BindUser(ctx context.Context, contactID, userID string) (contacts.Contact, error) {
	return contacts.Contact{}, nil
}

func TestChannelInboundProcessorBoundUser(t *testing.T) {
	store := &fakeConfigStore{
		session: channel.ChannelSession{
			SessionID: "feishu:bot-1:chat-1",
			UserID:    "user-123",
		},
	}
	gateway := &fakeChatGateway{
		resp: chat.ChatResponse{
			Messages: []chat.GatewayMessage{
				{"role": "assistant", "content": "AI回复内容"},
			},
		},
	}
	processor := NewChannelInboundProcessor(slog.Default(), store, gateway, &fakeContactService{}, nil, "", 0)

	cfg := channel.ChannelConfig{ID: "cfg-1", BotID: "bot-1", ChannelType: channel.ChannelFeishu}
	msg := channel.InboundMessage{
		Channel: channel.ChannelFeishu,
		Text:    "你好",
		ChatID:  "chat-1",
		ReplyTo: "target-id",
	}

	out, err := processor.HandleInbound(context.Background(), cfg, msg)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if gateway.gotReq.Query != "你好" {
		t.Errorf("Chat 请求 Query 错误: %s", gateway.gotReq.Query)
	}
	if gateway.gotReq.SessionID != "feishu:bot-1:chat-1" {
		t.Errorf("SessionID 传递错误: %s", gateway.gotReq.SessionID)
	}
	if out != nil {
		t.Fatalf("不应直接返回回复: %+v", out)
	}
}

func TestChannelInboundProcessorUnboundUser(t *testing.T) {
	store := &fakeConfigStore{}
	gateway := &fakeChatGateway{}
	processor := NewChannelInboundProcessor(slog.Default(), store, gateway, &fakeContactService{}, nil, "", 0)

	cfg := channel.ChannelConfig{ID: "cfg-1", BotID: "bot-1", ChannelType: channel.ChannelFeishu}
	msg := channel.InboundMessage{
		Channel: channel.ChannelFeishu,
		Text:    "你好",
		ReplyTo: "target-id",
	}

	out, err := processor.HandleInbound(context.Background(), cfg, msg)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if out == nil || !strings.Contains(out.Text, "尚未绑定") {
		t.Fatalf("应返回绑定提示，实际返回: %+v", out)
	}
	if gateway.gotReq.Query != "" {
		t.Error("未绑定用户不应触发 Chat 调用")
	}
}

func TestChannelInboundProcessorIgnoreEmpty(t *testing.T) {
	store := &fakeConfigStore{}
	gateway := &fakeChatGateway{}
	processor := NewChannelInboundProcessor(slog.Default(), store, gateway, &fakeContactService{}, nil, "", 0)

	cfg := channel.ChannelConfig{ID: "cfg-1"}
	msg := channel.InboundMessage{Text: "  "}

	out, err := processor.HandleInbound(context.Background(), cfg, msg)
	if err != nil {
		t.Fatalf("空消息不应报错: %v", err)
	}
	if out != nil {
		t.Fatalf("空消息不应返回回复: %+v", out)
	}
	if gateway.gotReq.Query != "" {
		t.Error("空消息不应触发 Chat 调用")
	}
}
