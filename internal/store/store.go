package store

import (
	"context"
	"errors"
	"time"
)

var ErrUserExists = errors.New("user already exists")

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Room struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type Message struct {
	ID           int64
	RoomID       int64
	SenderUserID *int64
	SenderNick   string
	Body         string
	CreatedAt    time.Time
}

type UserStore interface {
	CreateUser(ctx context.Context, username, passwordHash string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
}

type RoomStore interface {
	FindOrCreateRoom(ctx context.Context, name string) (Room, error)
	ListRooms(ctx context.Context) ([]Room, error)
}

type MessageStore interface {
	SaveMessage(ctx context.Context, roomID int64, senderUserID *int64, senderNick, body string) (Message, error)
	ListRoomMessages(ctx context.Context, roomID int64, limit int) ([]Message, error)
}
