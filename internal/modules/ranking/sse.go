package ranking

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type SSEManager struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

var Manager = &SSEManager{
	clients: make(map[chan []byte]bool),
}

func (m *SSEManager) AddClient(ch chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[ch] = true
}

func (m *SSEManager) RemoveClient(ch chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, ch)
	close(ch)
}

func (m *SSEManager) Broadcast(data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, err := json.Marshal(data)
	if err != nil {
		log.Printf("Erro ao serializar SSE: %v", err)
		return
	}
	for ch := range m.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Handler) RankingSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 10)
	Manager.AddClient(ch)
	defer Manager.RemoveClient(ch)

	ranking, err := h.service.GetRanking(r.Context())
	if err == nil {
		data, _ := json.Marshal(ranking)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
	}

	<-r.Context().Done()
}
