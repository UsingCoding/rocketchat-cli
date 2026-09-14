package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeSendsAuthenticationHeaders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-User-Id"); got != "user-1" {
			t.Fatalf("X-User-Id = %q", got)
		}
		if got := r.Header.Get("X-Auth-Token"); got != "token-1" {
			t.Fatalf("X-Auth-Token = %q", got)
		}
		if r.URL.Path != "/api/v1/me" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"_id":"user-1","username":"vadim","active":true}`)
	}))
	defer server.Close()

	client := New(server.URL, "user-1", "token-1", time.Second)
	u, err := client.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != "user-1" || u.Username != "vadim" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestAPIErrorNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"success":false,"error":"User not found."}`)
	}))
	defer server.Close()

	client := New(server.URL, "u", "t", time.Second)
	_, err := client.UserInfo(context.Background(), "username", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
