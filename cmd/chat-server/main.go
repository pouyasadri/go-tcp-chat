package main

import (
	"log"
	"net"

	"github.com/pouyasadri/go-tcp-chat/internal/chat"
)

func main() {
	s := chat.NewServer()
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
