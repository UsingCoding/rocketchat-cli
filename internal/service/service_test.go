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
	"github.com/UsingCoding/rocketchat-cli/internal/model"
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

func TestDirectMessagesPagingAndNormalization(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(fmt.Sprintf("all=%t", all), func(t *testing.T) {
			var offsets []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/me" {
					fmt.Fprint(w, `{"_id":"caller","username":"caller","success":true}`)
					return
				}
				if r.URL.Path != "/api/v1/dm.list" {
					http.NotFound(w, r)
					return
				}
				q := r.URL.Query()
				if q.Get("count") != "50" || q.Get("sort") != `{"_updatedAt":-1}` {
					t.Fatalf("query = %q", r.URL.RawQuery)
				}
				offsets = append(offsets, q.Get("offset"))
				switch q.Get("offset") {
				case "0":
					fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["caller","alice-id"],"usernames":["alice","caller"],"msgs":7,"unread":3,"lm":"2026-09-14T10:00:00Z"},{"_id":"group-1","t":"d","usersCount":3}],"total":3,"success":true}`)
				case "2":
					fmt.Fprint(w, `{"ims":[{"_id":"dm-2","t":"d","usersCount":2,"uids":["caller","bob-id"],"usernames":["caller","bob"]}],"total":3,"success":true}`)
				default:
					t.Fatalf("offset = %q", q.Get("offset"))
				}
			}))
			defer server.Close()

			dms, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessages(context.Background(), all)
			if err != nil {
				t.Fatal(err)
			}
			wantOffsets := []string{"0"}
			if all {
				wantOffsets = append(wantOffsets, "2")
			}
			if !reflect.DeepEqual(offsets, wantOffsets) {
				t.Fatalf("offsets = %#v, want %#v", offsets, wantOffsets)
			}
			wantIDs := []string{"dm-1"}
			if all {
				wantIDs = append(wantIDs, "dm-2")
			}
			if len(dms) != len(wantIDs) {
				t.Fatalf("dms = %#v", dms)
			}
			for i, want := range wantIDs {
				if dms[i].ID != want {
					t.Fatalf("dm %d ID = %q, want %q", i, dms[i].ID, want)
				}
			}
			if dms[0].OtherParticipant != (model.Participant{ID: "alice-id", Username: "alice"}) || dms[0].MessageCount != 7 || dms[0].Unread != 3 || dms[0].LastActivity.Format(time.RFC3339) != "2026-09-14T10:00:00Z" {
				t.Fatalf("dm = %#v", dms[0])
			}
		})
	}
}

func TestDirectMessageResolvesRoomAndUserIdentifiers(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		lookups    []string
	}{
		{name: "room ID", identifier: "dm-1"},
		{name: "username", identifier: "alice", lookups: []string{"username"}},
		{name: "user ID fallback", identifier: "alice-id", lookups: []string{"username", "userId"}},
		{name: "email", identifier: "alice@example.com", lookups: []string{"email"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			var userLookups int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				switch r.URL.Path {
				case "/api/v1/rooms.info":
					if r.URL.Query().Get("roomName") == tt.identifier {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"success":false,"error":"room not found"}`)
						return
					}
					if r.URL.Query().Get("roomId") != tt.identifier {
						t.Fatalf("room lookup = %q", r.URL.RawQuery)
					}
					if tt.lookups != nil {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"success":false,"error":"room not found"}`)
						return
					}
					fmt.Fprint(w, `{"room":{"_id":"dm-1","t":"d","usersCount":2},"success":true}`)
				case "/api/v1/users.info":
					if got := r.URL.Query().Get(tt.lookups[userLookups]); got != tt.identifier {
						t.Fatalf("user lookup = %q", r.URL.RawQuery)
					}
					userLookups++
					if r.URL.Query().Get("username") == "alice-id" {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"success":false,"error":"User not found."}`)
						return
					}
					fmt.Fprint(w, `{"user":{"_id":"alice-id","username":"alice"},"success":true}`)
				case "/api/v1/me":
					fmt.Fprint(w, `{"_id":"caller","username":"caller","success":true}`)
				case "/api/v1/dm.list":
					fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["caller","alice-id"],"usernames":["caller","alice"]}],"total":1,"success":true}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			dm, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessage(context.Background(), tt.identifier)
			if err != nil {
				t.Fatal(err)
			}
			if dm.ID != "dm-1" || dm.OtherParticipant.Username != "alice" {
				t.Fatalf("dm = %#v", dm)
			}
			want := []string{"/api/v1/me", "/api/v1/rooms.info"}
			if tt.lookups != nil {
				want = append(want, "/api/v1/dm.list")
			}
			for range tt.lookups {
				want = append(want, "/api/v1/users.info")
			}
			want = append(want, "/api/v1/dm.list")
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("paths = %#v, want %#v", paths, want)
			}
		})
	}
}

