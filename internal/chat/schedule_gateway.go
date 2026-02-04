package chat

import (
	"context"
	"fmt"

	"github.com/memohai/memoh/internal/schedule"
)

// ScheduleGateway 将 schedule 触发请求转交给 chat Resolver。
type ScheduleGateway struct {
	resolver *Resolver
}

func NewScheduleGateway(resolver *Resolver) *ScheduleGateway {
	return &ScheduleGateway{resolver: resolver}
}

func (g *ScheduleGateway) TriggerSchedule(ctx context.Context, botID string, payload schedule.TriggerPayload, token string) error {
	if g == nil || g.resolver == nil {
		return fmt.Errorf("chat resolver not configured")
	}
	return g.resolver.TriggerSchedule(ctx, botID, SchedulePayload{
		ID:          payload.ID,
		Name:        payload.Name,
		Description: payload.Description,
		Pattern:     payload.Pattern,
		MaxCalls:    payload.MaxCalls,
		Command:     payload.Command,
	}, token)
}
