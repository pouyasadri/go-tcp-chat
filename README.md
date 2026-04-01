# Go TCP Chat

[![CI](https://github.com/pouyasadri/go-tcp-chat/actions/workflows/ci.yml/badge.svg)](https://github.com/pouyasadri/go-tcp-chat/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/github/go-mod/go-version/pouyasadri/go-tcp-chat)
![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fpouyasadri%2Fgo--tcp--chat-blue?logo=github)

A lightweight TCP chat server written in Go. It supports multi-room messaging, direct messages, account authentication, and persisted room history over raw TCP.

## Why this project

This project demonstrates:

- Concurrent network programming with Go (`net`, goroutines, channels)
- Event-loop style command handling in the server
- Room-based message broadcast model
- Authentication flow (`/register`, `/login`, `/logout`, `/whoami`)
- Persisted room history with pagination (`/history`)
- Direct messaging inside a room via `/dm`
- Defensive input handling and deterministic output behavior

## Tech stack

- Go 1.26
- SQLite persistence (`modernc.org/sqlite`)
- Password hashing with `bcrypt` (`golang.org/x/crypto`)
- GitHub Actions for CI + GHCR image publishing

## Architecture

```mermaid
flowchart LR
    A[Client A - nc] --> L[TCP Listener :8080]
    B[Client B - nc] --> L
    L --> S[Server command loop]
    S --> Auth[Auth commands]
    S --> R1[Room: general]
    S --> R2[Room: random]
    S --> H[/history command]
    S --> D[/dm command]
    S --> DB[(SQLite)]
    R1 --> O[Broadcast to local room members]
    H --> DB
    D --> O
```

## Project structure

```text
.
├── .github/
│   └── workflows/
│       └── ci.yml
├── Dockerfile
├── cmd/
│   └── chat-server/
│       └── main.go
├── docs/
│   └── demo.gif
├── internal/
│   └── chat/
│       ├── client.go
│       ├── command.go
│       ├── command_test.go
│       ├── room.go
│       ├── server.go
│       └── server_test.go
│   └── store/
│       ├── store.go
│       └── sqlite/
│           ├── migrations/
│           │   └── 001_init.sql
│           ├── repository.go
│           ├── sqlite.go
│           └── sqlite_test.go
├── go.mod
└── README.md
```

## Run locally

```bash
go run ./cmd/chat-server
```

Server listens on `:8080`.
By default it creates/uses `chat.db` in the project root.

## Run with Docker

Build locally:

```bash
docker build -t go-tcp-chat:local .
docker run --rm -p 8080:8080 go-tcp-chat:local
```

Pull from GHCR:

```bash
docker pull ghcr.io/pouyasadri/go-tcp-chat:latest
docker run --rm -p 8080:8080 ghcr.io/pouyasadri/go-tcp-chat:latest
```

## Try it with netcat

Open two terminals and connect both clients:

```bash
nc localhost 8080
```

Then use commands like:

- `/help` show available commands
- `/nick <name>` set your nickname
- `/join <room>` join or create a room
- `/rooms` list active rooms (sorted)
- `/msg <message>` send a message to current room
- `/dm <nick> <message>` send a direct message to a user in the same room
- `/register <username> <password>` create an account and login
- `/login <username> <password>` login with an existing account
- `/logout` clear auth session for current connection
- `/whoami` show current identity state
- `/history` show latest 20 messages in current room
- `/history 50` show latest 50 messages
- `/history before <message_id> <n>` paginate older messages
- `/quit` disconnect

Example:

```text
/join general
/msg hello
/history
/history before 42 10
```

## Notes on design

- The server owns room state and processes commands from clients through a channel.
- A client must join a room before sending messages.
- Direct messages only work for users in the same room.
- `/dm` requires an authenticated user (`/register` or `/login`).
- Rooms and messages are persisted to SQLite.
- Room join prints recent message history (latest 20) when available.
- Empty rooms are cleaned up automatically when users leave.
- Unknown commands return an error plus help text to improve UX.

## Quality checks

Run all checks locally:

```bash
go test ./...
go vet ./...
```

## Next improvements

- Add graceful shutdown with context and OS signals
- Add connection deadlines and idle timeout handling
- Add integration tests for multi-client flows
