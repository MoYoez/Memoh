package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/history"
	"github.com/memohai/memoh/internal/memory"
	"github.com/memohai/memoh/internal/models"
	"github.com/memohai/memoh/internal/settings"
)

const defaultMaxContextMinutes = 24 * 60

type Resolver struct {
	modelsService   *models.Service
	queries         *sqlc.Queries
	memoryService   *memory.Service
	historyService  *history.Service
	settingsService *settings.Service
	gatewayBaseURL  string
	timeout         time.Duration
	logger          *slog.Logger
	httpClient      *http.Client
	streamingClient *http.Client
}

type userSettings struct {
	ChatModelID        string
	MemoryModelID      string
	EmbeddingModelID   string
	MaxContextLoadTime int
	Language           string
}

func NewResolver(log *slog.Logger, modelsService *models.Service, queries *sqlc.Queries, memoryService *memory.Service, historyService *history.Service, settingsService *settings.Service, gatewayBaseURL string, timeout time.Duration) *Resolver {
	if strings.TrimSpace(gatewayBaseURL) == "" {
		gatewayBaseURL = "http://127.0.0.1:8081"
	}
	gatewayBaseURL = strings.TrimRight(gatewayBaseURL, "/")
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Resolver{
		modelsService:   modelsService,
		queries:         queries,
		memoryService:   memoryService,
		historyService:  historyService,
		settingsService: settingsService,
		gatewayBaseURL:  gatewayBaseURL,
		timeout:         timeout,
		logger:          log.With(slog.String("service", "chat")),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		streamingClient: &http.Client{},
	}
}

func (r *Resolver) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return ChatResponse{}, fmt.Errorf("query is required")
	}
	if strings.TrimSpace(req.BotID) == "" {
		return ChatResponse{}, fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return ChatResponse{}, fmt.Errorf("session id is required")
	}
	skipHistory := req.MaxContextLoadTime < 0

	settings, err := r.loadUserSettings(ctx, req.UserID)
	if err != nil {
		return ChatResponse{}, err
	}
	chatModel, provider, err := r.selectChatModel(ctx, req, settings)
	if err != nil {
		return ChatResponse{}, err
	}
	clientType, err := normalizeClientType(provider.ClientType)
	if err != nil {
		return ChatResponse{}, err
	}

	maxContextLoadTime, language, err := r.loadBotSettings(ctx, req.BotID)
	if err != nil {
		return ChatResponse{}, err
	}
	if req.MaxContextLoadTime > 0 {
		maxContextLoadTime = req.MaxContextLoadTime
	}
	if strings.TrimSpace(req.Language) != "" {
		language = req.Language
	}

	var messages []GatewayMessage
	var historySkills []string
	if !skipHistory {
		messages, err = r.loadHistoryMessages(ctx, req.BotID, req.SessionID, maxContextLoadTime)
		if err != nil {
			return ChatResponse{}, err
		}
		historySkills, err = r.loadHistorySkills(ctx, req.BotID, req.SessionID, maxContextLoadTime)
		if err != nil {
			return ChatResponse{}, err
		}
	}
	if len(req.Messages) > 0 {
		messages = append(messages, req.Messages...)
	}
	messages = sanitizeGatewayMessages(messages)
	messages = normalizeGatewayMessagesForModel(messages)
	useSkills := normalizeSkills(append(historySkills, req.UseSkills...))

	payload := agentGatewayRequest{
		APIKey:             provider.ApiKey,
		BaseURL:            provider.BaseUrl,
		Model:              chatModel.ModelID,
		ClientType:         clientType,
		Locale:             req.Locale,
		Language:           req.Language,
		MaxSteps:           req.MaxSteps,
		MaxContextLoadTime: normalizeMaxContextLoad(maxContextLoadTime),
		Platforms:          req.Platforms,
		CurrentPlatform:    req.CurrentPlatform,
		Messages:           messages,
		Query:              req.Query,
		Skills:             req.Skills,
		UseSkills:          useSkills,
		ToolContext:        req.ToolContext,
		ToolChoice:         req.ToolChoice,
	}
	payload.Language = language

	resp, err := r.postChat(ctx, payload, req.Token)
	if err != nil {
		return ChatResponse{}, err
	}
	resp.Messages = normalizeGatewayMessages(resp.Messages)

	if err := r.storeHistory(ctx, req.BotID, req.SessionID, req.Query, resp.Messages, resp.Skills); err != nil {
		return ChatResponse{}, err
	}
	if err := r.storeMemory(ctx, req.BotID, req.SessionID, req.Query, resp.Messages); err != nil {
		return ChatResponse{}, err
	}

	return ChatResponse{
		Messages: resp.Messages,
		Skills:   resp.Skills,
		Model:    chatModel.ModelID,
		Provider: provider.ClientType,
	}, nil
}

