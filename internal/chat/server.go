package chat

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pouyasadri/go-tcp-chat/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	rooms    map[string]*Room
	commands chan Command
	store    persistence
}

type persistence interface {
	store.UserStore
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
		case CMDRegister:
			s.Register(cmd.Client, cmd.Args)
		case CMDLogin:
			s.Login(cmd.Client, cmd.Args)
		case CMDLogout:
			s.Logout(cmd.Client)
		case CMDWhoAmI:
			s.WhoAmI(cmd.Client)
		case CMDHistory:
			s.History(cmd.Client, cmd.Args)
		case CMDQuit:
			s.Quit(cmd.Client)
		}
	}
}

func (s *Server) Register(c *Client, args []string) {
	if s.store == nil {
		c.Err(fmt.Errorf("auth is unavailable"))
		return
	}
	if len(args) < 3 {
		c.Err(fmt.Errorf("usage: /register <username> <password>"))
		return
	}

	username := normalizeUsername(args[1])
	password := args[2]
	if err := validateCredentials(username, password); err != nil {
		c.Err(err)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.Err(fmt.Errorf("failed to create account"))
		return
	}

	user, err := s.store.CreateUser(context.Background(), username, string(hashed))
	if err != nil {
		if err == store.ErrUserExists {
			c.Err(fmt.Errorf("username is already taken"))
			return
		}
		c.Err(fmt.Errorf("failed to create account"))
		return
	}

	c.UserID = &user.ID
	c.NickName = user.Username
	c.Msg(fmt.Sprintf("Registered and logged in as %s", user.Username))
}

func (s *Server) Login(c *Client, args []string) {
	if s.store == nil {
		c.Err(fmt.Errorf("auth is unavailable"))
		return
	}
	if len(args) < 3 {
		c.Err(fmt.Errorf("usage: /login <username> <password>"))
		return
	}

	username := normalizeUsername(args[1])
	password := args[2]

	user, err := s.store.GetUserByUsername(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Err(fmt.Errorf("invalid username or password"))
			return
		}
		c.Err(fmt.Errorf("failed to login"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.Err(fmt.Errorf("invalid username or password"))
		return
	}

	c.UserID = &user.ID
	c.NickName = user.Username
	c.Msg(fmt.Sprintf("Logged in as %s", user.Username))
}

func (s *Server) Logout(c *Client) {
	if c.UserID == nil {
		c.Msg("You are not logged in")
		return
	}

	c.UserID = nil
	c.NickName = "anonymous"
	c.Msg("Logged out")
}

func (s *Server) WhoAmI(c *Client) {
	if c.UserID == nil {
		c.Msg("You are anonymous")
		return
	}

	c.Msg(fmt.Sprintf("You are logged in as %s (id=%d)", c.NickName, *c.UserID))
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
	s.printRecentHistory(c, 20)
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
		_, err := s.store.SaveMessage(context.Background(), c.Room.ID, c.UserID, c.NickName, msg)
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
	if c.UserID == nil {
		c.Err(fmt.Errorf("you must login before sending direct messages"))
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

func (s *Server) History(c *Client, args []string) {
	if s.store == nil {
		c.Err(fmt.Errorf("history is unavailable"))
		return
	}
	if c.Room == nil {
		c.Err(fmt.Errorf("you must join a room before requesting history"))
		return
	}

	limit, beforeID, err := parseHistoryArgs(args)
	if err != nil {
		c.Err(err)
		return
	}

	var messages []store.Message
	if beforeID == nil {
		messages, err = s.store.ListRoomMessages(context.Background(), c.Room.ID, limit)
	} else {
		messages, err = s.store.ListRoomMessagesBefore(context.Background(), c.Room.ID, *beforeID, limit)
	}
	if err != nil {
		c.Err(fmt.Errorf("failed to read history"))
		return
	}

	s.printMessages(c, messages)
}

func parseHistoryArgs(args []string) (int, *int64, error) {
	const defaultLimit = 20

	if len(args) == 1 {
		return defaultLimit, nil, nil
	}

	if args[1] == "before" {
		if len(args) < 3 {
			return 0, nil, fmt.Errorf("usage: /history [n] or /history before <id> [n]")
		}
		beforeID, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil || beforeID <= 0 {
			return 0, nil, fmt.Errorf("invalid message id for /history before")
		}

		limit := defaultLimit
		if len(args) >= 4 {
			parsedLimit, err := strconv.Atoi(args[3])
			if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
				return 0, nil, fmt.Errorf("history limit must be between 1 and 100")
			}
			limit = parsedLimit
		}

		return limit, &beforeID, nil
	}

	limit, err := strconv.Atoi(args[1])
	if err != nil || limit <= 0 || limit > 100 {
		return 0, nil, fmt.Errorf("history limit must be between 1 and 100")
	}

	return limit, nil, nil
}

func (s *Server) printRecentHistory(c *Client, limit int) {
	if s.store == nil || c.Room == nil {
		return
	}

	messages, err := s.store.ListRoomMessages(context.Background(), c.Room.ID, limit)
	if err != nil || len(messages) == 0 {
		return
	}

	c.Msg("Recent history:")
	s.printMessages(c, messages)
}

func (s *Server) printMessages(c *Client, messages []store.Message) {
	if len(messages) == 0 {
		c.Msg("No message history found")
		return
	}

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ts := msg.CreatedAt.Format(time.RFC3339)
		c.Msg(fmt.Sprintf("[#%d %s] %s: %s", msg.ID, ts, msg.SenderNick, msg.Body))
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validateCredentials(username, password string) error {
	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("username must be 3-32 characters")
	}
	if strings.Contains(username, " ") {
		return fmt.Errorf("username cannot contain spaces")
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
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
