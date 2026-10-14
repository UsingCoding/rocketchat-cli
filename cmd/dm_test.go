package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/UsingCoding/rocketchat-cli/internal/clierr"
)

func TestDMHistoryRejectsInvalidInputBeforeHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	}))
	defer server.Close()

	for _, args := range [][]string{
		{"dm", "history", "alice", "--limit", "0"},
		{"dm", "history", "alice", "--since", "not-a-time"},
		{"dm", "history", "alice", "--before", "not-a-time"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := executeCommand(t, server.URL, "", args...)
			if clierr.ExitCode(err) != int(clierr.CodeUsage) {
				t.Fatalf("exit code = %d, error = %v", clierr.ExitCode(err), err)
			}
		})
	}
}

func TestDMHumanAndNormalizedJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			fmt.Fprint(w, `{"room":{"_id":"dm-1","t":"d","usersCount":2},"success":true}`)
		case "/api/v1/me":
			fmt.Fprint(w, `{"_id":"user-1","username":"caller","success":true}`)
		case "/api/v1/dm.list":
			fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["user-1","alice-id"],"usernames":["alice","caller"],"msgs":7,"unread":3,"lm":"2026-09-14T10:00:00Z"}],"total":1,"success":true}`)
		case "/api/v1/dm.messages":
			fmt.Fprint(w, `{"messages":[{"_id":"message-1","rid":"dm-1","msg":"hello","u":{"_id":"alice-id","username":"alice"}}],"total":1,"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name  string
		args  []string
		human []string
		check func(*testing.T, []byte)
	}{
		{
			name:  "list",
			args:  []string{"dm", "list"},
			human: []string{"ID", "USER", "MESSAGES", "UNREAD", "LAST ACTIVITY", "dm-1", "alice"},
			check: func(t *testing.T, b []byte) {
				var dms []map[string]any
				if err := json.Unmarshal(b, &dms); err != nil {
					t.Fatal(err)
				}
				if len(dms) != 1 || dms[0]["id"] != "dm-1" || dms[0]["message_count"] != float64(7) || dms[0]["unread"] != float64(3) {
					t.Fatalf("dms = %#v", dms)
				}
				other, ok := dms[0]["other_participant"].(map[string]any)
				if !ok || other["id"] != "alice-id" || other["username"] != "alice" || dms[0]["last_activity"] != "2026-09-14T10:00:00Z" {
					t.Fatalf("dms = %#v", dms)
				}
			},
		},
		{
			name:  "get",
			args:  []string{"dm", "get", "dm-1"},
			human: []string{"ID:             dm-1", "User:           alice", "Messages:       7", "Unread:         3", "Last activity:  2026-09-14T10:00:00Z"},
			check: func(t *testing.T, b []byte) {
				var dm map[string]any
				if err := json.Unmarshal(b, &dm); err != nil {
					t.Fatal(err)
				}
				other := dm["other_participant"].(map[string]any)
				if dm["id"] != "dm-1" || other["username"] != "alice" || dm["message_count"] != float64(7) || dm["unread"] != float64(3) {
					t.Fatalf("dm = %#v", dm)
				}
			},
		},
		{
			name:  "history",
			args:  []string{"dm", "history", "dm-1"},
			human: []string{"ID", "TIME", "USER", "MESSAGE", "message-1", "alice", "hello"},
			check: func(t *testing.T, b []byte) {
				var messages []map[string]any
				if err := json.Unmarshal(b, &messages); err != nil {
					t.Fatal(err)
				}
				if len(messages) != 1 || messages[0]["id"] != "message-1" || messages[0]["room_id"] != "dm-1" || messages[0]["text"] != "hello" {
					t.Fatalf("messages = %#v", messages)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(t, server.URL, "", tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.human {
				if !strings.Contains(output, want) {
					t.Fatalf("human output missing %q: %q", want, output)
				}
			}

			output, err = executeCommand(t, server.URL, "", append([]string{"--json"}, tt.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, []byte(output))
		})
	}
}

func TestDMRawOutputUsesFinalEndpointResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			fmt.Fprint(w, `{"room":{"_id":"dm-1","t":"d","usersCount":2},"marker":"room","success":true}`)
		case "/api/v1/me":
			fmt.Fprint(w, `{"_id":"user-1","username":"caller","marker":"me","success":true}`)
		case "/api/v1/dm.list":
			fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["user-1","alice-id"],"usernames":["caller","alice"]}],"total":1,"marker":"list","success":true}`)
		case "/api/v1/dm.messages":
			fmt.Fprint(w, `{"messages":[],"total":0,"marker":"history","success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	for _, tt := range []struct {
		name   string
		args   []string
		marker string
	}{
		{name: "list", args: []string{"dm", "list"}, marker: "list"},
		{name: "get", args: []string{"dm", "get", "dm-1"}, marker: "list"},
		{name: "history", args: []string{"dm", "history", "dm-1"}, marker: "history"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(t, server.URL, "", append([]string{"--raw"}, tt.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, `"marker": "`+tt.marker+`"`) {
				t.Fatalf("raw output = %q", output)
			}
		})
	}
}

func TestDMGetMissingConversationReturnsNotFoundWithoutMutation(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"success":false,"error":"room not found"}`)
		case "/api/v1/users.info":
			fmt.Fprint(w, `{"user":{"_id":"alice-id","username":"alice"},"success":true}`)
		case "/api/v1/me":
			fmt.Fprint(w, `{"_id":"user-1","username":"caller","success":true}`)
		case "/api/v1/dm.list":
			fmt.Fprint(w, `{"ims":[],"total":0,"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := executeCommand(t, server.URL, "", "dm", "get", "alice")
	if clierr.ExitCode(err) != int(clierr.CodeNotFound) {
		t.Fatalf("exit code = %d, error = %v", clierr.ExitCode(err), err)
	}
	if strings.Contains(strings.Join(paths, ","), "dm.create") {
		t.Fatalf("mutation request made: %#v", paths)
	}
}

func TestDMRawOutputRendersMemberResponseOnParticipantError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			fmt.Fprint(w, `{"_id":"user-1","username":"caller","success":true}`)
		case "/api/v1/dm.list":
			fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2}],"total":1,"marker":"invalid-room","success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	output, err := executeCommand(t, server.URL, "", "--raw", "dm", "list")
	if err == nil || !strings.Contains(err.Error(), "invalid participant metadata") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(output, `"marker": "invalid-room"`) {
		t.Fatalf("raw output = %q", output)
	}
}
