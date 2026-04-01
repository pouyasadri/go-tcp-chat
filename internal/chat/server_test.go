package chat

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"
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

	sender := &Client{Conn: senderConn, NickName: "alice"}
	receiver := &Client{Conn: receiverConn, NickName: "bob"}

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

	s.DM(c, []string{"/dm", "anonymous", "self msg"})
	if !strings.Contains(conn.String(), "cannot send direct message to yourself") {
		t.Fatalf("expected self dm error, got %q", conn.String())
	}

	conn.Reset()
	s.DM(c, []string{"/dm", "nobody", "hi"})
	if !strings.Contains(conn.String(), "is not in room") {
		t.Fatalf("expected not found error, got %q", conn.String())
	}

	conn.Reset()
	s.DM(c, []string{"/dm", "nobody"})
	if !strings.Contains(conn.String(), "usage: /dm <nick> <message>") {
		t.Fatalf("expected usage error, got %q", conn.String())
	}
}
