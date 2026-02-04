package channel

import (
	"fmt"
	"strings"
	"sync"
)

type ChannelDescriptor struct {
	Type                ChannelType
	DisplayName         string
	NormalizeConfig     func(map[string]interface{}) (map[string]interface{}, error)
	NormalizeUserConfig func(map[string]interface{}) (map[string]interface{}, error)
}

type channelRegistry struct {
	mu    sync.RWMutex
	items map[ChannelType]ChannelDescriptor
}

var registry = &channelRegistry{
	items: map[ChannelType]ChannelDescriptor{},
}

func RegisterChannel(desc ChannelDescriptor) error {
	normalized := normalizeChannelType(string(desc.Type))
	if normalized == "" {
		return fmt.Errorf("channel type is required")
	}
	desc.Type = normalized
	if strings.TrimSpace(desc.DisplayName) == "" {
		desc.DisplayName = normalized.String()
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.items[desc.Type]; exists {
		return fmt.Errorf("channel type already registered: %s", desc.Type)
	}
	registry.items[desc.Type] = desc
	return nil
}

func MustRegisterChannel(desc ChannelDescriptor) {
	if err := RegisterChannel(desc); err != nil {
		panic(err)
	}
}

func GetChannelDescriptor(channelType ChannelType) (ChannelDescriptor, bool) {
	normalized := normalizeChannelType(channelType.String())
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	desc, ok := registry.items[normalized]
	return desc, ok
}

func ListChannelDescriptors() []ChannelDescriptor {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	items := make([]ChannelDescriptor, 0, len(registry.items))
	for _, item := range registry.items {
		items = append(items, item)
	}
	return items
}

func normalizeChannelType(raw string) ChannelType {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return ""
	}
	return ChannelType(normalized)
}
