package chat

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pouyasadri/go-tcp-chat/internal/store"
)

func newTestClient() (*Client, *stubConn) {
	conn := &stubConn{remote: stubAddr("test-remote")}
	c := &Client{
		Conn:     conn,
		NickName: "anonymous",
	}
	return c, conn
}

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

type stubConn struct {
	bytes.Buffer
	remote net.Addr
}

func (s *stubConn) Read(_ []byte) (int, error)  { return 0, io.EOF }
func (s *stubConn) Write(p []byte) (int, error) { return s.Buffer.Write(p) }
func (s *stubConn) Close() error                { return nil }
func (s *stubConn) LocalAddr() net.Addr         { return stubAddr("local") }
func (s *stubConn) RemoteAddr() net.Addr        { return s.remote }
func (s *stubConn) SetDeadline(_ time.Time) error {
	return nil
}
func (s *stubConn) SetReadDeadline(_ time.Time) error {
	return nil
}
func (s *stubConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

func TestNickValidationAndUpdate(t *testing.T) {
	s := NewServer(nil)
	c, conn := newTestClient()

	s.Nick(c, []string{"/nick"})
	line := conn.String()
	if !strings.Contains(line, "usage: /nick <name>") {
		t.Fatalf("expected usage error, got %q", line)
	}

	conn.Reset()
	s.Nick(c, []string{"/nick", "pouya"})
	if c.NickName != "pouya" {
		t.Fatalf("nickname not updated, got %q", c.NickName)
	}
}

func TestJoinSameRoomNoDuplicate(t *testing.T) {
	s := NewServer(nil)
	c, conn := newTestClient()

	s.Join(c, []string{"/join", "general"})
	welcome := conn.String()
	if !strings.Contains(welcome, "Welcome to general") {
		t.Fatalf("unexpected first join response: %q", welcome)
	}

	conn.Reset()
	s.Join(c, []string{"/join", "general"})
	line := conn.String()
	if !strings.Contains(line, "already in room") {
		t.Fatalf("expected already in room warning, got %q", line)
	}
}

func TestQuitCurrentRoomRemovesEmptyRoom(t *testing.T) {
	s := NewServer(nil)
	c, _ := newTestClient()

	s.Join(c, []string{"/join", "general"})
	if len(s.rooms) != 1 {
		t.Fatalf("expected one room, got %d", len(s.rooms))
	}

	s.QuitCurrentRoom(c)
	if len(s.rooms) != 0 {
		t.Fatalf("expected room cleanup, got %d rooms", len(s.rooms))
	}
	if c.Room != nil {
		t.Fatalf("expected client room to be nil")
	}
}

func TestDMSendsToRecipientAndSenderConfirmation(t *testing.T) {
	s := NewServer(nil)

	senderConn := &stubConn{remote: stubAddr("sender")}
	receiverConn := &stubConn{remote: stubAddr("receiver")}
	senderID := int64(1)
	receiverID := int64(2)

	sender := &Client{Conn: senderConn, NickName: "alice", UserID: &senderID}
	receiver := &Client{Conn: receiverConn, NickName: "bob", UserID: &receiverID}

	room := &Room{Name: "general", Members: map[net.Addr]*Client{sender.Conn.RemoteAddr(): sender, receiver.Conn.RemoteAddr(): receiver}}
	s.rooms[room.Name] = room
	sender.Room = room
	receiver.Room = room

	s.DM(sender, []string{"/dm", "bob", "hello", "there"})

	received := receiverConn.String()
	if !strings.Contains(received, "[DM from alice] hello there") {
		t.Fatalf("expected receiver DM, got %q", received)
	}

	sent := senderConn.String()
	if !strings.Contains(sent, "[DM to bob] hello there") {
		t.Fatalf("expected sender confirmation, got %q", sent)
	}
}

func TestDMValidation(t *testing.T) {
	s := NewServer(nil)
	c, conn := newTestClient()

	s.DM(c, []string{"/dm", "bob", "hi"})
	if !strings.Contains(conn.String(), "join a room") {
		t.Fatalf("expected room validation error, got %q", conn.String())
	}

	conn.Reset()
	room := &Room{Name: "general", Members: map[net.Addr]*Client{c.Conn.RemoteAddr(): c}}
	s.rooms[room.Name] = room
	c.Room = room
	userID := int64(1)
	c.UserID = &userID

	s.DM(c, []string{"/dm", "anonymous", "self msg"})
	if !strings.Contains(conn.String(), "cannot send direct message to yourself") {
		t.Fatalf("expected self dm error, got %q", conn.String())
	}

	conn.Reset()
	c.UserID = nil
	otherConn := &stubConn{remote: stubAddr("other")}
	otherUserID := int64(2)
	room.Members[otherConn.RemoteAddr()] = &Client{Conn: otherConn, NickName: "friend", UserID: &otherUserID}
	s.DM(c, []string{"/dm", "friend", "hi"})
	if !strings.Contains(conn.String(), "must login") {
		t.Fatalf("expected auth error, got %q", conn.String())
	}

	conn.Reset()
	c.UserID = &userID
	s.DM(c, []string{"/dm", "nobody", "hi"})
	if !strings.Contains(conn.String(), "is not in room") {
		t.Fatalf("expected not found error, got %q", conn.String())
	}

	conn.Reset()
	c.UserID = nil
	s.DM(c, []string{"/dm", "nobody"})
	if !strings.Contains(conn.String(), "usage: /dm <nick> <message>") {
		t.Fatalf("expected usage error, got %q", conn.String())
	}
}

type authStore struct {
	nextUserID int64
	users      map[string]store.User
}

func newAuthStore() *authStore {
	return &authStore{nextUserID: 1, users: make(map[string]store.User)}
}

func (a *authStore) CreateUser(_ context.Context, username, passwordHash string) (store.User, error) {
	if _, ok := a.users[username]; ok {
		return store.User{}, store.ErrUserExists
	}
	user := store.User{ID: a.nextUserID, Username: username, PasswordHash: passwordHash}
	a.nextUserID++
	a.users[username] = user
	return user, nil
}

func (a *authStore) GetUserByUsername(_ context.Context, username string) (store.User, error) {
	user, ok := a.users[username]
	if !ok {
		return store.User{}, sql.ErrNoRows
	}
	return user, nil
}

func (a *authStore) FindOrCreateRoom(_ context.Context, name string) (store.Room, error) {
	return store.Room{Name: name, ID: 1}, nil
}

func (a *authStore) ListRooms(_ context.Context) ([]store.Room, error) {
	return []store.Room{{ID: 1, Name: "general"}}, nil
}

func (a *authStore) SaveMessage(_ context.Context, roomID int64, senderUserID *int64, senderNick, body string) (store.Message, error) {
	return store.Message{RoomID: roomID, SenderUserID: senderUserID, SenderNick: senderNick, Body: body}, nil
}

func (a *authStore) ListRoomMessages(_ context.Context, roomID int64, limit int) ([]store.Message, error) {
	return nil, nil
}

func TestRegisterLoginLogoutWhoAmI(t *testing.T) {
	a := newAuthStore()
	s := NewServer(a)
	c, conn := newTestClient()

	s.Register(c, []string{"/register", "Alice", "password123"})
	if c.UserID == nil || c.NickName != "alice" {
		t.Fatalf("expected user to be registered and logged in")
	}
	if !strings.Contains(conn.String(), "Registered and logged in as alice") {
		t.Fatalf("unexpected register output: %q", conn.String())
	}

	conn.Reset()
	s.Logout(c)
	if c.UserID != nil || c.NickName != "anonymous" {
		t.Fatalf("expected user to be logged out")
	}

	conn.Reset()
	s.Login(c, []string{"/login", "alice", "password123"})
	if c.UserID == nil || c.NickName != "alice" {
		t.Fatalf("expected user to be logged in")
	}

	conn.Reset()
	s.WhoAmI(c)
	if !strings.Contains(conn.String(), "logged in as alice") {
		t.Fatalf("unexpected whoami output: %q", conn.String())
	}
}
