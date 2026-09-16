package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/UsingCoding/rocketchat-cli/internal/api"
)

func TestReplyToReplyUsesThreadRoot(t *testing.T) {
	t.Parallel()
	var sent map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/chat.getMessage":
			fmt.Fprint(w, `{"message":{"_id":"reply-1","rid":"room-1","tmid":"root-1","msg":"old","u":{"_id":"u1","username":"alice"}},"success":true}`)
		case "/api/v1/chat.sendMessage":
			if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
				t.Fatal(err)
			}
			fmt.Fprint(w, `{"message":{"_id":"reply-2","rid":"room-1","tmid":"root-1","msg":"new","u":{"_id":"u2","username":"bob"}},"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := New(api.New(server.URL, "u", "t", time.Second))
	m, err := svc.Reply(context.Background(), "reply-1", "new", true)
	if err != nil {
		t.Fatal(err)
	}
	if m.ThreadID != "root-1" {
		t.Fatalf("thread id = %q", m.ThreadID)
	}
	message, ok := sent["message"].(map[string]any)
	if !ok {
		t.Fatalf("bad body: %#v", sent)
	}
	if message["tmid"] != "root-1" {
		t.Fatalf("tmid = %#v", message["tmid"])
	}
	if message["tshow"] != true {
		t.Fatalf("tshow = %#v", message["tshow"])
	}
}

func TestResolvePrivateRoomThenHistory(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			if r.URL.Query().Get("roomName") != "platform" {
				t.Fatalf("roomName = %q", r.URL.Query().Get("roomName"))
			}
			fmt.Fprint(w, `{"room":{"_id":"room-p","name":"platform","t":"p"},"success":true}`)
		case "/api/v1/groups.history":
			if r.URL.Query().Get("roomId") != "room-p" {
				t.Fatalf("roomId = %q", r.URL.Query().Get("roomId"))
			}
			fmt.Fprint(w, `{"messages":[{"_id":"m1","rid":"room-p","msg":"hello","u":{"_id":"u1","username":"alice"}}],"total":1,"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := New(api.New(server.URL, "u", "t", time.Second))
	messages, err := svc.ChannelHistory(context.Background(), "platform", api.HistoryOptions{Count: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Text != "hello" {
		t.Fatalf("messages = %#v", messages)
	}
}

func TestSendRoutesChannelTargetWithoutUserLookup(t *testing.T) {
	t.Parallel()
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			if got := r.URL.Query().Get("roomName"); got != "platform" {
				t.Fatalf("roomName = %q", got)
			}
			fmt.Fprint(w, `{"room":{"_id":"room-platform","name":"platform","t":"c"},"success":true}`)
		case "/api/v1/chat.sendMessage":
			assertJSONBody(t, r, map[string]any{"message": map[string]any{"rid": "room-platform", "msg": "hello"}})
			fmt.Fprint(w, `{"message":{"_id":"message-1","rid":"room-platform","msg":"hello","u":{"_id":"u1","username":"alice"}},"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m, err := New(api.New(server.URL, "u", "t", time.Second)).Send(context.Background(), ":platform", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"/api/v1/rooms.info", "/api/v1/chat.sendMessage"}) {
		t.Fatalf("paths = %#v", paths)
	}
	if m.ID != "message-1" || m.RoomID != "room-platform" || m.Text != "hello" {
		t.Fatalf("message = %#v", m)
	}
}

func TestSendDirectMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		target         string
		lookupKey      string
		userIDFallback bool
	}{
		{name: "username", target: "alice", lookupKey: "username"},
		{name: "user ID fallback", target: "user-1", lookupKey: "username", userIDFallback: true},
		{name: "email", target: "alice@example.com", lookupKey: "email"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				switch r.URL.Path {
				case "/api/v1/users.info":
					if len(paths) == 1 && r.URL.Query().Get(tt.lookupKey) != tt.target {
						t.Fatalf("initial lookup = %q", r.URL.RawQuery)
					}
					if tt.userIDFallback && r.URL.Query().Get("username") == tt.target {
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprint(w, `{"success":false,"error":"User not found."}`)
						return
					}
					if tt.userIDFallback && r.URL.Query().Get("userId") != tt.target {
						t.Fatalf("fallback lookup = %q", r.URL.RawQuery)
					}
					fmt.Fprint(w, `{"user":{"_id":"user-1","username":"alice"},"success":true}`)
				case "/api/v1/dm.create":
					assertJSONBody(t, r, map[string]any{"username": "alice"})
					fmt.Fprint(w, `{"room":{"rid":"dm-1"},"success":true}`)
				case "/api/v1/chat.sendMessage":
					assertJSONBody(t, r, map[string]any{"message": map[string]any{"rid": "dm-1", "msg": "hello"}})
					fmt.Fprint(w, `{"message":{"_id":"message-1","rid":"dm-1","msg":"hello","u":{"_id":"user-1","username":"alice"}},"success":true}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			m, err := New(api.New(server.URL, "u", "t", time.Second)).Send(context.Background(), tt.target, "hello")
			if err != nil {
				t.Fatal(err)
			}
			wantPaths := []string{"/api/v1/users.info"}
			if tt.userIDFallback {
				wantPaths = append(wantPaths, "/api/v1/users.info")
			}
			wantPaths = append(wantPaths, "/api/v1/dm.create", "/api/v1/chat.sendMessage")
			if !reflect.DeepEqual(paths, wantPaths) {
				t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
			}
			if m.ID != "message-1" || m.RoomID != "dm-1" || m.Text != "hello" {
				t.Fatalf("message = %#v", m)
			}
		})
	}
}

func TestSendFailuresShortCircuit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		target    string
		responses map[string]string
		wantPaths []string
		wantError string
	}{
		{
			name:      "channel resolution",
			target:    ":missing",
			responses: map[string]string{"/api/v1/rooms.info": `{"success":false,"error":"Room not found."}`},
			wantPaths: []string{"/api/v1/rooms.info", "/api/v1/rooms.info"},
		},
		{
			name:      "user lookup",
			target:    "alice",
			responses: map[string]string{"/api/v1/users.info": `{"success":false,"error":"server unavailable"}`},
			wantPaths: []string{"/api/v1/users.info"},
		},
		{
			name:   "DM creation",
			target: "alice",
			responses: map[string]string{
				"/api/v1/users.info": `{"user":{"_id":"user-1","username":"alice"},"success":true}`,
				"/api/v1/dm.create":  `{"success":false,"error":"DM unavailable"}`,
			},
			wantPaths: []string{"/api/v1/users.info", "/api/v1/dm.create"},
		},
		{
			name:   "final send",
			target: "alice",
			responses: map[string]string{
				"/api/v1/users.info":       `{"user":{"_id":"user-1","username":"alice"},"success":true}`,
				"/api/v1/dm.create":        `{"room":{"rid":"dm-1"},"success":true}`,
				"/api/v1/chat.sendMessage": `{"success":false,"error":"send unavailable"}`,
			},
			wantPaths: []string{"/api/v1/users.info", "/api/v1/dm.create", "/api/v1/chat.sendMessage"},
		},
		{
			name:   "missing DM room ID",
			target: "alice",
			responses: map[string]string{
				"/api/v1/users.info": `{"user":{"_id":"user-1","username":"alice"},"success":true}`,
				"/api/v1/dm.create":  `{"room":{},"success":true}`,
			},
			wantPaths: []string{"/api/v1/users.info", "/api/v1/dm.create"},
			wantError: "dm.create response missing room.rid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				response, ok := tt.responses[r.URL.Path]
				if !ok {
					http.NotFound(w, r)
					return
				}
				if strings.Contains(response, `"success":false`) {
					w.WriteHeader(http.StatusBadRequest)
				}
				fmt.Fprint(w, response)
			}))
			defer server.Close()

			_, err := New(api.New(server.URL, "u", "t", time.Second)).Send(context.Background(), tt.target, "hello")
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantError != "" && err.Error() != tt.wantError {
				t.Fatalf("error = %q", err)
			}
			if !reflect.DeepEqual(paths, tt.wantPaths) {
				t.Fatalf("paths = %#v, want %#v", paths, tt.wantPaths)
			}
		})
	}
}

