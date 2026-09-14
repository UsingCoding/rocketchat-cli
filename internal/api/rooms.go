package api

import (
	"context"
	"net/url"
	"strconv"
)

func (c *Client) RoomInfo(ctx context.Context, key, value string) (apiRoom, error) {
	var out struct {
		Room apiRoom `json:"room"`
	}
	q := url.Values{key: []string{value}}
	err := c.get(ctx, "rooms.info", q, &out)
	return out.Room, err
}

func (c *Client) Rooms(ctx context.Context) ([]apiRoom, error) {
	var out struct {
		Update []apiRoom `json:"update"`
	}
	err := c.get(ctx, "rooms.get", nil, &out)
	return out.Update, err
}

func (c *Client) Members(ctx context.Context, roomType, roomID string, offset, count int) ([]apiUser, int, error) {
	endpoint := "channels.members"
	if roomType == "p" {
		endpoint = "groups.members"
	}
	var out struct {
		Members []apiUser `json:"members"`
		Total   int       `json:"total"`
	}
	q := url.Values{
		"roomId": []string{roomID},
		"offset": []string{strconv.Itoa(offset)},
		"count":  []string{strconv.Itoa(count)},
	}
	err := c.get(ctx, endpoint, q, &out)
	return out.Members, out.Total, err
}

type HistoryOptions struct {
	Offset int
	Count  int
	Oldest string
	Latest string
}

func (c *Client) History(ctx context.Context, roomType, roomID string, o HistoryOptions) ([]apiMessage, int, error) {
	endpoint := "channels.history"
	if roomType == "p" {
		endpoint = "groups.history"
	}
	var out struct {
		Messages []apiMessage `json:"messages"`
		Total    int          `json:"total"`
	}
	q := url.Values{
		"roomId": []string{roomID},
		"offset": []string{strconv.Itoa(o.Offset)},
		"count":  []string{strconv.Itoa(o.Count)},
	}
	if o.Oldest != "" {
		q.Set("oldest", o.Oldest)
	}
	if o.Latest != "" {
		q.Set("latest", o.Latest)
	}
	err := c.get(ctx, endpoint, q, &out)
	return out.Messages, out.Total, err
}
