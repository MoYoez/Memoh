package channel

import (
	"sync"

	"github.com/google/uuid"
)

type SessionHub struct {
	mu       sync.RWMutex
	sessions map[string]map[string]chan OutboundMessage
}

func NewSessionHub() *SessionHub {
	return &SessionHub{
		sessions: map[string]map[string]chan OutboundMessage{},
	}
}

func (h *SessionHub) Subscribe(sessionID string) (string, <-chan OutboundMessage, func()) {
	streamID := uuid.NewString()
	ch := make(chan OutboundMessage, 32)

	h.mu.Lock()
	streams, ok := h.sessions[sessionID]
	if !ok {
		streams = map[string]chan OutboundMessage{}
		h.sessions[sessionID] = streams
	}
	streams[streamID] = ch
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		streams := h.sessions[sessionID]
		if streams != nil {
			if current, ok := streams[streamID]; ok {
				delete(streams, streamID)
				close(current)
			}
			if len(streams) == 0 {
				delete(h.sessions, sessionID)
			}
		}
		h.mu.Unlock()
	}

	return streamID, ch, cancel
}

func (h *SessionHub) Publish(sessionID string, msg OutboundMessage) {
	h.mu.RLock()
	streams := h.sessions[sessionID]
	h.mu.RUnlock()
	if len(streams) == 0 {
		return
	}

	for _, stream := range streams {
		select {
		case stream <- msg:
		default:
			// Drop if receiver is slow.
		}
	}
}
