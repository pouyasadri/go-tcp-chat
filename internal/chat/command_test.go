package chat

import "testing"

func TestParseInput(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantID  CommandID
		wantOK  bool
		wantLen int
	}{
		{name: "help", line: "/help", wantID: CMDHelp, wantOK: true, wantLen: 1},
		{name: "nick", line: "/nick pouya", wantID: CMDNick, wantOK: true, wantLen: 2},
		{name: "join", line: "/join general", wantID: CMDJoin, wantOK: true, wantLen: 2},
		{name: "rooms", line: "/rooms", wantID: CMDRooms, wantOK: true, wantLen: 1},
		{name: "msg", line: "/msg hello world", wantID: CMDMsg, wantOK: true, wantLen: 3},
		{name: "dm", line: "/dm alice hey there", wantID: CMDDM, wantOK: true, wantLen: 4},
		{name: "register", line: "/register alice secret123", wantID: CMDRegister, wantOK: true, wantLen: 3},
		{name: "login", line: "/login alice secret123", wantID: CMDLogin, wantOK: true, wantLen: 3},
		{name: "logout", line: "/logout", wantID: CMDLogout, wantOK: true, wantLen: 1},
		{name: "whoami", line: "/whoami", wantID: CMDWhoAmI, wantOK: true, wantLen: 1},
		{name: "history", line: "/history", wantID: CMDHistory, wantOK: true, wantLen: 1},
		{name: "history before", line: "/history before 42 10", wantID: CMDHistory, wantOK: true, wantLen: 4},
		{name: "quit", line: "/quit", wantID: CMDQuit, wantOK: true, wantLen: 1},
		{name: "empty", line: " ", wantID: CMDHelp, wantOK: false, wantLen: 0},
		{name: "unknown", line: "/noop", wantID: CMDHelp, wantOK: false, wantLen: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotArgs, gotOK := parseInput(tc.line)
			if gotID != tc.wantID {
				t.Fatalf("parseInput() id = %v, want %v", gotID, tc.wantID)
			}
			if gotOK != tc.wantOK {
				t.Fatalf("parseInput() ok = %v, want %v", gotOK, tc.wantOK)
			}
			if len(gotArgs) != tc.wantLen {
				t.Fatalf("parseInput() args len = %d, want %d", len(gotArgs), tc.wantLen)
			}
		})
	}
}

func TestParseHistoryArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLimit  int
		wantBefore *int64
		wantErr    bool
	}{
		{name: "default", args: []string{"/history"}, wantLimit: 20},
		{name: "limit", args: []string{"/history", "10"}, wantLimit: 10},
		{name: "before default limit", args: []string{"/history", "before", "50"}, wantLimit: 20, wantBefore: int64Ptr(50)},
		{name: "before with limit", args: []string{"/history", "before", "50", "5"}, wantLimit: 5, wantBefore: int64Ptr(50)},
		{name: "invalid limit", args: []string{"/history", "0"}, wantErr: true},
		{name: "invalid before id", args: []string{"/history", "before", "x"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotLimit, gotBefore, err := parseHistoryArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotLimit != tc.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tc.wantLimit)
			}
			if (gotBefore == nil) != (tc.wantBefore == nil) {
				t.Fatalf("before nil mismatch")
			}
			if gotBefore != nil && *gotBefore != *tc.wantBefore {
				t.Fatalf("before = %d, want %d", *gotBefore, *tc.wantBefore)
			}
		})
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{name: "trim spaces", line: "  /join   room  ", want: []string{"/join", "room"}},
		{name: "empty", line: "", want: nil},
		{name: "tabs", line: "/msg\thello", want: []string{"/msg", "hello"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitArgs(tc.line)
			if len(got) != len(tc.want) {
				t.Fatalf("splitArgs() len = %d, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("splitArgs()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
