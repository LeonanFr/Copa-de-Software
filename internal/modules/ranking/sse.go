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
	log.Printf("SSE: cliente adicionado. Total: %d", len(m.clients))
}

func (m *SSEManager) RemoveClient(ch chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, ch)
	close(ch)
	log.Printf("SSE: cliente removido. Total: %d", len(m.clients))
}

func (m *SSEManager) Broadcast(data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE: erro ao serializar: %v", err)
		return
	}
	log.Printf("SSE: broadcast para %d clientes", len(m.clients))
	for ch := range m.clients {
		select {
		case ch <- msg:
			log.Printf("SSE: mensagem enviada para um cliente")
		default:
			log.Printf("SSE: cliente com canal cheio, ignorando")
		}
	}
}

func (h *Handler) RankingSSE(w http.ResponseWriter, r *http.Request) {
	log.Printf("SSE: nova conexão de %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 10)
	Manager.AddClient(ch)
	defer Manager.RemoveClient(ch)

	ranking, err := h.service.GetRanking(r.Context())
	if err != nil {
		log.Printf("SSE: erro ao obter ranking inicial: %v", err)
	} else {
		data, _ := json.Marshal(ranking)
		_, err := fmt.Fprintf(w, "data: %s\n\n", data)
		if err != nil {
			return
		}
		w.(http.Flusher).Flush()
		log.Printf("SSE: ranking inicial enviado")
	}

	<-r.Context().Done()
	log.Printf("SSE: cliente %s desconectado", r.RemoteAddr)
}
