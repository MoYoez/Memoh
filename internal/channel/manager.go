package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type ConfigStore interface {
	ResolveEffectiveConfig(ctx context.Context, botID string, channelType ChannelType) (ChannelConfig, error)
	GetUserConfig(ctx context.Context, actorUserID string, channelType ChannelType) (ChannelUserBinding, error)
	UpsertUserConfig(ctx context.Context, actorUserID string, channelType ChannelType, req UpsertUserConfigRequest) (ChannelUserBinding, error)
	ListConfigsByType(ctx context.Context, channelType ChannelType) ([]ChannelConfig, error)
	ResolveUserBinding(ctx context.Context, channelType ChannelType, criteria BindingCriteria) (string, error)
	GetChannelSession(ctx context.Context, sessionID string) (ChannelSession, error)
	UpsertChannelSession(ctx context.Context, sessionID string, botID string, channelConfigID string, userID string, contactID string, platform string) error
}

// Middleware 消息处理中间件定义
type Middleware func(next InboundHandler) InboundHandler

type Manager struct {
	service         ConfigStore
	processor       InboundProcessor
	adapters        map[ChannelType]Adapter
	refreshInterval time.Duration
	logger          *slog.Logger
	middlewares     []Middleware

	mu      sync.Mutex
	runners map[string]*runningAdapter
}

type runningAdapter struct {
	adapter      Adapter
	config       ChannelConfig
	stop         func()
	supportsStop bool
}

func NewManager(log *slog.Logger, service ConfigStore, processor InboundProcessor) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{
		service:         service,
		processor:       processor,
		adapters:        map[ChannelType]Adapter{},
		refreshInterval: 30 * time.Second,
		runners:         map[string]*runningAdapter{},
		logger:          log.With(slog.String("component", "channel")),
		middlewares:     []Middleware{},
	}
}

// Use 注册中间件
func (m *Manager) Use(mw ...Middleware) {
	m.middlewares = append(m.middlewares, mw...)
}

func (m *Manager) RegisterAdapter(adapter Adapter) {
	if adapter == nil {
		return
	}
	m.adapters[adapter.Type()] = adapter
	if m.logger != nil {
		m.logger.Info("adapter registered", slog.String("channel", adapter.Type().String()))
	}
}

func (m *Manager) Start(ctx context.Context) {
	if m.logger != nil {
		m.logger.Info("manager start")
	}
	go func() {
		m.refresh(ctx)
		ticker := time.NewTicker(m.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				if m.logger != nil {
					m.logger.Info("manager stop")
				}
				m.stopAll()
				return
			case <-ticker.C:
				m.refresh(ctx)
			}
		}
	}()
}

func (m *Manager) Send(ctx context.Context, botID string, channelType ChannelType, req SendRequest) error {
	if m.service == nil {
		return fmt.Errorf("channel manager not configured")
	}
	adapter := m.adapters[channelType]
	if adapter == nil {
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
	config, err := m.service.ResolveEffectiveConfig(ctx, botID, channelType)
	if err != nil {
		return err
	}
	target := strings.TrimSpace(req.To)
	if target == "" {
		targetUserID := strings.TrimSpace(req.ToUserID)
		if targetUserID == "" {
			return fmt.Errorf("target user_id is required")
		}
		userCfg, err := m.service.GetUserConfig(ctx, targetUserID, channelType)
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("channel binding missing", slog.String("channel", channelType.String()), slog.String("user_id", targetUserID))
			}
			return fmt.Errorf("channel binding required")
		}
		target, err = resolveTargetFromUserConfig(channelType, userCfg.Config)
		if err != nil {
			return err
		}
	}
	text := strings.TrimSpace(req.Message)
	if text == "" {
		return fmt.Errorf("message is required")
	}
	if m.logger != nil {
		m.logger.Info("send outbound", slog.String("channel", channelType.String()), slog.String("bot_id", botID))
	}
	err = adapter.Send(ctx, config, OutboundMessage{
		To:   target,
		Text: text,
	})
	if err != nil && m.logger != nil {
		m.logger.Error("send outbound failed", slog.String("channel", channelType.String()), slog.String("bot_id", botID), slog.Any("error", err))
	}
	return err
}

func (m *Manager) HandleInbound(ctx context.Context, cfg ChannelConfig, msg InboundMessage) error {
	return m.handleInbound(ctx, cfg, msg)
}

func (m *Manager) refresh(ctx context.Context) {
	if m.service == nil {
		return
	}
	configs := make([]ChannelConfig, 0)
	for channelType := range m.adapters {
		items, err := m.service.ListConfigsByType(ctx, channelType)
		if err != nil {
			if m.logger != nil {
				m.logger.Error("list configs failed", slog.String("channel", channelType.String()), slog.Any("error", err))
			}
			continue
		}
		configs = append(configs, items...)
	}
	m.reconcile(ctx, configs)
}

