package chat

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"
	"strings"

	"github.com/pouyasadri/go-tcp-chat/internal/store"
)

type Server struct {
	rooms    map[string]*Room
	commands chan Command
	store    persistence
}

type persistence interface {
	store.RoomStore
	store.MessageStore
}

func NewServer(p persistence) *Server {
	return &Server{
		rooms:    make(map[string]*Room),
		commands: make(chan Command),
		store:    p,
	}
}

func (s *Server) Run() {
	for cmd := range s.commands {
		switch cmd.ID {
		case CMDHelp:
			cmd.Client.printHelp()
		case CMDNick:
			s.Nick(cmd.Client, cmd.Args)
		case CMDJoin:
			s.Join(cmd.Client, cmd.Args)
		case CMDRooms:
			s.ListRooms(cmd.Client)
		case CMDMsg:
			s.Msg(cmd.Client, cmd.Args)
		case CMDDM:
			s.DM(cmd.Client, cmd.Args)
		case CMDQuit:
			s.Quit(cmd.Client)
		}
	}
}

func (s *Server) NewClient(conn net.Conn) {
	log.Printf("new client connected: %s", conn.RemoteAddr())

	c := &Client{
		Conn:     conn,
		NickName: "anonymous",
		Commands: s.commands,
	}
	c.Msg("Welcome to go-tcp-chat. Type /help to get started.")
	c.ReadInput()
}

func (s *Server) Nick(c *Client, args []string) {
	if len(args) < 2 {
		c.Err(fmt.Errorf("usage: /nick <name>"))
		return
	}

	c.NickName = args[1]
	c.Msg(fmt.Sprintf("Your nickname is now: %s", c.NickName))
}

func (s *Server) Join(c *Client, args []string) {
	if len(args) < 2 {
		c.Err(fmt.Errorf("usage: /join <room>"))
		return
	}

	roomName := args[1]
	if c.Room != nil && c.Room.Name == roomName {
		c.Msg(fmt.Sprintf("You are already in room: %s", roomName))
		return
	}

	s.QuitCurrentRoom(c)

	r, ok := s.rooms[roomName]
	if !ok {
		r = &Room{Name: roomName, Members: make(map[net.Addr]*Client)}
		if s.store != nil {
			persisted, err := s.store.FindOrCreateRoom(context.Background(), roomName)
			if err != nil {
				c.Err(fmt.Errorf("failed to join room: %w", err))
				return
			}
			r.ID = persisted.ID
		}
		s.rooms[roomName] = r
	} else if s.store != nil && r.ID == 0 {
		persisted, err := s.store.FindOrCreateRoom(context.Background(), roomName)
		if err != nil {
			c.Err(fmt.Errorf("failed to join room: %w", err))
			return
		}
		r.ID = persisted.ID
	}

	r.Members[c.Conn.RemoteAddr()] = c
	c.Room = r
	c.Room.Broadcast(c, fmt.Sprintf("%s has joined the room", c.NickName))
	c.Msg(fmt.Sprintf("Welcome to %s", roomName))
}

func (s *Server) ListRooms(c *Client) {
	if s.store != nil {
		rooms, err := s.store.ListRooms(context.Background())
		if err != nil {
			c.Err(fmt.Errorf("failed to list rooms: %w", err))
			return
		}
		if len(rooms) == 0 {
			c.Msg("No rooms available yet. Create one with /join <room>.")
			return
		}

		names := make([]string, 0, len(rooms))
		for _, room := range rooms {
			names = append(names, room.Name)
		}
		c.Msg(fmt.Sprintf("Available rooms: %s", strings.Join(names, ", ")))
		return
	}

	if len(s.rooms) == 0 {
		c.Msg("No rooms available yet. Create one with /join <room>.")
		return
	}

	rooms := make([]string, 0, len(s.rooms))
	for name := range s.rooms {
		rooms = append(rooms, name)
	}
	slices.Sort(rooms)

	c.Msg(fmt.Sprintf("Available rooms: %s", strings.Join(rooms, ", ")))
}

func (s *Server) Msg(c *Client, args []string) {
	if c.Room == nil {
		c.Err(fmt.Errorf("you must join a room before sending messages"))
		return
	}
	if len(args) < 2 {
		c.Err(fmt.Errorf("usage: /msg <message>"))
		return
	}

	msg := strings.Join(args[1:], " ")
	if s.store != nil {
		_, err := s.store.SaveMessage(context.Background(), c.Room.ID, nil, c.NickName, msg)
		if err != nil {
			c.Err(fmt.Errorf("failed to persist message: %w", err))
			return
		}
	}
	c.Room.Broadcast(c, fmt.Sprintf("%s: %s", c.NickName, msg))
}

func (s *Server) DM(c *Client, args []string) {
	if c.Room == nil {
		c.Err(fmt.Errorf("you must join a room before sending direct messages"))
		return
	}
	if len(args) < 3 {
		c.Err(fmt.Errorf("usage: /dm <nick> <message>"))
		return
	}

	targetNick := args[1]
	if targetNick == c.NickName {
		c.Err(fmt.Errorf("cannot send direct message to yourself"))
		return
	}

	var recipient *Client
	for _, member := range c.Room.Members {
		if member.NickName == targetNick {
			recipient = member
			break
		}
	}

	if recipient == nil {
		c.Err(fmt.Errorf("user %q is not in room %q", targetNick, c.Room.Name))
		return
	}

	message := strings.Join(args[2:], " ")
	recipient.Msg(fmt.Sprintf("[DM from %s] %s", c.NickName, message))
	c.Msg(fmt.Sprintf("[DM to %s] %s", targetNick, message))
}

func (s *Server) Quit(c *Client) {
	log.Printf("client disconnected: %s", c.Conn.RemoteAddr())
	s.QuitCurrentRoom(c)
	c.Msg("Goodbye!")
	_ = c.Conn.Close()
}

func (s *Server) QuitCurrentRoom(c *Client) {
	if c.Room == nil {
		return
	}

	delete(c.Room.Members, c.Conn.RemoteAddr())
	c.Room.Broadcast(c, fmt.Sprintf("%s has left the room", c.NickName))

	if len(c.Room.Members) == 0 {
		delete(s.rooms, c.Room.Name)
	}
	c.Room = nil
}
