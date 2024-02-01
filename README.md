# Go Chat Server

This is a simple chat server implemented in Go. It allows multiple clients to connect, join rooms, and send messages to each other.

## Project Structure

The project consists of several Go files:

- `main.go`: This is the entry point of the application. It starts the server and listens for incoming TCP connections.
- `server.go`: This file defines the `Server` struct and its methods. The server handles incoming commands from clients and broadcasts messages to all clients in a room.
- `client.go`: This file defines the `Client` struct and its methods. Each client represents a connection from a user.
- `command.go`: This file defines the `Command` struct and the different command IDs.

## How to Run

To run the server, use the following command:

```bash
go run main.go
```
## How to Use
Once the server is running, you can connect to it using any TCP client (like netcat). Here are some commands you can use: 
- `/nick <name>`: Change your nickname.
- `/join <room>`: Join a room.
- `/msg <message>`: Send a message to the current room.
- `/quit`: Disconnect from the server.
## Future Improvements
- Add support for private messages.
- Add support for user authentication.
- Add support for persistent storage of messages.
## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
