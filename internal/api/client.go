package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Error struct {
	StatusCode int
	ErrorType  string
	Message    string
}

func (e *Error) Error() string {
	if e.ErrorType != "" {
		return fmt.Sprintf("Rocket.Chat API: %s (%s)", e.Message, e.ErrorType)
	}
	return fmt.Sprintf("Rocket.Chat API: %s", e.Message)
}

func (e *Error) NotFound() bool {
	m := strings.ToLower(e.Message + " " + e.ErrorType)
	return e.StatusCode == http.StatusNotFound || strings.Contains(m, "not found") || strings.Contains(m, "room-not-found") || strings.Contains(m, "invalid-message")
}

type Client struct {
	baseURL string
	userID  string
	token   string
	http    *http.Client

	mu      sync.Mutex
	lastRaw []byte
}

func New(baseURL, userID, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/") + "/api/v1",
		userID:  userID,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) LastRaw() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.lastRaw...)
}

func (c *Client) setLastRaw(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastRaw = append(c.lastRaw[:0], b...)
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, q, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, body any, out any) error {
	u := c.baseURL + "/" + strings.TrimLeft(path, "/")
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, r)
	if err != nil {
		return err
	}
	req.Header.Set("X-User-Id", c.userID)
	req.Header.Set("X-Auth-Token", c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.setLastRaw(b)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp.StatusCode, b)
	}
	var envelope struct {
		Success *bool  `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Type    string `json:"errorType"`
	}
	_ = json.Unmarshal(b, &envelope)
	if envelope.Success != nil && !*envelope.Success {
		msg := envelope.Error
		if msg == "" {
			msg = envelope.Message
		}
		return &Error{StatusCode: resp.StatusCode, ErrorType: envelope.Type, Message: msg}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode Rocket.Chat response: %w", err)
	}
	return nil
}

func decodeAPIError(status int, b []byte) error {
	var x struct {
		Error     string `json:"error"`
		ErrorType string `json:"errorType"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(b, &x); err != nil {
		return &Error{StatusCode: status, Message: strings.TrimSpace(string(b))}
	}
	msg := x.Error
	if msg == "" {
		msg = x.Message
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &Error{StatusCode: status, ErrorType: x.ErrorType, Message: msg}
}

func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.NotFound()
}
