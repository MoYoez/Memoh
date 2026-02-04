package channel

import (
	"testing"
)

func TestResolveTargetFromUserConfigTelegram(t *testing.T) {
	t.Parallel()

	target, err := resolveTargetFromUserConfig(ChannelTelegram, map[string]interface{}{
		"chat_id":  "123",
		"user_id":  "456",
		"username": "alice",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target != "123" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestResolveTargetFromUserConfigTelegramUsername(t *testing.T) {
	t.Parallel()

	target, err := resolveTargetFromUserConfig(ChannelTelegram, map[string]interface{}{
		"username": "alice",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target != "@alice" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestResolveTargetFromUserConfigFeishu(t *testing.T) {
	t.Parallel()

	target, err := resolveTargetFromUserConfig(ChannelFeishu, map[string]interface{}{
		"open_id": "ou_123",
		"user_id": "u_123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target != "open_id:ou_123" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestResolveTargetFromUserConfigUnsupported(t *testing.T) {
	t.Parallel()

	_, err := resolveTargetFromUserConfig("unknown", map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
