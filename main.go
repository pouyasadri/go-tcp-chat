package main

import (
	"log"
	"net"
)

func main() {

	s := NewServer()
	go s.Run()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Unable to start server: ", err.Error())
	}
	defer listener.Close()
	log.Printf("Server started on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %s", err.Error())
			continue
		}
		go s.NewClient(conn)
	}
}
