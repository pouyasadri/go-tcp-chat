package chat

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type Client struct {
	Conn     net.Conn
	NickName string
	Room     *Room
	Commands chan<- Command
}

func (c *Client) ReadInput() {
	reader := bufio.NewReader(c.Conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			c.Commands <- Command{ID: CMDQuit, Client: c}
			return
		}

		line = strings.TrimSpace(line)
		id, args, ok := parseInput(line)
		if !ok {
			c.Err(fmt.Errorf("unknown command: %s", line))
			c.printHelp()
			continue
		}

		c.Commands <- Command{
			ID:     id,
			Client: c,
			Args:   args,
		}
	}
}

func (c *Client) Err(err error) {
	_, _ = c.Conn.Write([]byte("Error: " + err.Error() + "\n"))
}

func (c *Client) Msg(msg string) {
	_, _ = c.Conn.Write([]byte(msg + "\n"))
}

func splitArgs(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	return strings.Fields(line)
}

func (c *Client) printHelp() {
	c.Msg("Available commands:")
	c.Msg("/help            - Show this help")
	c.Msg("/nick <name>     - Change your nickname")
	c.Msg("/join <room>     - Join or create a room")
	c.Msg("/rooms           - List rooms")
	c.Msg("/msg <message>   - Send message to current room")
	c.Msg("/quit            - Disconnect")
}