func TestSendMissingUserIncludesChannelAdvice(t *testing.T) {
	t.Parallel()
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"success":false,"error":"User not found."}`)
	}))
	defer server.Close()

	_, err := New(api.New(server.URL, "u", "t", time.Second)).Send(context.Background(), "platform", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	const want = `"platform" was not found as a user. If this is a channel, prefix it with ':', for example: rocketchat message send :<channel> <text>: Rocket.Chat API: User not found.`
	if err.Error() != want {
		t.Fatalf("error = %q", err)
	}
	if !api.IsNotFound(err) {
		t.Fatalf("not-found classification lost: %v", err)
	}
	if !reflect.DeepEqual(paths, []string{"/api/v1/users.info", "/api/v1/users.info"}) {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestEditFetchesCanonicalMessageThenUpdates(t *testing.T) {
	t.Parallel()
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/chat.getMessage":
			if got := r.URL.Query().Get("msgId"); got != "caller-id" {
				t.Fatalf("msgId = %q", got)
			}
			fmt.Fprint(w, `{"message":{"_id":"canonical-id","rid":"room-1","msg":"old","u":{"_id":"u1","username":"alice"}},"success":true}`)
		case "/api/v1/chat.update":
			assertJSONBody(t, r, map[string]any{"roomId": "room-1", "msgId": "canonical-id", "text": "new"})
			fmt.Fprint(w, `{"message":{"_id":"updated-id","rid":"room-1","msg":"new","u":{"_id":"u1","username":"alice"}},"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m, err := New(api.New(server.URL, "u", "t", time.Second)).Edit(context.Background(), "caller-id", "new")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"/api/v1/chat.getMessage", "/api/v1/chat.update"}) {
		t.Fatalf("paths = %#v", paths)
	}
	if m.ID != "updated-id" || m.RoomID != "room-1" || m.Text != "new" {
		t.Fatalf("message = %#v", m)
	}
}

