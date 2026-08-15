package main

import (
	"sync"
	"time"
)

// Message representa a estrutura de dados trocada via WebSocket.
type Message struct {
	Type      string    `json:"type"`              // "message", "join", "leave", "user_list", "history", "clear_history", "history_cleared"
	Sender    string    `json:"sender,omitempty"`  // Nome do utilizador
	Content   string    `json:"content,omitempty"` // Texto da mensagem
	Timestamp string    `json:"timestamp,omitempty"`
	Color     string    `json:"color,omitempty"`   // Cor única atribuída ao utilizador
	Users     []string  `json:"users,omitempty"`   // Lista de utilizadores ativos
	History   []Message `json:"history,omitempty"` // Histórico recente de mensagens
}

const maxHistory = 50

// Hub mantém o conjunto de clientes ativos e faz a transmissão de mensagens.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	history    []Message
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		history:    make([]Message, 0, maxHistory),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			// Fazer cópia do histórico atual para enviar ao novo utilizador
			historyCopy := make([]Message, len(h.history))
			copy(historyCopy, h.history)
			h.mu.Unlock()

			// Enviar histórico recente diretamente ao novo cliente
			if len(historyCopy) > 0 {
				select {
				case client.send <- Message{Type: "history", History: historyCopy}:
				default:
				}
			}

			// Criar notificação de entrada
			joinMsg := Message{
				Type:      "join",
				Sender:    client.username,
				Color:     client.color,
				Timestamp: time.Now().Format("15:04"),
			}

			// Guardar no histórico e transmitir a todos
			h.appendToHistory(joinMsg)
			h.broadcastToAll(joinMsg)

			// Enviar lista atualizada de utilizadores
			h.sendUserList()

		case client := <-h.unregister:
			username := client.username
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

			// Criar notificação de saída
			leaveMsg := Message{
				Type:      "leave",
				Sender:    username,
				Timestamp: time.Now().Format("15:04"),
			}

			// Guardar no histórico e transmitir a todos
			h.appendToHistory(leaveMsg)
			h.broadcastToAll(leaveMsg)

			// Enviar lista atualizada de utilizadores
			h.sendUserList()

		case message := <-h.broadcast:
			if message.Type == "clear_history" {
				h.mu.Lock()
				h.history = make([]Message, 0, maxHistory)
				h.mu.Unlock()

				clearedMsg := Message{
					Type:      "history_cleared",
					Sender:    message.Sender,
					Timestamp: time.Now().Format("15:04"),
				}
				h.broadcastToAll(clearedMsg)
			} else {
				h.appendToHistory(message)
				h.broadcastToAll(message)
			}
		}
	}
}

func (h *Hub) appendToHistory(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.history = append(h.history, msg)
	if len(h.history) > maxHistory {
		h.history = h.history[len(h.history)-maxHistory:]
	}
}

func (h *Hub) broadcastToAll(message Message) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) sendUserList() {
	h.mu.RLock()
	users := make([]string, 0, len(h.clients))
	for client := range h.clients {
		users = append(users, client.username)
	}
	h.mu.RUnlock()

	userListMsg := Message{
		Type:  "user_list",
		Users: users,
	}

	h.broadcastToAll(userListMsg)
}
