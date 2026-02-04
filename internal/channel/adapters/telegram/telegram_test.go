package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestResolveTelegramSender(t *testing.T) {
	t.Parallel()

	id, name := resolveTelegramSender(nil)
	if id != "" || name != "" {
		t.Fatalf("expected empty sender")
	}
	user := &tgbotapi.User{ID: 123, UserName: "alice"}
	id, name = resolveTelegramSender(user)
	if id != "123" || name != "alice" {
		t.Fatalf("unexpected sender: %s %s", id, name)
	}
}
