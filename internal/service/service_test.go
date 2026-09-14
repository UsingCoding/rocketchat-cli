package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
