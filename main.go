package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	hub := newHub()
	go hub.run()

	// Servidor de ficheiros estáticos para a interface HTML/CSS/JS
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Endpoint WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("Servidor de Chat WebSocket a rodar na porta %s", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
