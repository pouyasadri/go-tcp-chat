package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pouyasadri/go-tcp-chat/internal/store"
)

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (store.User, error) {
	result, err := s.db.ExecContext(
		ctx,
		"INSERT INTO users(username, password_hash) VALUES (?, ?)",
		username,
		passwordHash,
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return store.User{}, store.ErrUserExists
		}
		return store.User{}, fmt.Errorf("create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.User{}, fmt.Errorf("create user last insert id: %w", err)
	}

	return s.getUserByID(ctx, id)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (store.User, error) {
	var user store.User
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.User{}, sql.ErrNoRows
		}
		return store.User{}, fmt.Errorf("get user by username: %w", err)
	}

	return user, nil
}

func (s *Store) FindOrCreateRoom(ctx context.Context, name string) (store.Room, error) {
	result, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO rooms(name) VALUES (?)", name)
	if err != nil {
		return store.Room{}, fmt.Errorf("find or create room: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Room{}, fmt.Errorf("find or create room last insert id: %w", err)
	}

	if id > 0 {
		return s.getRoomByID(ctx, id)
	}

	var room store.Room
	err = s.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM rooms WHERE name = ?", name).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err != nil {
		return store.Room{}, fmt.Errorf("find existing room by name: %w", err)
	}

	return room, nil
}

func (s *Store) ListRooms(ctx context.Context) ([]store.Room, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, created_at FROM rooms ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	rooms := make([]store.Room, 0)
	for rows.Next() {
		var room store.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rooms: %w", err)
	}

	return rooms, nil
}

func (s *Store) SaveMessage(ctx context.Context, roomID int64, senderUserID *int64, senderNick, body string) (store.Message, error) {
	result, err := s.db.ExecContext(
		ctx,
		"INSERT INTO messages(room_id, sender_user_id, sender_nick, body) VALUES (?, ?, ?, ?)",
		roomID,
		senderUserID,
		senderNick,
		body,
	)
	if err != nil {
		return store.Message{}, fmt.Errorf("save message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Message{}, fmt.Errorf("save message last insert id: %w", err)
	}

	return s.getMessageByID(ctx, id)
}

func (s *Store) ListRoomMessages(ctx context.Context, roomID int64, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, room_id, sender_user_id, sender_nick, body, created_at
		 FROM messages
		 WHERE room_id = ?
		 ORDER BY id DESC
		 LIMIT ?`,
		roomID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list room messages: %w", err)
	}
	defer rows.Close()

	messages := make([]store.Message, 0)
	for rows.Next() {
		var msg store.Message
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderUserID, &msg.SenderNick, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}

func (s *Store) ListRoomMessagesBefore(ctx context.Context, roomID, beforeID int64, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, room_id, sender_user_id, sender_nick, body, created_at
		 FROM messages
		 WHERE room_id = ? AND id < ?
		 ORDER BY id DESC
		 LIMIT ?`,
		roomID,
		beforeID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list room messages before: %w", err)
	}
	defer rows.Close()

	messages := make([]store.Message, 0)
	for rows.Next() {
		var msg store.Message
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderUserID, &msg.SenderNick, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message before: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages before: %w", err)
	}

	return messages, nil
}

func (s *Store) getUserByID(ctx context.Context, id int64) (store.User, error) {
	var user store.User
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return store.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *Store) getRoomByID(ctx context.Context, id int64) (store.Room, error) {
	var room store.Room
	err := s.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM rooms WHERE id = ?", id).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err != nil {
		return store.Room{}, fmt.Errorf("get room by id: %w", err)
	}

	return room, nil
}

func (s *Store) getMessageByID(ctx context.Context, id int64) (store.Message, error) {
	var msg store.Message
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, room_id, sender_user_id, sender_nick, body, created_at FROM messages WHERE id = ?",
		id,
	).Scan(&msg.ID, &msg.RoomID, &msg.SenderUserID, &msg.SenderNick, &msg.Body, &msg.CreatedAt)
	if err != nil {
		return store.Message{}, fmt.Errorf("get message by id: %w", err)
	}

	return msg, nil
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
