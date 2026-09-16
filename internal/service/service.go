package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/model"
)

type Service struct {
	API *api.Client
}

func New(client *api.Client) *Service { return &Service{API: client} }

func (s *Service) Me(ctx context.Context) (model.User, error) {
	u, err := s.API.Me(ctx)
	return toUser(u), err
}

func (s *Service) User(ctx context.Context, identifier string) (model.User, error) {
	var u api.UserDTO
	var err error
	if strings.Contains(identifier, "@") {
		u, err = s.API.UserInfo(ctx, "email", identifier)
	} else {
		u, err = s.API.UserInfo(ctx, "username", identifier)
		if err != nil && api.IsNotFound(err) {
			u, err = s.API.UserInfo(ctx, "userId", identifier)
		}
	}
	return toUser(u), err
}

func (s *Service) Users(ctx context.Context, offset, limit int, all bool) ([]model.User, error) {
	if !all {
		users, _, err := s.API.Users(ctx, offset, limit)
		return mapUsers(users), err
	}
	var result []model.User
	for {
		users, total, err := s.API.Users(ctx, offset, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, mapUsers(users)...)
		offset += len(users)
		if len(users) == 0 || offset >= total {
			return result, nil
		}
	}
}

func (s *Service) SearchUsers(ctx context.Context, text string, offset, limit int, all bool) ([]model.User, error) {
	if !all {
		users, _, err := s.API.SearchUsers(ctx, text, offset, limit)
		return mapUsers(users), err
	}
	var result []model.User
	for {
		users, total, err := s.API.SearchUsers(ctx, text, offset, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, mapUsers(users)...)
		offset += len(users)
		if len(users) == 0 || offset >= total {
			return result, nil
		}
	}
}

func (s *Service) ResolveRoom(ctx context.Context, identifier string) (api.RoomDTO, error) {
	r, err := s.API.RoomInfo(ctx, "roomName", identifier)
	if err != nil && api.IsNotFound(err) {
		r, err = s.API.RoomInfo(ctx, "roomId", identifier)
	}
	if err != nil {
		return api.RoomDTO{}, err
	}
	if r.Type != "c" && r.Type != "p" {
		return api.RoomDTO{}, fmt.Errorf("room %q is not a public/private channel", identifier)
	}
	return r, nil
}

func (s *Service) Channel(ctx context.Context, identifier string) (model.Channel, error) {
	r, err := s.ResolveRoom(ctx, identifier)
	return toChannel(r), err
}

func (s *Service) Channels(ctx context.Context, wantPublic, wantPrivate, unreadOnly bool) ([]model.Channel, error) {
	rooms, err := s.API.Rooms(ctx)
	if err != nil {
		return nil, err
	}
	if !wantPublic && !wantPrivate {
		wantPublic, wantPrivate = true, true
	}
	out := make([]model.Channel, 0, len(rooms))
	for _, r := range rooms {
		if r.Type == "c" && !wantPublic || r.Type == "p" && !wantPrivate || (r.Type != "c" && r.Type != "p") {
			continue
		}
		if unreadOnly && r.Unread == 0 {
			continue
		}
		out = append(out, toChannel(r))
	}
	return out, nil
}

