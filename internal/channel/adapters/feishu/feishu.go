package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/channel/adapters/common"
)

type FeishuAdapter struct {
	logger *slog.Logger
}

func NewFeishuAdapter(log *slog.Logger) *FeishuAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &FeishuAdapter{
		logger: log.With(slog.String("adapter", "feishu")),
	}
}

func (a *FeishuAdapter) Type() channel.ChannelType {
	return channel.ChannelFeishu
}

func (a *FeishuAdapter) Start(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.AdapterRunner, error) {
	if a.logger != nil {
		a.logger.Info("start", slog.String("config_id", cfg.ID))
	}
	feishuCfg, err := decodeFeishuConfig(cfg.Credentials)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return channel.AdapterRunner{}, err
	}
	eventDispatcher := dispatcher.NewEventDispatcher(
		feishuCfg.VerificationToken,
		feishuCfg.EncryptKey,
	)
	eventDispatcher.OnP2MessageReceiveV1(func(_ context.Context, event *larkim.P2MessageReceiveV1) error {
		msg := extractFeishuInbound(event)
		if msg.Text == "" {
			return nil
		}
		msg.BotID = cfg.BotID
		if a.logger != nil {
			a.logger.Info(
				"inbound received",
				slog.String("config_id", cfg.ID),
				slog.String("session_id", msg.SessionID()),
				slog.String("chat_type", msg.ChatType),
				slog.String("text", common.SummarizeText(msg.Text)),
			)
		}
		go func() {
			if err := handler(ctx, cfg, msg); err != nil && a.logger != nil {
				a.logger.Error("handle inbound failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
			}
		}()
		return nil
	})
	eventDispatcher.OnP2MessageReadV1(func(_ context.Context, _ *larkim.P2MessageReadV1) error {
		return nil
	})

	client := larkws.NewClient(
		feishuCfg.AppID,
		feishuCfg.AppSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogger(newLarkSlogLogger(a.logger)),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	go func() {
		if err := client.Start(ctx); err != nil && a.logger != nil {
			a.logger.Error("client start failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
	}()

	return channel.AdapterRunner{
		Stop:         func() {},
		SupportsStop: false,
	}, nil
}

func (a *FeishuAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	feishuCfg, err := decodeFeishuConfig(cfg.Credentials)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return fmt.Errorf("message is required")
	}
	receiveID, receiveType, err := resolveFeishuReceiveID(strings.TrimSpace(msg.To))
	if err != nil {
		return err
	}
	contentPayload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	client := lark.NewClient(feishuCfg.AppID, feishuCfg.AppSecret)
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(receiveID).
		MsgType(larkim.MsgTypeText).
		Content(string(contentPayload)).
		Uuid(uuid.NewString()).
		Build()
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveType).
		Body(body).
		Build()
	resp, err := client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("send failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}
	if resp == nil || !resp.Success() {
		if a.logger != nil {
			code := 0
			msg := ""
			if resp != nil {
				code = resp.Code
				msg = resp.Msg
			}
			a.logger.Error("send failed", slog.String("config_id", cfg.ID), slog.Int("code", code), slog.String("msg", msg))
		}
		return fmt.Errorf("feishu send failed")
	}
	if a.logger != nil {
		a.logger.Info("send success", slog.String("config_id", cfg.ID))
	}
	return nil
}

func extractFeishuInbound(event *larkim.P2MessageReceiveV1) channel.InboundMessage {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return channel.InboundMessage{Channel: channel.ChannelFeishu}
	}
	message := event.Event.Message
	if message.MessageType == nil || *message.MessageType != larkim.MsgTypeText {
		return channel.InboundMessage{Channel: channel.ChannelFeishu}
	}
	var payload struct {
		Text string `json:"text"`
	}
	if message.Content != nil {
		_ = json.Unmarshal([]byte(*message.Content), &payload)
	}
	senderID, senderOpenID := "", ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		if event.Event.Sender.SenderId.UserId != nil {
			senderID = strings.TrimSpace(*event.Event.Sender.SenderId.UserId)
		}
		if event.Event.Sender.SenderId.OpenId != nil {
			senderOpenID = strings.TrimSpace(*event.Event.Sender.SenderId.OpenId)
		}
	}
	chatID := ""
	chatType := ""
	if message.ChatId != nil {
		chatID = strings.TrimSpace(*message.ChatId)
	}
	if message.ChatType != nil {
		chatType = strings.TrimSpace(*message.ChatType)
	}
	replyTo := senderOpenID
	if replyTo == "" {
		replyTo = senderID
	}
	if chatType != "" && chatType != "p2p" && chatID != "" {
		replyTo = "chat_id:" + chatID
	}
	return channel.InboundMessage{
		Channel:  channel.ChannelFeishu,
		Text:     strings.TrimSpace(payload.Text),
		UserID:   senderID,
		OpenID:   senderOpenID,
		ChatID:   chatID,
		ChatType: chatType,
		ReplyTo:  replyTo,
	}
}

func resolveFeishuReceiveID(raw string) (string, string, error) {
	if raw == "" {
		return "", "", fmt.Errorf("feishu target is required")
	}
	if strings.HasPrefix(raw, "open_id:") {
		return strings.TrimPrefix(raw, "open_id:"), larkim.ReceiveIdTypeOpenId, nil
	}
	if strings.HasPrefix(raw, "user_id:") {
		return strings.TrimPrefix(raw, "user_id:"), larkim.ReceiveIdTypeUserId, nil
	}
	if strings.HasPrefix(raw, "chat_id:") {
		return strings.TrimPrefix(raw, "chat_id:"), larkim.ReceiveIdTypeChatId, nil
	}
	return raw, larkim.ReceiveIdTypeOpenId, nil
}

func decodeFeishuConfig(raw map[string]interface{}) (channel.FeishuConfig, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return channel.FeishuConfig{}, err
	}
	return channel.DecodeFeishuConfig(payload)
}
