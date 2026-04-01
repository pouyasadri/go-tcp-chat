package chat

type CommandID int

const (
	CMDHelp CommandID = iota
	CMDNick
	CMDJoin
	CMDRooms
	CMDMsg
	CMDDM
	CMDRegister
	CMDLogin
	CMDLogout
	CMDWhoAmI
	CMDQuit
)

type Command struct {
	ID     CommandID
	Client *Client
	Args   []string
}

func parseInput(line string) (CommandID, []string, bool) {
	args := splitArgs(line)
	if len(args) == 0 {
		return CMDHelp, nil, false
	}

	switch args[0] {
	case "/help":
		return CMDHelp, args, true
	case "/nick":
		return CMDNick, args, true
	case "/join":
		return CMDJoin, args, true
	case "/rooms":
		return CMDRooms, args, true
	case "/msg":
		return CMDMsg, args, true
	case "/dm":
		return CMDDM, args, true
	case "/register":
		return CMDRegister, args, true
	case "/login":
		return CMDLogin, args, true
	case "/logout":
		return CMDLogout, args, true
	case "/whoami":
		return CMDWhoAmI, args, true
	case "/quit":
		return CMDQuit, args, true
	default:
		return CMDHelp, args, false
	}
}
