package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/rocketchat-cli/internal/clierr"
)

func TestMessageSendRejectsEmptyChannelTargetBeforeHTTP(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	}))
	defer server.Close()

	_, err := executeMessageCommand(t, server.URL, "", "message", "send", ":", "-")
	if clierr.ExitCode(err) != int(clierr.CodeUsage) {
		t.Fatalf("exit code = %d, error = %v", clierr.ExitCode(err), err)
	}
	if err.Error() != "channel target after ':' is empty" {
		t.Fatalf("error = %q", err)
	}
}

func TestMessageCommandsRejectEmptyStdinBeforeHTTP(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	}))
	defer server.Close()

	for _, args := range [][]string{
		{"message", "send", ":platform", "-"},
		{"message", "edit", "message-id", "-"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := executeMessageCommand(t, server.URL, "", args...)
			if clierr.ExitCode(err) != int(clierr.CodeUsage) {
				t.Fatalf("exit code = %d, error = %v", clierr.ExitCode(err), err)
			}
			if err.Error() != "message text from stdin is empty" {
				t.Fatalf("error = %q", err)
			}
		})
	}
}

func TestMessageSendMissingUserReturnsChannelAdvice(t *testing.T) {
	t.Parallel()
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"success":false,"error":"User not found."}`)
	}))
	defer server.Close()

	_, err := executeMessageCommand(t, server.URL, "", "message", "send", "platform", "hello")
	if clierr.ExitCode(err) != int(clierr.CodeNotFound) {
		t.Fatalf("exit code = %d, error = %v", clierr.ExitCode(err), err)
	}
	if !strings.Contains(err.Error(), `prefix it with ':', for example: rocketchat message send :<channel> <text>`) {
		t.Fatalf("missing channel advice: %q", err)
	}
	if strings.Contains(strings.Join(paths, ","), "dm.create") || strings.Contains(strings.Join(paths, ","), "chat.sendMessage") {
		t.Fatalf("mutation request made: %#v", paths)
	}
}

func TestMessageRawOutputUsesFinalMutationResponse(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		args       []string
		finalPath  string
		finalReply string
	}{
		{
			name:       "direct send",
			args:       []string{"message", "send", "alice", "hello"},
			finalPath:  "/api/v1/chat.sendMessage",
			finalReply: `{"message":{"_id":"sent-id","rid":"dm-1","msg":"hello"},"marker":"send","success":true}`,
		},
		{
			name:       "edit",
			args:       []string{"message", "edit", "caller-id", "new"},
			finalPath:  "/api/v1/chat.update",
			finalReply: `{"message":{"_id":"edited-id","rid":"room-1","msg":"new"},"marker":"edit","success":true}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				switch r.URL.Path {
				case "/api/v1/users.info":
					fmt.Fprint(w, `{"user":{"_id":"user-1","username":"alice"},"success":true}`)
				case "/api/v1/dm.create":
					fmt.Fprint(w, `{"room":{"rid":"dm-1"},"success":true}`)
				case "/api/v1/chat.getMessage":
					fmt.Fprint(w, `{"message":{"_id":"canonical-id","rid":"room-1","msg":"old"},"success":true}`)
				case tt.finalPath:
					fmt.Fprint(w, tt.finalReply)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			output, err := executeMessageCommand(t, server.URL, "", append([]string{"--raw"}, tt.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, `"marker": "`+map[bool]string{true: "send", false: "edit"}[tt.name == "direct send"]+`"`) {
				t.Fatalf("raw output = %q", output)
			}
			if paths[len(paths)-1] != tt.finalPath {
				t.Fatalf("paths = %#v", paths)
			}
		})
	}
}

func TestMessageEditHumanAndNormalizedJSONOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/chat.getMessage":
			fmt.Fprint(w, `{"message":{"_id":"canonical-id","rid":"room-1","msg":"old"},"success":true}`)
		case "/api/v1/chat.update":
			fmt.Fprint(w, `{"message":{"_id":"updated-id","rid":"room-1","msg":"new"},"success":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	output, err := executeMessageCommand(t, server.URL, "", "message", "edit", "caller-id", "new")
	if err != nil {
		t.Fatal(err)
	}
	if output != "Message updated: updated-id\n" {
		t.Fatalf("human output = %q", output)
	}

	output, err = executeMessageCommand(t, server.URL, "", "--json", "message", "edit", "caller-id", "new")
	if err != nil {
		t.Fatal(err)
	}
	var message map[string]any
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"id": "updated-id", "room_id": "room-1", "text": "new"} {
		if message[key] != want {
			t.Fatalf("%s = %#v", key, message[key])
		}
	}
	for _, wireKey := range []string{"_id", "rid", "msg"} {
		if _, ok := message[wireKey]; ok {
			t.Fatalf("wire key %q in normalized output: %#v", wireKey, message)
		}
	}
}

func executeMessageCommand(t *testing.T, serverURL, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{
		"--config", filepath.Join(t.TempDir(), "missing.toml"),
		"--url", serverURL,
		"--user-id", "user-1",
		"--token", "token-1",
	}, args...))
	err := cmd.Execute()
	return stdout.String(), err
}
