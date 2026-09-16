package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) Message(ctx context.Context, id string) (apiMessage, error) {
	var out struct {
		Message apiMessage `json:"message"`
	}
	err := c.get(ctx, "chat.getMessage", url.Values{"msgId": []string{id}}, &out)
	return out.Message, err
}

func (c *Client) SendMessage(ctx context.Context, roomID, text, threadID string, alsoSend bool) (apiMessage, error) {
	msg := map[string]any{
		"rid": roomID,
		"msg": text,
	}
	if threadID != "" {
		msg["tmid"] = threadID
		if alsoSend {
			msg["tshow"] = true
		}
	}
	var out struct {
		Message apiMessage `json:"message"`
	}
	err := c.post(ctx, "chat.sendMessage", map[string]any{"message": msg}, &out)
	return out.Message, err
}

func (c *Client) CreateDM(ctx context.Context, username string) (string, error) {
	var out struct {
		Room struct {
			ID string `json:"rid"`
		} `json:"room"`
	}
	if err := c.post(ctx, "dm.create", map[string]string{"username": username}, &out); err != nil {
		return "", err
	}
	if out.Room.ID == "" {
		return "", fmt.Errorf("dm.create response missing room.rid")
	}
	return out.Room.ID, nil
}

func (c *Client) UpdateMessage(ctx context.Context, roomID, messageID, text string) (apiMessage, error) {
	var out struct {
		Message apiMessage `json:"message"`
	}
	err := c.post(ctx, "chat.update", map[string]string{
		"roomId": roomID,
		"msgId":  messageID,
		"text":   text,
	}, &out)
	return out.Message, err
}

func (c *Client) ThreadMessages(ctx context.Context, threadID string, offset, count int) ([]apiMessage, int, error) {
	var out struct {
		Messages []apiMessage `json:"messages"`
		Total    int          `json:"total"`
	}
	q := url.Values{
		"tmid":   []string{threadID},
		"offset": []string{strconv.Itoa(offset)},
		"count":  []string{strconv.Itoa(count)},
	}
	err := c.get(ctx, "chat.getThreadMessages", q, &out)
	return out.Messages, out.Total, err
}

func (c *Client) SearchMessages(ctx context.Context, roomID, text string, offset, count int) ([]apiMessage, error) {
	var out struct {
		Messages []apiMessage `json:"messages"`
	}
	q := url.Values{
		"roomId":     []string{roomID},
		"searchText": []string{text},
		"offset":     []string{strconv.Itoa(offset)},
		"count":      []string{strconv.Itoa(count)},
	}
	err := c.get(ctx, "chat.search", q, &out)
	return out.Messages, err
}

func (c *Client) DeleteMessage(ctx context.Context, roomID, messageID string) error {
	return c.post(ctx, "chat.delete", map[string]any{
		"roomId": roomID,
		"msgId":  messageID,
	}, &map[string]any{})
}
