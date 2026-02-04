package chat

import (
	"encoding/json"
	"strings"
)

type toolResult struct {
	ToolCallID string
	Content    string
}

func normalizeGatewayMessages(messages []GatewayMessage) []GatewayMessage {
	normalized := make([]GatewayMessage, 0, len(messages))
	for _, msg := range messages {
		items := normalizeGatewayMessage(msg)
		normalized = append(normalized, toGatewayMessages(items)...)
	}
	return normalized
}

func normalizeGatewayMessage(msg GatewayMessage) []NormalizedMessage {
	if msg == nil {
		return nil
	}
	role := getString(msg["role"])
	if role == "" {
		role = "assistant"
	}

	var toolCalls []ToolCall
	var textParts []ContentPart
	var toolResults []toolResult

	if rawCalls, ok := msg["tool_calls"].([]interface{}); ok {
		for _, raw := range rawCalls {
			if call := normalizeToolCall(raw); call.Function.Name != "" {
				toolCalls = append(toolCalls, call)
			}
		}
	}

	switch content := msg["content"].(type) {
	case string:
		if strings.TrimSpace(content) != "" || len(toolCalls) > 0 {
			normalized := NormalizedMessage{Role: role}
			if strings.TrimSpace(content) != "" {
				normalized.Content = content
			}
			if len(toolCalls) > 0 {
				normalized.ToolCalls = toolCalls
			}
			return appendToolResults([]NormalizedMessage{normalized}, toolResults)
		}
	case []interface{}:
		for _, part := range content {
			switch p := part.(type) {
			case string:
				if strings.TrimSpace(p) != "" {
					textParts = append(textParts, ContentPart{Type: "text", Text: p})
				}
			case map[string]interface{}:
				if text := normalizeTextPart(p); text != "" {
					textParts = append(textParts, ContentPart{Type: "text", Text: text})
					continue
				}
				if call := normalizeToolCall(p); call.Function.Name != "" {
					toolCalls = append(toolCalls, call)
					continue
				}
				if result := normalizeToolResult(p); result.ToolCallID != "" {
					toolResults = append(toolResults, result)
					continue
				}
				if encoded := toJSONString(p); encoded != "" {
					textParts = append(textParts, ContentPart{Type: "text", Text: encoded})
				}
			default:
				if encoded := toJSONString(p); encoded != "" {
					textParts = append(textParts, ContentPart{Type: "text", Text: encoded})
				}
			}
		}
	case map[string]interface{}:
		if text := normalizeTextPart(content); text != "" {
			textParts = append(textParts, ContentPart{Type: "text", Text: text})
		} else if encoded := toJSONString(content); encoded != "" {
			textParts = append(textParts, ContentPart{Type: "text", Text: encoded})
		}
	}

	if len(textParts) == 0 && len(toolCalls) == 0 && len(toolResults) == 0 {
		return nil
	}

	output := NormalizedMessage{Role: role}
	if len(toolCalls) > 0 {
		output.ToolCalls = toolCalls
	}
	if len(textParts) == 1 && len(toolCalls) == 0 {
		output.Content = textParts[0].Text
	} else if len(textParts) > 0 {
		output.Parts = textParts
	}

	return appendToolResults([]NormalizedMessage{output}, toolResults)
}

func appendToolResults(messages []NormalizedMessage, results []toolResult) []NormalizedMessage {
	if len(results) == 0 {
		return messages
	}
	for _, result := range results {
		if strings.TrimSpace(result.ToolCallID) == "" {
			continue
		}
		item := NormalizedMessage{
			Role:       "tool",
			ToolCallID: result.ToolCallID,
		}
		if strings.TrimSpace(result.Content) != "" {
			item.Content = result.Content
		}
		messages = append(messages, item)
	}
	return messages
}

func normalizeTextPart(part map[string]interface{}) string {
	if part == nil {
		return ""
	}
	if partType, _ := part["type"].(string); partType == "text" {
		if text, ok := part["text"].(string); ok {
			return text
		}
	}
	if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return ""
}

func normalizeToolCall(part interface{}) ToolCall {
	switch value := part.(type) {
	case map[string]interface{}:
		if valueType, _ := value["type"].(string); valueType == "tool_use" || valueType == "tool-call" || valueType == "function_call" {
			return ToolCall{
				ID:   getString(value["id"]),
				Type: "function",
				Function: ToolCallFunction{
					Name:      getString(value["name"]),
					Arguments: toJSONString(value["input"], value["args"], value["arguments"]),
				},
			}
		}
		if fc, ok := value["function_call"].(map[string]interface{}); ok {
			return ToolCall{
				ID:   getString(value["id"]),
				Type: "function",
				Function: ToolCallFunction{
					Name:      getString(fc["name"]),
					Arguments: toJSONString(fc["arguments"], fc["args"]),
				},
			}
		}
		if fc, ok := value["functionCall"].(map[string]interface{}); ok {
			return ToolCall{
				ID:   getString(value["id"]),
				Type: "function",
				Function: ToolCallFunction{
					Name:      getString(fc["name"]),
					Arguments: toJSONString(fc["args"], fc["arguments"]),
				},
			}
		}
		if fn, ok := value["function"].(map[string]interface{}); ok {
			return ToolCall{
				ID:   getString(value["id"]),
				Type: "function",
				Function: ToolCallFunction{
					Name:      getString(fn["name"]),
					Arguments: toJSONString(fn["arguments"]),
				},
			}
		}
	}
	return ToolCall{}
}

