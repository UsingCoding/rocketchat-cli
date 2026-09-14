package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name,omitempty"`
	Email     string    `json:"email,omitempty"`
	Status    string    `json:"status,omitempty"`
	Active    bool      `json:"active"`
	Roles     []string  `json:"roles,omitempty"`
	LastLogin time.Time `json:"last_login,omitempty"`
}

type Channel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Topic        string `json:"topic,omitempty"`
	MessageCount int    `json:"message_count,omitempty"`
	MemberCount  int    `json:"member_count,omitempty"`
	Unread       int    `json:"unread,omitempty"`
	Archived     bool   `json:"archived,omitempty"`
}

type Author struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
}

type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	ThreadID  string    `json:"thread_id,omitempty"`
	Text      string    `json:"text"`
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type Thread struct {
	Root    Message   `json:"root"`
	Replies []Message `json:"replies"`
}
