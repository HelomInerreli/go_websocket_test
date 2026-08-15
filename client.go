package main

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Tempo permitido para escrever uma mensagem no peer.
	writeWait = 10 * time.Second

	// Tempo permitido para ler a próxima mensagem pong do peer.
	pongWait = 60 * time.Second

	// Enviar pings ao peer com este período (deve ser menor que pongWait).
	pingPeriod = (pongWait * 9) / 10

	// Tamanho máximo da mensagem permitida pelo peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Permite conexões de qualquer origem para facilitar testes locais
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client é o intermediário entre a conexão websocket e o hub.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan Message
	username string
	color    string
}

// generateUserColor cria uma cor viva em HSL/Hex determinística com base no nome do utilizador.
func generateUserColor(username string) string {
	colors := []string{
		"#10B981", "#3B82F6", "#8B5CF6", "#EC4899",
		"#F59E0B", "#06B6D4", "#6366F1", "#F43F5E",
		"#14B8A6", "#84CC16", "#D97706", "#A855F7",
	}

	hash := md5.Sum([]byte(strings.ToLower(username)))
	hashStr := hex.EncodeToString(hash[:])
	index := int(hashStr[0]) % len(colors)
	return colors[index]
}

// readPump transfere mensagens do websocket para o hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("erro ao ler websocket: %v", err)
			}
			break
		}

		// Sanitizar e processar mensagem
		content := strings.TrimSpace(msg.Content)
		if content != "" {
			msg.Type = "message"
			msg.Sender = c.username
			msg.Color = c.color
			msg.Timestamp = time.Now().Format("15:04")
			msg.Content = content

			c.hub.broadcast <- msg
		}
	}
}

// writePump transfere mensagens do hub para a conexão websocket.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// O hub fechou o canal.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs trata os pedidos de upgrade para WebSocket vindos do cliente HTTP.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "Nome de utilizador é obrigatório", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro no upgrade para websocket:", err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan Message, 256),
		username: username,
		color:    generateUserColor(username),
	}

	client.hub.register <- client

	// Executa leitura e escrita em goroutines dedicadas
	go client.writePump()
	go client.readPump()
}
