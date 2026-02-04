package chat

import "encoding/json"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GatewayMessage map[string]interface{}

type AgentSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

type ChatRequest struct {
	BotID              string           `json:"-"`
	SessionID          string           `json:"-"`
	Token              string           `json:"-"`
	UserID             string           `json:"-"`
	Query              string           `json:"query"`
	Model              string           `json:"model,omitempty"`
	Provider           string           `json:"provider,omitempty"`
	MaxContextLoadTime int              `json:"max_context_load_time,omitempty"`
	Locale             string           `json:"locale,omitempty"`
	Language           string           `json:"language,omitempty"`
	MaxSteps           int              `json:"max_steps,omitempty"`
	Platforms          []string         `json:"platforms,omitempty"`
	CurrentPlatform    string           `json:"current_platform,omitempty"`
	Messages           []GatewayMessage `json:"messages,omitempty"`
	Skills             []AgentSkill     `json:"skills,omitempty"`
	UseSkills          []string         `json:"use_skills,omitempty"`
	ToolContext        *ToolContext     `json:"toolContext,omitempty"`
	ToolChoice         map[string]any   `json:"toolChoice,omitempty"`
}

type ChatResponse struct {
	Messages []GatewayMessage `json:"messages"`
	Skills   []string         `json:"skills,omitempty"`
	Model    string           `json:"model,omitempty"`
	Provider string           `json:"provider,omitempty"`
}

type StreamChunk = json.RawMessage

type SchedulePayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	MaxCalls    *int   `json:"maxCalls,omitempty"`
	Command     string `json:"command"`
}

type ToolContext struct {
	BotID           string `json:"botId,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
	CurrentPlatform string `json:"currentPlatform,omitempty"`
	ReplyTarget     string `json:"replyTarget,omitempty"`
	SessionToken    string `json:"sessionToken,omitempty"`
	ContactID       string `json:"contactId,omitempty"`
	ContactName     string `json:"contactName,omitempty"`
	ContactAlias    string `json:"contactAlias,omitempty"`
	UserID          string `json:"userId,omitempty"`
}

// NormalizedMessage 是内部统一后的消息结构，屏蔽厂商差异。
type NormalizedMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	Parts      []ContentPart `json:"parts,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	Name       string        `json:"name,omitempty"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
