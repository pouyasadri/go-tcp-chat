package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "chat.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return store
}

func TestMigrateIsIdempotent(t *testing.T) {
	store := newTestStore(t)

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate second run: %v", err)
	}
}

func TestFindOrCreateRoom(t *testing.T) {
	store := newTestStore(t)

	roomA, err := store.FindOrCreateRoom(context.Background(), "general")
	if err != nil {
		t.Fatalf("first find/create: %v", err)
	}
	roomB, err := store.FindOrCreateRoom(context.Background(), "general")
	if err != nil {
		t.Fatalf("second find/create: %v", err)
	}

	if roomA.ID != roomB.ID {
		t.Fatalf("expected same room id, got %d and %d", roomA.ID, roomB.ID)
	}
}

func TestSaveAndListRoomMessages(t *testing.T) {
	store := newTestStore(t)

	room, err := store.FindOrCreateRoom(context.Background(), "general")
	if err != nil {
		t.Fatalf("find/create room: %v", err)
	}

	_, err = store.SaveMessage(context.Background(), room.ID, nil, "alice", "hello")
	if err != nil {
		t.Fatalf("save message 1: %v", err)
	}
	_, err = store.SaveMessage(context.Background(), room.ID, nil, "bob", "world")
	if err != nil {
		t.Fatalf("save message 2: %v", err)
	}

	messages, err := store.ListRoomMessages(context.Background(), room.ID, 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Body != "world" {
		t.Fatalf("expected newest message first, got %q", messages[0].Body)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	store := newTestStore(t)

	_, err := store.CreateUser(context.Background(), "pouya", "hashed-password")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	user, err := store.GetUserByUsername(context.Background(), "pouya")
	if err != nil {
		t.Fatalf("get user by username: %v", err)
	}
	if user.Username != "pouya" {
		t.Fatalf("unexpected username: %q", user.Username)
	}

	_, err = store.GetUserByUsername(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected sql.ErrNoRows for missing user")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
