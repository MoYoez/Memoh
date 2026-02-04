package schedule

import "context"

// TriggerPayload 描述触发定时任务时传递给聊天侧的参数。
type TriggerPayload struct {
	ID          string
	Name        string
	Description string
	Pattern     string
	MaxCalls    *int
	Command     string
}

// Triggerer 负责触发与聊天相关的调度执行。
type Triggerer interface {
	TriggerSchedule(ctx context.Context, botID string, payload TriggerPayload, token string) error
}
