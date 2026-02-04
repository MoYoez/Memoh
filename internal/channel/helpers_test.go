package channel

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseChannelType(t *testing.T) {
	t.Parallel()

	got, err := ParseChannelType(" Telegram ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != ChannelTelegram {
		t.Fatalf("unexpected channel type: %s", got)
	}

	if _, err := ParseChannelType("unknown"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMatchTelegramBinding(t *testing.T) {
	t.Parallel()

	cfg := TelegramUserConfig{
		Username: "Alice",
		UserID:   "u1",
		ChatID:   "c1",
	}
	if !matchTelegramBinding(cfg, BindingCriteria{ChatID: "c1"}) {
		t.Fatalf("expected chat id match")
	}
	if !matchTelegramBinding(cfg, BindingCriteria{UserID: "u1"}) {
		t.Fatalf("expected user id match")
	}
	if !matchTelegramBinding(cfg, BindingCriteria{Username: "alice"}) {
		t.Fatalf("expected username match")
	}
	if matchTelegramBinding(cfg, BindingCriteria{Username: "bob"}) {
		t.Fatalf("expected no match")
	}
}

func TestMatchFeishuBinding(t *testing.T) {
	t.Parallel()

	cfg := FeishuUserConfig{
		OpenID: "ou_1",
		UserID: "u_1",
	}
	if !matchFeishuBinding(cfg, BindingCriteria{OpenID: "ou_1"}) {
		t.Fatalf("expected open_id match")
	}
	if !matchFeishuBinding(cfg, BindingCriteria{UserID: "u_1"}) {
		t.Fatalf("expected user_id match")
	}
	if matchFeishuBinding(cfg, BindingCriteria{UserID: "u_2"}) {
		t.Fatalf("expected no match")
	}
}

func TestDecodeConfigMap(t *testing.T) {
	t.Parallel()

	cfg, err := decodeConfigMap([]byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg["a"] == nil {
		t.Fatalf("expected key in map")
	}
	cfg, err = decodeConfigMap([]byte(`null`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil || len(cfg) != 0 {
		t.Fatalf("expected empty map")
	}
}

func TestReadString(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"bot_token": 123,
	}
	got := readString(raw, "bot_token")
	if got != "123" {
		t.Fatalf("unexpected value: %s", got)
	}
}

func TestParseUUID(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	if _, err := parseUUID(id); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := parseUUID("invalid"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseTelegramUserConfigTrims(t *testing.T) {
	t.Parallel()

	cfg, err := parseTelegramUserConfig(map[string]interface{}{
		"username": " alice ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Username != "alice" {
		t.Fatalf("unexpected username: %s", cfg.Username)
	}
}

func TestResolveTargetFromUserConfigMissing(t *testing.T) {
	t.Parallel()

	if _, err := resolveTargetFromUserConfig(ChannelTelegram, map[string]interface{}{}); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if _, err := resolveTargetFromUserConfig(ChannelFeishu, map[string]interface{}{}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