func (r *Resolver) TriggerSchedule(ctx context.Context, botID string, schedule SchedulePayload, token string) error {
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(schedule.Command) == "" {
		return fmt.Errorf("schedule command is required")
	}

	req := ChatRequest{
		BotID:     botID,
		SessionID: "schedule:" + schedule.ID,
		Query:     schedule.Command,
		Locale:    "",
		Language:  "",
	}
	settings, err := r.loadUserSettings(ctx, "")
	if err != nil {
		return err
	}
	chatModel, provider, err := r.selectChatModel(ctx, req, settings)
	if err != nil {
		return err
	}
	clientType, err := normalizeClientType(provider.ClientType)
	if err != nil {
		return err
	}

	maxContextLoadTime, language, err := r.loadBotSettings(ctx, botID)
	if err != nil {
		return err
	}

	messages, err := r.loadHistoryMessages(ctx, botID, req.SessionID, maxContextLoadTime)
	if err != nil {
		return err
	}
	historySkills, err := r.loadHistorySkills(ctx, botID, req.SessionID, maxContextLoadTime)
	if err != nil {
		return err
	}
	useSkills := normalizeSkills(historySkills)

	payload := agentGatewayScheduleRequest{
		APIKey:             provider.ApiKey,
		BaseURL:            provider.BaseUrl,
		Model:              chatModel.ModelID,
		ClientType:         clientType,
		Locale:             "",
		Language:           language,
		MaxSteps:           0,
		MaxContextLoadTime: normalizeMaxContextLoad(maxContextLoadTime),
		Platforms:          nil,
		CurrentPlatform:    "",
		Messages:           messages,
		Query:              schedule.Command,
		Schedule:           schedule,
		UseSkills:          useSkills,
	}

	resp, err := r.postSchedule(ctx, payload, token)
	if err != nil {
		return err
	}
	resp.Messages = normalizeGatewayMessages(resp.Messages)
	if err := r.storeHistory(ctx, botID, req.SessionID, schedule.Command, resp.Messages, resp.Skills); err != nil {
		return err
	}
	if err := r.storeMemory(ctx, botID, req.SessionID, schedule.Command, resp.Messages); err != nil {
		return err
	}
	return nil
}

