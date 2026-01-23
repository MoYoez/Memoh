package models

import (
	"errors"
)

type ModelType string

const (
	ModelTypeChat      = "chat"
	ModelTypeEmbedding = "embedding"
)

type ClientType string

const (
	ClientTypeOpenAI    ClientType = "openai"
	ClientTypeAnthropic ClientType = "anthropic"
	ClientTypeGoogle    ClientType = "google"
)

type Model struct {
	ModelID    string     `json:"model_id"`
	Name       string     `json:"name"`
	BaseURL    string     `json:"base_url"`
	APIKey     string     `json:"api_key"`
	ClientType ClientType `json:"client_type"`
	Type       ModelType  `json:"type"`
	Dimensions int        `json:"dimensions"`
}

func (m *Model) Validate() error {
	if m.ModelID == "" {
		return errors.New("model ID is required")
	}
	if m.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if m.APIKey == "" {
		return errors.New("API key is required")
	}
	if m.ClientType == "" {
		return errors.New("client type is required")
	}
	if m.Type != ModelTypeChat && m.Type != ModelTypeEmbedding {
		return errors.New("invalid model type")
	}
	if m.ClientType != ClientTypeOpenAI && m.ClientType != ClientTypeAnthropic && m.ClientType != ClientTypeGoogle {
		return errors.New("invalid client type")
	}
	if m.Type == ModelTypeEmbedding && m.Dimensions <= 0 {
		return errors.New("dimensions must be greater than 0")
	}
	return nil
}

type AddRequest Model

type AddResponse struct {
	ID      string `json:"id"`
	ModelID string `json:"model_id"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	ModelId string `json:"model_id"`
	Model
}

type UpdateRequest Model

type ListRequest struct {
	Type       ModelType  `json:"type,omitempty"`
	ClientType ClientType `json:"client_type,omitempty"`
}

type DeleteRequest struct {
	ID      string `json:"id,omitempty"`
	ModelID string `json:"model_id,omitempty"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type CountResponse struct {
	Count int64 `json:"count"`
}
