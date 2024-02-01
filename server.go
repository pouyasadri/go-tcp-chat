package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type Server struct {
	rooms    map[string]*Room
	commands chan Command
}

func NewServer() *Server {
	return &Server{
		rooms:    make(map[string]*Room),
		commands: make(chan Command),
	}
}
func (s *Server) Run() {
	for cmd := range s.commands {
		switch cmd.ID {
		case CMD_NICK:
			s.Nick(cmd.Client, cmd.Args)
		case CMD_JOIN:
			s.Join(cmd.Client, cmd.Args)
		case CMD_ROOMS:
			s.ListRooms(cmd.Client)
		case CMD_MSG:
			s.Msg(cmd.Client, cmd.Args)
		case CMD_QUIT:
			s.Quit(cmd.Client)
		}
	}
}
func (s *Server) NewClient(conn net.Conn) {
	log.Printf("New client has connected: %s", conn.RemoteAddr().String())

	c := &Client{
		Conn:     conn,
		NickName: "anonymous",
		Commands: s.commands,
	}
	c.ReadInput()
}

func (s *Server) Nick(c *Client, args []string) {
	c.NickName = args[1]
	c.Msg(fmt.Sprintf("Your nickname has been updated to: %s\n", c.NickName))
}

func (s *Server) Join(c *Client, args []string) {
	if len(args) < 2 {
		c.Msg("Please provide a room name to join\n")
		return
	}
	roomName := args[1]
	r, ok := s.rooms[roomName]
	if !ok {
		r = &Room{
			Name:    roomName,
			Members: make(map[net.Addr]*Client),
		}
		s.rooms[roomName] = r
	}
	r.Members[c.Conn.RemoteAddr()] = c

	s.QuitCurrentRoom(c)

	c.Room = r
	c.Room.Broadcast(c, fmt.Sprintf("%s has joined the room\n", c.NickName))
	c.Msg(fmt.Sprintf("Welcome to %s\n", roomName))
}

func (s *Server) ListRooms(c *Client) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}
	c.Msg(fmt.Sprintf("Available rooms: %s\n", strings.Join(rooms, ", ")))
}
func (s *Server) Msg(c *Client, args []string) {
	if c.Room == nil {
		c.Err(fmt.Errorf("you must join a room before you can send messages"))
		return
	}
	msg := strings.Join(args[1:], " ")
	c.Room.Broadcast(c, fmt.Sprintf("%s: %s\n", c.NickName, msg))
}

func (s *Server) Quit(c *Client) {
	log.Printf("Client has disconnected: %s", c.Conn.RemoteAddr().String())
	s.QuitCurrentRoom(c)
	c.Msg("Goodbye!")
	c.Conn.Close()
}

func (s *Server) QuitCurrentRoom(c *Client) {
	if c.Room != nil {
		delete(c.Room.Members, c.Conn.RemoteAddr())
		c.Room.Broadcast(c, fmt.Sprintf("%s has left the room\n", c.NickName))
	}
}