func TestEditFailuresShortCircuit(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name      string
		responses map[string]string
		wantPaths []string
	}{
		{
			name:      "lookup failure",
			responses: map[string]string{"/api/v1/chat.getMessage": `{"success":false,"error":"Message not found."}`},
			wantPaths: []string{"/api/v1/chat.getMessage"},
		},
		{
			name: "update failure",
			responses: map[string]string{
				"/api/v1/chat.getMessage": `{"message":{"_id":"canonical-id","rid":"room-1"},"success":true}`,
				"/api/v1/chat.update":     `{"success":false,"error":"update unavailable"}`,
			},
			wantPaths: []string{"/api/v1/chat.getMessage", "/api/v1/chat.update"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				response := tt.responses[r.URL.Path]
				if strings.Contains(response, `"success":false`) {
					w.WriteHeader(http.StatusBadRequest)
				}
				fmt.Fprint(w, response)
			}))
			defer server.Close()

			_, err := New(api.New(server.URL, "u", "t", time.Second)).Edit(context.Background(), "caller-id", "new")
			if err == nil {
				t.Fatal("expected error")
			}
			if !reflect.DeepEqual(paths, tt.wantPaths) {
				t.Fatalf("paths = %#v, want %#v", paths, tt.wantPaths)
			}
		})
	}
}

func assertJSONBody(t *testing.T, r *http.Request, want any) {
	t.Helper()
	var got any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}