func (r *Resolver) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		if strings.TrimSpace(req.Query) == "" {
			errChan <- fmt.Errorf("query is required")
			return
		}
		if strings.TrimSpace(req.BotID) == "" {
			errChan <- fmt.Errorf("bot id is required")
			return
		}
		if strings.TrimSpace(req.SessionID) == "" {
			errChan <- fmt.Errorf("session id is required")
			return
		}
		skipHistory := req.MaxContextLoadTime < 0

		settings, err := r.loadUserSettings(ctx, req.UserID)
		if err != nil {
			errChan <- err
			return
		}
		chatModel, provider, err := r.selectChatModel(ctx, req, settings)
		if err != nil {
			errChan <- err
			return
		}
		clientType, err := normalizeClientType(provider.ClientType)
		if err != nil {
			errChan <- err
			return
		}

		maxContextLoadTime, language, err := r.loadBotSettings(ctx, req.BotID)
		if err != nil {
			errChan <- err
			return
		}
		if req.MaxContextLoadTime > 0 {
			maxContextLoadTime = req.MaxContextLoadTime
		}
		if strings.TrimSpace(req.Language) != "" {
			language = req.Language
		}

		var messages []GatewayMessage
		var historySkills []string
		if !skipHistory {
			messages, err = r.loadHistoryMessages(ctx, req.BotID, req.SessionID, maxContextLoadTime)
			if err != nil {
				errChan <- err
				return
			}
			historySkills, err = r.loadHistorySkills(ctx, req.BotID, req.SessionID, maxContextLoadTime)
			if err != nil {
				errChan <- err
				return
			}
		}
		if len(req.Messages) > 0 {
			messages = append(messages, req.Messages...)
		}
		messages = sanitizeGatewayMessages(messages)
		messages = normalizeGatewayMessagesForModel(messages)
		useSkills := normalizeSkills(append(historySkills, req.UseSkills...))

		payload := agentGatewayRequest{
			APIKey:             provider.ApiKey,
			BaseURL:            provider.BaseUrl,
			Model:              chatModel.ModelID,
			ClientType:         clientType,
			Locale:             req.Locale,
			Language:           req.Language,
			MaxSteps:           req.MaxSteps,
			MaxContextLoadTime: normalizeMaxContextLoad(maxContextLoadTime),
			Platforms:          req.Platforms,
			CurrentPlatform:    req.CurrentPlatform,
			Messages:           messages,
			Query:              req.Query,
			Skills:             req.Skills,
			UseSkills:          useSkills,
			ToolContext:        req.ToolContext,
			ToolChoice:         req.ToolChoice,
		}
		payload.Language = language

		if err := r.streamChat(ctx, payload, req.BotID, req.SessionID, req.Query, req.Token, chunkChan); err != nil {
			errChan <- err
			return
		}
	}()

	return chunkChan, errChan
}

type agentGatewayRequest struct {
	APIKey             string           `json:"apiKey"`
	BaseURL            string           `json:"baseUrl"`
	Model              string           `json:"model"`
	ClientType         string           `json:"clientType"`
	Locale             string           `json:"locale,omitempty"`
	Language           string           `json:"language,omitempty"`
	MaxSteps           int              `json:"maxSteps,omitempty"`
	MaxContextLoadTime int              `json:"maxContextLoadTime"`
	Platforms          []string         `json:"platforms,omitempty"`
	CurrentPlatform    string           `json:"currentPlatform,omitempty"`
	Messages           []GatewayMessage `json:"messages"`
	Query              string           `json:"query"`
	Skills             []AgentSkill     `json:"skills,omitempty"`
	UseSkills          []string         `json:"useSkills,omitempty"`
	ToolContext        *ToolContext     `json:"toolContext,omitempty"`
	ToolChoice         interface{}      `json:"toolChoice,omitempty"`
}

type agentGatewayScheduleRequest struct {
	APIKey             string           `json:"apiKey"`
	BaseURL            string           `json:"baseUrl"`
	Model              string           `json:"model"`
	ClientType         string           `json:"clientType"`
	Locale             string           `json:"locale,omitempty"`
	Language           string           `json:"language,omitempty"`
	MaxSteps           int              `json:"maxSteps,omitempty"`
	MaxContextLoadTime int              `json:"maxContextLoadTime"`
	Platforms          []string         `json:"platforms,omitempty"`
	CurrentPlatform    string           `json:"currentPlatform,omitempty"`
	Messages           []GatewayMessage `json:"messages"`
	Query              string           `json:"query"`
	Schedule           SchedulePayload  `json:"schedule"`
	Skills             []AgentSkill     `json:"skills,omitempty"`
	UseSkills          []string         `json:"useSkills,omitempty"`
}

