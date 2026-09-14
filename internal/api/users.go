package api

import (
	"context"
	"net/url"
	"strconv"
)

func (c *Client) Me(ctx context.Context) (apiUser, error) {
	var out apiUser
	err := c.get(ctx, "me", nil, &out)
	return out, err
}

func (c *Client) UserInfo(ctx context.Context, key, value string) (apiUser, error) {
	var out struct {
		User apiUser `json:"user"`
	}
	q := url.Values{key: []string{value}}
	err := c.get(ctx, "users.info", q, &out)
	return out.User, err
}

func (c *Client) Users(ctx context.Context, offset, count int) ([]apiUser, int, error) {
	var out struct {
		Users  []apiUser `json:"users"`
		Total  int       `json:"total"`
		Count  int       `json:"count"`
		Offset int       `json:"offset"`
	}
	q := url.Values{
		"offset": []string{strconv.Itoa(offset)},
		"count":  []string{strconv.Itoa(count)},
	}
	err := c.get(ctx, "users.list", q, &out)
	return out.Users, out.Total, err
}

func (c *Client) SearchUsers(ctx context.Context, text string, offset, count int) ([]apiUser, int, error) {
	var out struct {
		Result []apiUser `json:"result"`
		Total  int       `json:"total"`
	}
	q := url.Values{
		"text":      []string{text},
		"type":      []string{"users"},
		"workspace": []string{"local"},
		"offset":    []string{strconv.Itoa(offset)},
		"count":     []string{strconv.Itoa(count)},
	}
	err := c.get(ctx, "directory", q, &out)
	return out.Result, out.Total, err
}
