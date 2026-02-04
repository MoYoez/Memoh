package identity

import "strings"

const (
	UserTypeHuman = "human"
	UserTypeBot   = "bot"
)

func IsBotUserType(userType string) bool {
	return strings.EqualFold(strings.TrimSpace(userType), UserTypeBot)
}