type agentGatewayResponse struct {
	Messages []GatewayMessage `json:"messages"`
	Skills   []string         `json:"skills"`
}

func (r *Resolver) postChat(ctx context.Context, payload agentGatewayRequest, token string) (agentGatewayResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return agentGatewayResponse{}, err
	}
	url := r.gatewayBaseURL + "/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return agentGatewayResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return agentGatewayResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return agentGatewayResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agentGatewayResponse{}, fmt.Errorf("agent gateway error: %s", strings.TrimSpace(string(respBody)))
	}

	parsed, err := parseAgentGatewayResponse(respBody)
	if err != nil {
		r.logger.Error("failed to parse agent gateway response", slog.String("body", string(respBody)), slog.Any("error", err))
		return agentGatewayResponse{}, err
	}
	return parsed, nil
}

func (r *Resolver) postSchedule(ctx context.Context, payload agentGatewayScheduleRequest, token string) (agentGatewayResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return agentGatewayResponse{}, err
	}
	url := r.gatewayBaseURL + "/chat/schedule"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return agentGatewayResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return agentGatewayResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return agentGatewayResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agentGatewayResponse{}, fmt.Errorf("agent gateway error: %s", strings.TrimSpace(string(respBody)))
	}

	parsed, err := parseAgentGatewayResponse(respBody)
	if err != nil {
		r.logger.Error("failed to parse schedule gateway response", slog.String("body", string(respBody)), slog.Any("error", err))
		return agentGatewayResponse{}, err
	}
	return parsed, nil
}

func (r *Resolver) streamChat(ctx context.Context, payload agentGatewayRequest, botID, sessionID, query, token string, chunkChan chan<- StreamChunk) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := r.gatewayBaseURL + "/chat/stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := r.streamingClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent gateway error: %s", strings.TrimSpace(string(payload)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	currentEventType := ""
	stored := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			currentEventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		chunkChan <- StreamChunk([]byte(data))

		if stored {
			continue
		}

		if handled, err := r.tryStoreFromStreamPayload(ctx, botID, sessionID, query, currentEventType, data); err != nil {
			return err
		} else if handled {
			stored = true
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (r *Resolver) loadHistoryMessages(ctx context.Context, botID, sessionID string, maxContextLoadTime int) ([]GatewayMessage, error) {
	if r.historyService == nil {
		return nil, fmt.Errorf("history service not configured")
	}
	from := time.Now().UTC().Add(-time.Duration(normalizeMaxContextLoad(maxContextLoadTime)) * time.Minute)
	records, err := r.historyService.ListBySessionSince(ctx, botID, sessionID, from)
	if err != nil {
		return nil, err
	}
	messages := make([]GatewayMessage, 0, len(records))
	for _, record := range records {
		if len(record.Messages) == 0 {
			continue
		}
		for _, msg := range record.Messages {
			if msg == nil {
				continue
			}
			messages = append(messages, GatewayMessage(msg))
		}
	}
	return messages, nil
}

func (r *Resolver) loadHistorySkills(ctx context.Context, botID, sessionID string, maxContextLoadTime int) ([]string, error) {
	if r.historyService == nil {
		return nil, fmt.Errorf("history service not configured")
	}
	from := time.Now().UTC().Add(-time.Duration(normalizeMaxContextLoad(maxContextLoadTime)) * time.Minute)
	records, err := r.historyService.ListBySessionSince(ctx, botID, sessionID, from)
	if err != nil {
		return nil, err
	}
	combined := make([]string, 0, len(records))
	for _, record := range records {
		if len(record.Skills) == 0 {
			continue
		}
		combined = append(combined, record.Skills...)
	}
	return normalizeSkills(combined), nil
}

func (r *Resolver) storeHistory(ctx context.Context, botID, sessionID, query string, responseMessages []GatewayMessage, skills []string) error {
	if r.historyService == nil {
		return fmt.Errorf("history service not configured")
	}
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("bot id is required")
	}
	trimmedSession := strings.TrimSpace(sessionID)
	if trimmedSession == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(query) == "" && len(responseMessages) == 0 {
		return nil
	}
	messages := make([]map[string]interface{}, 0, len(responseMessages))
	for _, msg := range responseMessages {
		if msg == nil {
			continue
		}
		messages = append(messages, map[string]interface{}(msg))
	}
	_, err := r.historyService.Create(ctx, botID, trimmedSession, history.CreateRequest{
		Messages: messages,
		Skills:   skills,
	})
	return err
}

func (r *Resolver) storeMemory(ctx context.Context, botID, sessionID, query string, responseMessages []GatewayMessage) error {
	if r.memoryService == nil {
		return nil
	}
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("bot id is required")
	}
	trimmedSession := strings.TrimSpace(sessionID)
	if trimmedSession == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(query) == "" && len(responseMessages) == 0 {
		return nil
	}

	memoryMessages := make([]memory.Message, 0, len(responseMessages))
	for _, msg := range responseMessages {
		role, content := gatewayMessageToMemory(msg)
		if strings.TrimSpace(content) == "" {
			continue
		}
		memoryMessages = append(memoryMessages, memory.Message{
			Role:    role,
			Content: content,
		})
	}
	if len(memoryMessages) == 0 {
		return nil
	}

	_, err := r.memoryService.Add(ctx, memory.AddRequest{
		Messages:  memoryMessages,
		BotID:     botID,
		SessionID: trimmedSession,
	})
	return err
}

