package channel

import (
	"fmt"
	"time"
)

type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelFeishu   ChannelType = "feishu"
	ChannelCLI      ChannelType = "cli"
	ChannelWeb      ChannelType = "web"
)

func ParseChannelType(raw string) (ChannelType, error) {
	normalized := normalizeChannelType(raw)
	if normalized == "" {
		return "", fmt.Errorf("unsupported channel type: %s", raw)
	}
	if _, ok := GetChannelDescriptor(normalized); !ok {
		return "", fmt.Errorf("unsupported channel type: %s", raw)
	}
	return normalized, nil
}

type ChannelConfig struct {
	ID               string
	BotID            string
	ChannelType      ChannelType
	Credentials      map[string]interface{}
	ExternalIdentity string
	SelfIdentity     map[string]interface{}
	Routing          map[string]interface{}
	Capabilities     map[string]interface{}
	Status           string
	VerifiedAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ChannelUserBinding struct {
	ID          string
	ChannelType ChannelType
	UserID      string
	Config      map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UpsertConfigRequest struct {
	Credentials      map[string]interface{} `json:"credentials"`
	ExternalIdentity string                 `json:"external_identity,omitempty"`
	SelfIdentity     map[string]interface{} `json:"self_identity,omitempty"`
	Routing          map[string]interface{} `json:"routing,omitempty"`
	Capabilities     map[string]interface{} `json:"capabilities,omitempty"`
	Status           string                 `json:"status,omitempty"`
	VerifiedAt       *time.Time             `json:"verified_at,omitempty"`
}

type UpsertUserConfigRequest struct {
	Config map[string]interface{} `json:"config"`
}

type ChannelSession struct {
	SessionID       string
	BotID           string
	ChannelConfigID string
	UserID          string
	ContactID       string
	Platform        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SendRequest struct {
	To       string `json:"to"`
	ToUserID string `json:"to_user_id"`
	Message  string `json:"message"`
}
