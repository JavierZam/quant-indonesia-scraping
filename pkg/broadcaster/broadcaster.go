package broadcaster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// IngestionEvent represents a real-time progress update event.
type IngestionEvent struct {
	Type      string   `json:"type"`       // "start", "progress", "log", "done", "error"
	Stage     string   `json:"stage"`      // "SCRAPE", "AI_ANALYSIS", "PERSIST"
	Current   int      `json:"current"`    // e.g. 5
	Total     int      `json:"total"`      // e.g. 30
	Message   string   `json:"message"`    // e.g. "Analyzing article: Danantara Beli Saham BEI..."
	Ticker    string   `json:"ticker,omitempty"`
	Sentiment string   `json:"sentiment,omitempty"`
	Score     *float64 `json:"score,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// Broadcaster manages SSE client connections and broadcasts real-time ingestion events.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[chan IngestionEvent]bool
}

// NewBroadcaster creates a new Broadcaster instance.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan IngestionEvent]bool),
	}
}

// Subscribe adds a new SSE listener channel.
func (b *Broadcaster) Subscribe() chan IngestionEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan IngestionEvent, 500)
	b.clients[ch] = true
	return ch
}

// Unsubscribe removes an SSE listener channel.
func (b *Broadcaster) Unsubscribe(ch chan IngestionEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients, ch)
	close(ch)
}

// Broadcast sends an event to all connected SSE clients.
func (b *Broadcaster) Broadcast(event IngestionEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format("15:04:05")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	isCritical := event.Type == "start" || event.Type == "done" || event.Type == "cancelled" || event.Type == "error"

	for ch := range b.clients {
		if isCritical {
			// Critical events must be delivered: attempt send with short 100ms timeout
			select {
			case ch <- event:
			case <-time.After(100 * time.Millisecond):
			}
		} else {
			select {
			case ch <- event:
			default:
				// Skip non-critical progress ticks if client channel is temporarily full
			}
		}
	}
}

// Handler handles GET /api/v1/ingestion/stream via Server-Sent Events (SSE).
func (b *Broadcaster) Handler(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Send initial connection ping
	initEvent := IngestionEvent{
		Type:      "connected",
		Message:   "Connected to Real-time Ingestion Stream",
		Timestamp: time.Now().Format("15:04:05"),
	}
	data, _ := json.Marshal(initEvent)
	fmt.Fprintf(c.Response(), "data: %s\n\n", data)
	c.Response().Flush()

	notify := c.Request().Context().Done()

	for {
		select {
		case <-notify:
			return nil
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			eventData, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", eventData); err != nil {
				return nil
			}
			c.Response().Flush()
		}
	}
}
