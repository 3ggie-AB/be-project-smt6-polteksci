package websocket

import (
	"io"
	"log/slog"
	"sync"
	"time"

	"project_smt6/domain"

	"github.com/gin-gonic/gin"
)

type Broker struct {
	mu      sync.RWMutex
	clients map[chan domain.RealtimeEvent]struct{}
	logger  *slog.Logger
}

func NewBroker(logger *slog.Logger) *Broker {
	return &Broker{
		clients: make(map[chan domain.RealtimeEvent]struct{}),
		logger:  logger,
	}
}

func (b *Broker) Publish(event domain.RealtimeEvent) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for client := range b.clients {
		select {
		case client <- event:
		default:
			b.logger.Warn("sse client queue full; dropping realtime event", "type", event.Type)
		}
	}
}

func (b *Broker) ServeSSE() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := make(chan domain.RealtimeEvent, 128)
		b.subscribe(client)
		defer b.unsubscribe(client)

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		c.Stream(func(w io.Writer) bool {
			select {
			case event := <-client:
				c.SSEvent(event.Type, event)
				return true
			case <-heartbeat.C:
				c.SSEvent("heartbeat", gin.H{"ts": time.Now().UTC()})
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	}
}

func (b *Broker) subscribe(client chan domain.RealtimeEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[client] = struct{}{}
}

func (b *Broker) unsubscribe(client chan domain.RealtimeEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, client)
	close(client)
}