func (r *Resolver) tryStoreFromStreamPayload(ctx context.Context, botID, sessionID, query, eventType, data string) (bool, error) {
	// Case 1: event: done + data: {messages: [...]}
	if eventType == "done" {
		if parsed, ok := parseGatewayResponse([]byte(data)); ok {
			parsed.Messages = normalizeGatewayMessages(parsed.Messages)
			return r.storeRound(ctx, botID, sessionID, query, parsed.Messages, parsed.Skills)
		}
	}

	// Case 2: data: {"type":"done","data":{messages:[...]}}
	var envelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(data), &envelope); err == nil {
		if envelope.Type == "done" && len(envelope.Data) > 0 {
			if parsed, ok := parseGatewayResponse(envelope.Data); ok {
				parsed.Messages = normalizeGatewayMessages(parsed.Messages)
				return r.storeRound(ctx, botID, sessionID, query, parsed.Messages, parsed.Skills)
			}
		}
	}

	// Case 3: data: {messages:[...]} without event
	if parsed, ok := parseGatewayResponse([]byte(data)); ok {
		parsed.Messages = normalizeGatewayMessages(parsed.Messages)
		return r.storeRound(ctx, botID, sessionID, query, parsed.Messages, parsed.Skills)
	}
	return false, nil
}

func parseGatewayResponse(payload []byte) (agentGatewayResponse, bool) {
	parsed, err := parseAgentGatewayResponse(payload)
	if err != nil {
		return agentGatewayResponse{}, false
	}
	if len(parsed.Messages) == 0 {
		return agentGatewayResponse{}, false
	}
	return parsed, true
}