func TestResolveDirectMessageFindsJoinedRoomAfterRoomInfoMiss(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/rooms.info":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"success":false,"error":"room not found"}`)
		case "/api/v1/dm.list":
			if r.URL.Query().Get("offset") != "0" || r.URL.Query().Get("count") != "50" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["caller","alice-id"],"usernames":["caller","alice"]}],"total":1,"success":true}`)
		case "/api/v1/users.info":
			t.Fatalf("unexpected user lookup: %q", r.URL.RawQuery)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	room, err := New(api.New(server.URL, "caller", "t", time.Second)).resolveDirectMessage(context.Background(), "dm-1")
	if err != nil {
		t.Fatal(err)
	}
	if room.ID != "dm-1" {
		t.Fatalf("room = %#v", room)
	}
	if !reflect.DeepEqual(paths, []string{"/api/v1/rooms.info", "/api/v1/dm.list"}) {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestDirectMessageNotFoundAndGroupRejection(t *testing.T) {
	t.Run("missing joined DM", func(t *testing.T) {
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
				fmt.Fprint(w, `{"_id":"caller","username":"caller","success":true}`)
			case "/api/v1/dm.list":
				fmt.Fprint(w, `{"ims":[],"total":0,"success":true}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		_, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessage(context.Background(), "alice")
		if !api.IsNotFound(err) {
			t.Fatalf("error = %v", err)
		}
		if strings.Contains(strings.Join(paths, ","), "dm.create") {
			t.Fatalf("mutation request made: %#v", paths)
		}
	})

	t.Run("group room ID", func(t *testing.T) {
		var paths []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			if r.URL.Path == "/api/v1/rooms.info" {
				fmt.Fprint(w, `{"room":{"_id":"group-1","t":"d","usersCount":3},"success":true}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		_, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessageHistory(context.Background(), "group-1", api.HistoryOptions{Count: 50})
		if err == nil || !strings.Contains(err.Error(), "not a one-to-one direct message") {
			t.Fatalf("error = %v", err)
		}
		if !reflect.DeepEqual(paths, []string{"/api/v1/rooms.info"}) {
			t.Fatalf("paths = %#v", paths)
		}
	})
}

func TestDirectMessageHistoryQueryAndFailures(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		var paths []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			switch r.URL.Path {
			case "/api/v1/rooms.info":
				fmt.Fprint(w, `{"room":{"_id":"dm-1","t":"d","usersCount":2},"success":true}`)
			case "/api/v1/dm.list":
				fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["caller","alice-id"],"usernames":["caller","alice"]}],"total":1,"success":true}`)
			case "/api/v1/me":
				fmt.Fprint(w, `{"_id":"caller","username":"caller","success":true}`)
			case "/api/v1/dm.messages":
				q := r.URL.Query()
				if q.Get("roomId") != "dm-1" || q.Get("offset") != "2" || q.Get("count") != "3" || q.Get("oldest") != "2026-09-14T10:00:00Z" || q.Get("latest") != "2026-09-14T11:00:00Z" {
					t.Fatalf("query = %q", r.URL.RawQuery)
				}
				fmt.Fprint(w, `{"messages":[{"_id":"message-1","rid":"dm-1","msg":"hello","u":{"_id":"alice-id","username":"alice"}}],"total":1,"success":true}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		messages, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessageHistory(context.Background(), "dm-1", api.HistoryOptions{Offset: 2, Count: 3, Oldest: "2026-09-14T10:00:00Z", Latest: "2026-09-14T11:00:00Z"})
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 1 || messages[0].RoomID != "dm-1" || messages[0].ID != "message-1" {
			t.Fatalf("messages = %#v", messages)
		}
		if !reflect.DeepEqual(paths, []string{"/api/v1/rooms.info", "/api/v1/dm.list", "/api/v1/dm.messages"}) {
			t.Fatalf("paths = %#v", paths)
		}
	})

	for _, tt := range []struct {
		name      string
		failPath  string
		wantPaths []string
	}{
		{name: "room lookup", failPath: "/api/v1/rooms.info", wantPaths: []string{"/api/v1/rooms.info"}},
		{name: "user lookup", failPath: "/api/v1/users.info", wantPaths: []string{"/api/v1/rooms.info", "/api/v1/dm.list", "/api/v1/users.info"}},
		{name: "list pagination", failPath: "/api/v1/dm.list", wantPaths: []string{"/api/v1/rooms.info", "/api/v1/dm.list"}},
		{name: "history", failPath: "/api/v1/dm.messages", wantPaths: []string{"/api/v1/rooms.info", "/api/v1/dm.list", "/api/v1/dm.messages"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				if r.URL.Path == tt.failPath {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, `{"success":false,"error":"failed"}`)
					return
				}
				switch r.URL.Path {
				case "/api/v1/rooms.info":
					if tt.failPath == "/api/v1/users.info" {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"success":false,"error":"room not found"}`)
						return
					}
					fmt.Fprint(w, `{"room":{"_id":"dm-1","t":"d","usersCount":2},"success":true}`)
				case "/api/v1/users.info":
					fmt.Fprint(w, `{"user":{"_id":"alice-id","username":"alice"},"success":true}`)
				case "/api/v1/dm.list":
					fmt.Fprint(w, `{"ims":[{"_id":"dm-1","t":"d","usersCount":2,"uids":["caller","alice-id"],"usernames":["caller","alice"]}],"total":1,"success":true}`)
				case "/api/v1/me":
					fmt.Fprint(w, `{"_id":"caller","username":"caller","success":true}`)
				case "/api/v1/dm.messages":
					fmt.Fprint(w, `{"messages":[],"total":0,"success":true}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			identifier := "dm-1"
			if tt.failPath == "/api/v1/users.info" {
				identifier = "alice"
			}
			_, err := New(api.New(server.URL, "caller", "t", time.Second)).DirectMessageHistory(context.Background(), identifier, api.HistoryOptions{Count: 50})
			if err == nil {
				t.Fatal("expected error")
			}
			if !reflect.DeepEqual(paths, tt.wantPaths) {
				t.Fatalf("paths = %#v, want %#v", paths, tt.wantPaths)
			}
		})
	}
}
