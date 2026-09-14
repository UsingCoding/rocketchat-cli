package api

import "time"

type apiUser struct {
	ID       string   `json:"_id"`
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Active   bool     `json:"active"`
	Roles    []string `json:"roles"`
	Emails   []struct {
		Address  string `json:"address"`
		Verified bool   `json:"verified"`
	} `json:"emails"`
	LastLogin time.Time `json:"lastLogin"`
}

type apiRoom struct {
	ID         string `json:"_id"`
	Name       string `json:"name"`
	FName      string `json:"fname"`
	Type       string `json:"t"`
	Topic      string `json:"topic"`
	Messages   int    `json:"msgs"`
	UsersCount int    `json:"usersCount"`
	Unread     int    `json:"unread"`
	Archived   bool   `json:"archived"`
}

type apiMessage struct {
	ID       string    `json:"_id"`
	RoomID   string    `json:"rid"`
	ThreadID string    `json:"tmid"`
	Text     string    `json:"msg"`
	TS       time.Time `json:"ts"`
	User     struct {
		ID       string `json:"_id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"u"`
}