// parseAgentGatewayResponse parses the agent gateway response with flexible message handling.
// It can handle various message formats from different AI SDK versions.
func parseAgentGatewayResponse(payload []byte) (agentGatewayResponse, error) {
	// Use json.RawMessage to handle flexible message formats
	var raw struct {
		Messages []json.RawMessage `json:"messages"`
		Skills   []string          `json:"skills"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return agentGatewayResponse{}, fmt.Errorf("failed to parse response structure: %w", err)
	}

	messages := make([]GatewayMessage, 0, len(raw.Messages))
	for _, rawMsg := range raw.Messages {
		// Try parsing as object
		var msg map[string]interface{}
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			// If it's an array, try to extract messages from it
			var arr []interface{}
			if err := json.Unmarshal(rawMsg, &arr); err == nil {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						messages = append(messages, GatewayMessage(m))
					}
				}
				continue
			}
			// Skip unparseable messages
			continue
		}
		messages = append(messages, GatewayMessage(msg))
	}

	return agentGatewayResponse{
		Messages: messages,
		Skills:   raw.Skills,
	}, nil
}

func (r *Resolver) storeRound(ctx context.Context, botID, sessionID, query string, messages []GatewayMessage, skills []string) (bool, error) {
	if err := r.storeHistory(ctx, botID, sessionID, query, messages, skills); err != nil {
		return true, err
	}
	if err := r.storeMemory(ctx, botID, sessionID, query, messages); err != nil {
		return true, err
	}
	return true, nil
}

func normalizeSkills(skills []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(skills))
	for _, skill := range skills {
		trimmed := strings.TrimSpace(skill)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func gatewayMessageToMemory(msg GatewayMessage) (string, string) {
	role := "assistant"
	if raw, ok := msg["role"].(string); ok && strings.TrimSpace(raw) != "" {
		role = raw
	}
	if raw, ok := msg["content"]; ok {
		switch v := raw.(type) {
		case string:
			return role, v
		default:
			if encoded, err := json.Marshal(v); err == nil {
				return role, string(encoded)
			}
		}
	}
	if encoded, err := json.Marshal(msg); err == nil {
		return role, string(encoded)
	}
	return role, ""
}

func sanitizeGatewayMessages(messages []GatewayMessage) []GatewayMessage {
	if len(messages) == 0 {
		return messages
	}
	cleaned := make([]GatewayMessage, 0, len(messages))
	for _, msg := range messages {
		if !isMeaningfulGatewayMessage(msg) {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return cleaned
}

func normalizeGatewayMessagesForModel(messages []GatewayMessage) []GatewayMessage {
	if len(messages) == 0 {
		return messages
	}
	cleaned := make([]GatewayMessage, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		role, content := gatewayMessageToMemory(msg)
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		if strings.TrimSpace(role) == "" {
			role = "assistant"
		}
		if role == "tool" {
			role = "assistant"
			content = "[tool] " + content
		}
		cleaned = append(cleaned, GatewayMessage{
			"role":    role,
			"content": content,
		})
	}
	return cleaned
}

func isMeaningfulGatewayMessage(msg GatewayMessage) bool {
	if len(msg) == 0 {
		return false
	}
	if raw, ok := msg["role"].(string); ok && strings.TrimSpace(raw) != "" {
		return true
	}
	if raw, ok := msg["content"]; ok {
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return true
			}
		default:
			if !isEmptyValue(v) {
				return true
			}
		}
	}
	for _, value := range msg {
		if !isEmptyValue(value) {
			return true
		}
	}
	return false
}

func isEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		if len(v) == 0 {
			return true
		}
		for _, item := range v {
			if !isEmptyValue(item) {
				return false
			}
		}
		return true
	case map[string]interface{}:
		if len(v) == 0 {
			return true
		}
		for _, item := range v {
			if !isEmptyValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (r *Resolver) selectChatModel(ctx context.Context, req ChatRequest, settings userSettings) (models.GetResponse, sqlc.LlmProvider, error) {
	if r.modelsService == nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("models service not configured")
	}
	modelID := strings.TrimSpace(req.Model)
	providerFilter := strings.TrimSpace(req.Provider)

	if modelID != "" && providerFilter == "" {
		model, err := r.modelsService.GetByModelID(ctx, modelID)
		if err != nil {
			return models.GetResponse{}, sqlc.LlmProvider{}, err
		}
		if model.Type != models.ModelTypeChat {
			return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("model is not a chat model")
		}
		provider, err := models.FetchProviderByID(ctx, r.queries, model.LlmProviderID)
		if err != nil {
			return models.GetResponse{}, sqlc.LlmProvider{}, err
		}
		return model, provider, nil
	}

	if providerFilter == "" && modelID == "" && strings.TrimSpace(settings.ChatModelID) != "" {
		selected, err := r.modelsService.GetByModelID(ctx, settings.ChatModelID)
		if err != nil {
			return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("settings chat model not found: %w", err)
		}
		if selected.Type != models.ModelTypeChat {
			return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("settings chat model is not a chat model")
		}
		provider, err := models.FetchProviderByID(ctx, r.queries, selected.LlmProviderID)
		if err != nil {
			return models.GetResponse{}, sqlc.LlmProvider{}, err
		}
		return selected, provider, nil
	}

	var candidates []models.GetResponse
	var err error
	if providerFilter != "" {
		candidates, err = r.modelsService.ListByClientType(ctx, models.ClientType(providerFilter))
	} else {
		candidates, err = r.modelsService.ListByType(ctx, models.ModelTypeChat)
	}
	if err != nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, err
	}

	filtered := make([]models.GetResponse, 0, len(candidates))
	for _, model := range candidates {
		if model.Type != models.ModelTypeChat {
			continue
		}
		filtered = append(filtered, model)
	}
	if len(filtered) == 0 {
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("no chat models available")
	}

	if modelID != "" {
		for _, model := range filtered {
			if model.ModelID == modelID {
				provider, err := models.FetchProviderByID(ctx, r.queries, model.LlmProviderID)
				if err != nil {
					return models.GetResponse{}, sqlc.LlmProvider{}, err
				}
				return model, provider, nil
			}
		}
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("chat model not found")
	}

	selected := filtered[0]
	provider, err := models.FetchProviderByID(ctx, r.queries, selected.LlmProviderID)
	if err != nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, err
	}
	return selected, provider, nil
}

func normalizeMaxContextLoad(value int) int {
	if value <= 0 {
		return defaultMaxContextMinutes
	}
	return value
}

func (r *Resolver) loadUserSettings(ctx context.Context, userID string) (userSettings, error) {
	defaults := userSettings{
		MaxContextLoadTime: defaultMaxContextMinutes,
		Language:           "Same as user input",
	}
	if r.settingsService == nil || strings.TrimSpace(userID) == "" {
		return defaults, nil
	}
	settingsRow, err := r.settingsService.Get(ctx, userID)
	if err != nil {
		return userSettings{}, err
	}
	maxLoad := settingsRow.MaxContextLoadTime
	if maxLoad <= 0 {
		maxLoad = defaultMaxContextMinutes
	}
	language := strings.TrimSpace(settingsRow.Language)
	if language == "" {
		language = "Same as user input"
	}
	return userSettings{
		ChatModelID:        strings.TrimSpace(settingsRow.ChatModelID),
		MemoryModelID:      strings.TrimSpace(settingsRow.MemoryModelID),
		EmbeddingModelID:   strings.TrimSpace(settingsRow.EmbeddingModelID),
		MaxContextLoadTime: maxLoad,
		Language:           language,
	}, nil
}

func (r *Resolver) loadBotSettings(ctx context.Context, botID string) (int, string, error) {
	if r.settingsService == nil {
		return settings.DefaultMaxContextLoadTime, settings.DefaultLanguage, nil
	}
	settingsRow, err := r.settingsService.GetBot(ctx, botID)
	if err != nil {
		return 0, "", err
	}
	return settingsRow.MaxContextLoadTime, settingsRow.Language, nil
}

func normalizeClientType(clientType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(clientType)) {
	case "openai":
		return "openai", nil
	case "openai-compat":
		return "openai", nil
	case "anthropic":
		return "anthropic", nil
	case "google":
		return "google", nil
	default:
		return "", fmt.Errorf("unsupported agent gateway client type: %s", clientType)
	}
}

func parseUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	var pgID pgtype.UUID
	pgID.Valid = true
	copy(pgID.Bytes[:], parsed[:])
	return pgID, nil
}
