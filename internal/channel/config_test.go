package channel

import "testing"

func TestNormalizeChannelConfigTelegram(t *testing.T) {
	t.Parallel()

	got, err := NormalizeChannelConfig(ChannelTelegram, map[string]interface{}{
		"bot_token": "token-123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["botToken"] != "token-123" {
		t.Fatalf("unexpected botToken: %#v", got["botToken"])
	}
}

func TestNormalizeChannelConfigTelegramRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := NormalizeChannelConfig(ChannelTelegram, map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNormalizeChannelConfigFeishu(t *testing.T) {
	t.Parallel()

	got, err := NormalizeChannelConfig(ChannelFeishu, map[string]interface{}{
		"app_id":     "app",
		"app_secret": "secret",
		"encrypt_key": "enc",
		"verification_token": "verify",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["appId"] != "app" || got["appSecret"] != "secret" {
		t.Fatalf("unexpected feishu config: %#v", got)
	}
	if got["encryptKey"] != "enc" || got["verificationToken"] != "verify" {
		t.Fatalf("unexpected feishu security config: %#v", got)
	}
}

func TestNormalizeChannelUserConfigTelegram(t *testing.T) {
	t.Parallel()

	got, err := NormalizeChannelUserConfig(ChannelTelegram, map[string]interface{}{
		"username": "alice",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["username"] != "alice" {
		t.Fatalf("unexpected username: %#v", got["username"])
	}
}

func TestNormalizeChannelUserConfigTelegramRequiresBinding(t *testing.T) {
	t.Parallel()

	_, err := NormalizeChannelUserConfig(ChannelTelegram, map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNormalizeChannelUserConfigFeishu(t *testing.T) {
	t.Parallel()

	got, err := NormalizeChannelUserConfig(ChannelFeishu, map[string]interface{}{
		"open_id": "ou_123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["open_id"] != "ou_123" {
		t.Fatalf("unexpected open_id: %#v", got["open_id"])
	}
}

func TestNormalizeChannelUserConfigFeishuRequiresBinding(t *testing.T) {
	t.Parallel()

	_, err := NormalizeChannelUserConfig(ChannelFeishu, map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
