package main

import (
	"sync"
	"time"
)

// Message representa a estrutura de dados trocada via WebSocket.
type Message struct {
	Type      string   `json:"type"`              // "message", "join", "leave", "user_list"
	Sender    string   `json:"sender,omitempty"`  // Nome do utilizador
	Content   string   `json:"content,omitempty"` // Texto da mensagem
	Timestamp string   `json:"timestamp,omitempty"`
	Color     string   `json:"color,omitempty"`   // Cor única atribuída ao utilizador
	Users     []string `json:"users,omitempty"`   // Lista de utilizadores ativos
}

// Hub mantém o conjunto de clientes ativos e faz a transmissão de mensagens.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			// Notificar todos que um novo utilizador entrou
			h.broadcastToAll(Message{
				Type:      "join",
				Sender:    client.username,
				Color:     client.color,
				Timestamp: time.Now().Format("15:04"),
			})

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

			// Notificar que o utilizador saiu (sem segurar o mutex para evitar deadlock)
			h.broadcastToAll(Message{
				Type:      "leave",
				Sender:    username,
				Timestamp: time.Now().Format("15:04"),
			})

			// Enviar lista atualizada de utilizadores
			h.sendUserList()

		case message := <-h.broadcast:
			h.broadcastToAll(message)
		}
	}
}

func (h *Hub) broadcastToAll(message Message) {
	h.mu.RLock()
	// Criar cópia da lista de clientes para iterar com segurança
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			// Se o canal do cliente estiver cheio ou fechado, remove com segurança
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
