package main

import (
	"context"
	"log"
	"net"

	"github.com/pouyasadri/go-tcp-chat/internal/chat"
	"github.com/pouyasadri/go-tcp-chat/internal/store/sqlite"
)

func main() {
	store, err := sqlite.Open("chat.db")
	if err != nil {
		log.Fatal("unable to open sqlite store: ", err)
	}
	defer store.Close()

	if err := store.Migrate(context.Background()); err != nil {
		log.Fatal("unable to run migrations: ", err)
	}

	s := chat.NewServer(store)
	go s.Run()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("unable to start server: ", err)
	}
	defer listener.Close()

	log.Printf("server started on :8080")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v", err)
			continue
		}
		go s.NewClient(conn)
	}
}