func normalizeToolResult(part map[string]interface{}) toolResult {
	if part == nil {
		return toolResult{}
	}
	if partType, _ := part["type"].(string); partType == "tool_result" || partType == "tool-result" {
		return toolResult{
			ToolCallID: firstString(part["tool_use_id"], part["toolCallId"], part["tool_call_id"], part["id"]),
			Content:    normalizeToolResultContent(part["content"], part["result"], part["output"]),
		}
	}
	if raw, ok := part["toolResult"].(map[string]interface{}); ok {
		return toolResult{
			ToolCallID: firstString(raw["toolUseId"], raw["tool_call_id"], raw["id"]),
			Content:    normalizeToolResultContent(raw["content"], raw["output"], raw["result"]),
		}
	}
	if raw, ok := part["functionResponse"].(map[string]interface{}); ok {
		return toolResult{
			ToolCallID: firstString(raw["id"]),
			Content:    normalizeToolResultContent(raw["response"], raw["output"], raw["result"]),
		}
	}
	return toolResult{}
}

func normalizeToolResultContent(values ...interface{}) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		case []interface{}:
			parts := make([]string, 0, len(v))
			for _, item := range v {
				switch itemValue := item.(type) {
				case string:
					if strings.TrimSpace(itemValue) != "" {
						parts = append(parts, itemValue)
					}
				case map[string]interface{}:
					if text := normalizeTextPart(itemValue); text != "" {
						parts = append(parts, text)
					} else if encoded := toJSONString(itemValue); encoded != "" {
						parts = append(parts, encoded)
					}
				default:
					if encoded := toJSONString(itemValue); encoded != "" {
						parts = append(parts, encoded)
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		case map[string]interface{}:
			if text := normalizeTextPart(v); text != "" {
				return text
			}
			if encoded := toJSONString(v); encoded != "" {
				return encoded
			}
		default:
			if encoded := toJSONString(v); encoded != "" {
				return encoded
			}
		}
	}
	return ""
}

func toGatewayMessages(messages []NormalizedMessage) []GatewayMessage {
	converted := make([]GatewayMessage, 0, len(messages))
	for _, msg := range messages {
		item := GatewayMessage{
			"role": msg.Role,
		}
		if strings.TrimSpace(msg.Content) != "" {
			item["content"] = msg.Content
		} else if len(msg.Parts) > 0 {
			parts := make([]map[string]interface{}, 0, len(msg.Parts))
			for _, part := range msg.Parts {
				entry := map[string]interface{}{
					"type": part.Type,
				}
				if strings.TrimSpace(part.Text) != "" {
					entry["text"] = part.Text
				}
				parts = append(parts, entry)
			}
			item["content"] = parts
		}
		if len(msg.ToolCalls) > 0 {
			payload := make([]map[string]interface{}, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				if strings.TrimSpace(call.Function.Name) == "" {
					continue
				}
				entry := map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":      call.Function.Name,
						"arguments": call.Function.Arguments,
					},
				}
				if strings.TrimSpace(call.ID) != "" {
					entry["id"] = call.ID
				}
				payload = append(payload, entry)
			}
			if len(payload) > 0 {
				item["tool_calls"] = payload
			}
		}
		if strings.TrimSpace(msg.ToolCallID) != "" {
			item["tool_call_id"] = msg.ToolCallID
		}
		if strings.TrimSpace(msg.Name) != "" {
			item["name"] = msg.Name
		}
		converted = append(converted, item)
	}
	return converted
}

func getString(value interface{}) string {
	if raw, ok := value.(string); ok {
		return raw
	}
	return ""
}

func firstString(values ...interface{}) string {
	for _, value := range values {
		if raw, ok := value.(string); ok && strings.TrimSpace(raw) != "" {
			return raw
		}
	}
	return ""
}

func toJSONString(values ...interface{}) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		if raw, ok := value.(string); ok {
			if strings.TrimSpace(raw) != "" {
				return raw
			}
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(encoded)) == "" {
			continue
		}
		return string(encoded)
	}
	return ""
}