func (m *Manager) reconcile(ctx context.Context, configs []ChannelConfig) {
	active := map[string]ChannelConfig{}
	for _, cfg := range configs {
		if cfg.ID == "" {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(cfg.Status))
		if status != "" && status != "active" && status != "verified" {
			continue
		}
		active[cfg.ID] = cfg
		if err := m.ensureRunner(ctx, cfg); err != nil {
			if m.logger != nil {
				m.logger.Error("adapter start failed", slog.String("channel", cfg.ChannelType.String()), slog.String("config_id", cfg.ID), slog.Any("error", err))
			}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for id, runner := range m.runners {
		if _, ok := active[id]; ok {
			continue
		}
		if runner.supportsStop && runner.stop != nil {
			if m.logger != nil {
				m.logger.Info("adapter stop", slog.String("channel", runner.config.ChannelType.String()), slog.String("config_id", id))
			}
			runner.stop()
		}
		delete(m.runners, id)
	}
}

func (m *Manager) ensureRunner(ctx context.Context, cfg ChannelConfig) error {
	m.mu.Lock()
	runner := m.runners[cfg.ID]
	m.mu.Unlock()

	if runner != nil {
		if runner.config.UpdatedAt.Equal(cfg.UpdatedAt) {
			return nil
		}
		if !runner.supportsStop || runner.stop == nil {
			if m.logger != nil {
				m.logger.Warn("adapter restart skipped", slog.String("channel", cfg.ChannelType.String()), slog.String("config_id", cfg.ID))
			}
			return nil
		}
		if m.logger != nil {
			m.logger.Info("adapter restart", slog.String("channel", cfg.ChannelType.String()), slog.String("config_id", cfg.ID))
		}
		runner.stop()
		m.mu.Lock()
		delete(m.runners, cfg.ID)
		m.mu.Unlock()
	}

	adapter := m.adapters[cfg.ChannelType]
	if adapter == nil {
		return fmt.Errorf("unsupported channel type: %s", cfg.ChannelType)
	}
	if m.logger != nil {
		m.logger.Info("adapter start", slog.String("channel", cfg.ChannelType.String()), slog.String("config_id", cfg.ID))
	}

	// 包装中间件
	handler := m.handleInbound
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}

	started, err := adapter.Start(ctx, cfg, handler)
	if err != nil {
		return err
	}
	entry := &runningAdapter{
		adapter:      adapter,
		config:       cfg,
		stop:         started.Stop,
		supportsStop: started.SupportsStop,
	}
	m.mu.Lock()
	m.runners[cfg.ID] = entry
	m.mu.Unlock()
	return nil
}

func (m *Manager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, runner := range m.runners {
		if runner.supportsStop && runner.stop != nil {
			if m.logger != nil {
				m.logger.Info("adapter stop", slog.String("channel", runner.config.ChannelType.String()), slog.String("config_id", id))
			}
			runner.stop()
		}
		delete(m.runners, id)
	}
}

func (m *Manager) handleInbound(ctx context.Context, cfg ChannelConfig, msg InboundMessage) error {
	if m.processor == nil {
		return fmt.Errorf("inbound processor not configured")
	}
	reply, err := m.processor.HandleInbound(ctx, cfg, msg)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("inbound processing failed", slog.String("channel", msg.Channel.String()), slog.Any("error", err))
		}
		return err
	}
	if reply == nil || strings.TrimSpace(reply.Text) == "" {
		return nil
	}
	adapter := m.adapters[msg.Channel]
	if adapter == nil {
		return fmt.Errorf("unsupported channel type: %s", msg.Channel)
	}
	target := strings.TrimSpace(reply.To)
	if target == "" {
		return fmt.Errorf("reply target missing")
	}
	if m.logger != nil {
		m.logger.Info("send reply", slog.String("channel", msg.Channel.String()))
	}

	// 增加简单的重试逻辑
	var lastErr error
	for i := 0; i < 3; i++ {
		err = adapter.Send(ctx, cfg, OutboundMessage{
			To:   target,
			Text: reply.Text,
		})
		if err == nil {
			return nil
		}
		lastErr = err
		if m.logger != nil {
			m.logger.Warn("send reply retry",
				slog.String("channel", msg.Channel.String()),
				slog.Int("attempt", i+1),
				slog.Any("error", err))
		}
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond) // 指数退避
	}

	return fmt.Errorf("send reply failed after retries: %w", lastErr)
}

func resolveTargetFromUserConfig(channelType ChannelType, config map[string]interface{}) (string, error) {
	switch channelType {
	case ChannelTelegram:
		userCfg, err := parseTelegramUserConfig(config)
		if err != nil {
			return "", err
		}
		if userCfg.ChatID != "" {
			return userCfg.ChatID, nil
		}
		if userCfg.UserID != "" {
			return userCfg.UserID, nil
		}
		if userCfg.Username != "" {
			name := userCfg.Username
			if !strings.HasPrefix(name, "@") {
				name = "@" + name
			}
			return name, nil
		}
		return "", fmt.Errorf("telegram binding is incomplete")
	case ChannelFeishu:
		userCfg, err := parseFeishuUserConfig(config)
		if err != nil {
			return "", err
		}
		if userCfg.OpenID != "" {
			return "open_id:" + userCfg.OpenID, nil
		}
		if userCfg.UserID != "" {
			return "user_id:" + userCfg.UserID, nil
		}
		return "", fmt.Errorf("feishu binding is incomplete")
	default:
		return "", fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
