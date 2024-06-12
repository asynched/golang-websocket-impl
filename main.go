package main

import (
	"log"
	"net/http"

	"github.com/asynched/golang-websocket-impl/internal/ws"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.Lmsgprefix)
	log.SetPrefix("[websocket] ")
}

func main() {
	router := http.NewServeMux()

	router.HandleFunc("GET /ws/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.Upgrade(w, r)

		if err != nil {
			log.Printf("Failed to upgrade connection: %v\n", err)

			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Client connected: %v\n", r.RemoteAddr)

		defer conn.Close()

		for {
			buffer := make([]byte, 512)

			n, err := conn.Read(buffer)

			if err != nil {
				log.Printf("Client disconnected: %v\n", err)
				break
			}

			log.Printf("Read %d bytes from client\n", n)

			log.Print(string(buffer[:n]))
			conn.Write(buffer[:n])
		}
	})

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