func (s *Service) ChannelMembers(ctx context.Context, identifier string, offset, limit int, all bool) ([]model.User, error) {
	r, err := s.ResolveRoom(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if !all {
		users, _, err := s.API.Members(ctx, r.Type, r.ID, offset, limit)
		return mapUsers(users), err
	}
	var result []model.User
	for {
		users, total, err := s.API.Members(ctx, r.Type, r.ID, offset, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, mapUsers(users)...)
		offset += len(users)
		if len(users) == 0 || offset >= total {
			return result, nil
		}
	}
}

func (s *Service) ChannelHistory(ctx context.Context, identifier string, o api.HistoryOptions) ([]model.Message, error) {
	r, err := s.ResolveRoom(ctx, identifier)
	if err != nil {
		return nil, err
	}
	messages, _, err := s.API.History(ctx, r.Type, r.ID, o)
	return mapMessages(messages), err
}

func (s *Service) Message(ctx context.Context, id string) (model.Message, error) {
	m, err := s.API.Message(ctx, id)
	return toMessage(m), err
}

func (s *Service) Send(ctx context.Context, target, text string) (model.Message, error) {
	if strings.HasPrefix(target, ":") {
		r, err := s.ResolveRoom(ctx, strings.TrimPrefix(target, ":"))
		if err != nil {
			return model.Message{}, err
		}
		m, err := s.API.SendMessage(ctx, r.ID, text, "", false)
		return toMessage(m), err
	}

	u, err := s.User(ctx, target)
	if err != nil {
		if api.IsNotFound(err) {
			return model.Message{}, fmt.Errorf("%q was not found as a user. If this is a channel, prefix it with ':', for example: rocketchat message send :<channel> <text>: %w", target, err)
		}
		return model.Message{}, err
	}
	roomID, err := s.API.CreateDM(ctx, u.Username)
	if err != nil {
		return model.Message{}, err
	}
	m, err := s.API.SendMessage(ctx, roomID, text, "", false)
	return toMessage(m), err
}

func (s *Service) Edit(ctx context.Context, messageID, text string) (model.Message, error) {
	target, err := s.API.Message(ctx, messageID)
	if err != nil {
		return model.Message{}, err
	}
	m, err := s.API.UpdateMessage(ctx, target.RoomID, target.ID, text)
	return toMessage(m), err
}

func (s *Service) Reply(ctx context.Context, messageID, text string, alsoSend bool) (model.Message, error) {
	target, err := s.API.Message(ctx, messageID)
	if err != nil {
		return model.Message{}, err
	}
	threadID := target.ID
	if target.ThreadID != "" {
		threadID = target.ThreadID
	}
	m, err := s.API.SendMessage(ctx, target.RoomID, text, threadID, alsoSend)
	return toMessage(m), err
}

func (s *Service) Thread(ctx context.Context, messageID string, offset, limit int, all bool) (model.Thread, error) {
	target, err := s.API.Message(ctx, messageID)
	if err != nil {
		return model.Thread{}, err
	}
	rootID := target.ID
	if target.ThreadID != "" {
		rootID = target.ThreadID
	}
	root := target
	if target.ID != rootID {
		root, err = s.API.Message(ctx, rootID)
		if err != nil {
			return model.Thread{}, err
		}
	}

	var replies []model.Message
	for {
		batch, total, err := s.API.ThreadMessages(ctx, rootID, offset, limit)
		if err != nil {
			return model.Thread{}, err
		}
		for _, m := range batch {
			if m.ID != rootID {
				replies = append(replies, toMessage(m))
			}
		}
		if !all {
			break
		}
		offset += len(batch)
		if len(batch) == 0 || offset >= total {
			break
		}
	}
	return model.Thread{Root: toMessage(root), Replies: replies}, nil
}

func (s *Service) SearchMessages(ctx context.Context, channel, text string, offset, limit int) ([]model.Message, error) {
	r, err := s.ResolveRoom(ctx, channel)
	if err != nil {
		return nil, err
	}
	messages, err := s.API.SearchMessages(ctx, r.ID, text, offset, limit)
	return mapMessages(messages), err
}

func (s *Service) DeleteMessage(ctx context.Context, messageID string) error {
	m, err := s.API.Message(ctx, messageID)
	if err != nil {
		return err
	}
	return s.API.DeleteMessage(ctx, m.RoomID, m.ID)
}

func toUser(u api.UserDTO) model.User {
	email := ""
	if len(u.Emails) > 0 {
		email = u.Emails[0].Address
	}
	return model.User{
		ID: u.ID, Username: u.Username, Name: u.Name, Email: email,
		Status: u.Status, Active: u.Active, Roles: u.Roles, LastLogin: u.LastLogin,
	}
}

func mapUsers(users []api.UserDTO) []model.User {
	out := make([]model.User, 0, len(users))
	for _, u := range users {
		out = append(out, toUser(u))
	}
	return out
}

func toChannel(r api.RoomDTO) model.Channel {
	t := r.Type
	if t == "c" {
		t = "public"
	}
	if t == "p" {
		t = "private"
	}
	name := r.Name
	if name == "" {
		name = r.FName
	}
	return model.Channel{ID: r.ID, Name: name, Type: t, Topic: r.Topic, MessageCount: r.Messages, MemberCount: r.UsersCount, Unread: r.Unread, Archived: r.Archived}
}

func toMessage(m api.MessageDTO) model.Message {
	return model.Message{
		ID: m.ID, RoomID: m.RoomID, ThreadID: m.ThreadID, Text: m.Text, CreatedAt: m.TS,
		Author: model.Author{ID: m.User.ID, Username: m.User.Username, Name: m.User.Name},
	}
}

func mapMessages(messages []api.MessageDTO) []model.Message {
	out := make([]model.Message, 0, len(messages))
	for _, m := range messages {
		out = append(out, toMessage(m))
	}
	return out
}
