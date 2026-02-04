package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/channel/adapters/common"
)

type TelegramAdapter struct {
	logger *slog.Logger
}

func NewTelegramAdapter(log *slog.Logger) *TelegramAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &TelegramAdapter{
		logger: log.With(slog.String("adapter", "telegram")),
	}
}

func (a *TelegramAdapter) Type() channel.ChannelType {
	return channel.ChannelTelegram
}

func (a *TelegramAdapter) Start(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.AdapterRunner, error) {
	if a.logger != nil {
		a.logger.Info("start", slog.String("config_id", cfg.ID))
	}
	telegramCfg, err := decodeTelegramConfig(cfg.Credentials)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return channel.AdapterRunner{}, err
	}
	bot, err := tgbotapi.NewBotAPI(telegramCfg.BotToken)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("create bot failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return channel.AdapterRunner{}, err
	}
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updates := bot.GetUpdatesChan(updateConfig)

	go func() {
		for {
			select {
			case <-ctx.Done():
				if a.logger != nil {
					a.logger.Info("stop", slog.String("config_id", cfg.ID))
				}
				bot.StopReceivingUpdates()
				return
			case update, ok := <-updates:
				if !ok {
					if a.logger != nil {
						a.logger.Info("updates channel closed", slog.String("config_id", cfg.ID))
					}
					return
				}
				if update.Message == nil {
					continue
				}
				text := strings.TrimSpace(update.Message.Text)
				if text == "" {
					continue
				}
				userID, username := resolveTelegramSender(update.Message.From)
				chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
				msg := channel.InboundMessage{
					Channel:  channel.ChannelTelegram,
					Text:     text,
					Username: username,
					UserID:   userID,
					ChatID:   chatID,
					ChatType: update.Message.Chat.Type,
					ReplyTo:  chatID,
					BotID:    cfg.BotID,
				}
				if a.logger != nil {
					a.logger.Info(
						"inbound received",
						slog.String("config_id", cfg.ID),
						slog.String("chat_type", msg.ChatType),
						slog.String("chat_id", msg.ChatID),
						slog.String("user_id", msg.UserID),
						slog.String("username", msg.Username),
						slog.String("text", common.SummarizeText(msg.Text)),
					)
				}
				go func() {
					if err := handler(ctx, cfg, msg); err != nil && a.logger != nil {
						a.logger.Error("handle inbound failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
					}
				}()
			}
		}
	}()

	return channel.AdapterRunner{
		Stop: func() {
			if a.logger != nil {
				a.logger.Info("stop", slog.String("config_id", cfg.ID))
			}
			bot.StopReceivingUpdates()
		},
		SupportsStop: true,
	}, nil
}

func (a *TelegramAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	telegramCfg, err := decodeTelegramConfig(cfg.Credentials)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return fmt.Errorf("telegram target is required")
	}
	bot, err := tgbotapi.NewBotAPI(telegramCfg.BotToken)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("create bot failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return fmt.Errorf("message is required")
	}
	if strings.HasPrefix(to, "@") {
		message := tgbotapi.NewMessageToChannel(to, text)
		_, err = bot.Send(message)
		if err != nil && a.logger != nil {
			a.logger.Error("send failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}
	chatID, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram target must be @username or chat_id")
	}
	message := tgbotapi.NewMessage(chatID, text)
	_, err = bot.Send(message)
	if err != nil && a.logger != nil {
		a.logger.Error("send failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
	}
	return err
}

func resolveTelegramSender(user *tgbotapi.User) (string, string) {
	if user == nil {
		return "", ""
	}
	return strconv.FormatInt(user.ID, 10), strings.TrimSpace(user.UserName)
}

func decodeTelegramConfig(raw map[string]interface{}) (channel.TelegramConfig, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return channel.TelegramConfig{}, err
	}
	return channel.DecodeTelegramConfig(payload)
}
