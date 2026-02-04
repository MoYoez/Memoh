package channel

import "context"

// InboundProcessor 负责处理入站消息并产出可发送的响应。
type InboundProcessor interface {
	HandleInbound(ctx context.Context, cfg ChannelConfig, msg InboundMessage) (*OutboundMessage, error)
}
