package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	hub := newHub()
	go hub.run()

	// Extrai a subpasta static dos ficheiros embebidos no próprio binário
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Erro ao carregar ficheiros estáticos embebidos: %v", err)
	}

	// Servidor de ficheiros estáticos (agora embutidos 100% no binário Go)
	http.Handle("/", http.FileServer(http.FS(subFS)))

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
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
