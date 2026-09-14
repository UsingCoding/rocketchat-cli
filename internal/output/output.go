package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/UsingCoding/rocketchat-cli/internal/model"
)

func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func Raw(w io.Writer, b []byte) error {
	if len(b) == 0 {
		_, err := fmt.Fprintln(w, "{}")
		return err
	}
	var dst bytes.Buffer
	if json.Indent(&dst, b, "", "  ") == nil {
		_, err := fmt.Fprintln(w, dst.String())
		return err
	}
	_, err := w.Write(append(b, '\n'))
	return err
}

func Users(w io.Writer, users []model.User) {
	t := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(t, "ID\tUSERNAME\tNAME\tSTATUS\tEMAIL")
	for _, u := range users {
		fmt.Fprintf(t, "%s\t%s\t%s\t%s\t%s\n", u.ID, u.Username, u.Name, u.Status, u.Email)
	}
	_ = t.Flush()
}

func User(w io.Writer, u model.User) {
	fmt.Fprintf(w, "ID:          %s\n", u.ID)
	fmt.Fprintf(w, "Username:    %s\n", u.Username)
	fmt.Fprintf(w, "Name:        %s\n", u.Name)
	if u.Email != "" {
		fmt.Fprintf(w, "Email:       %s\n", u.Email)
	}
	fmt.Fprintf(w, "Status:      %s\n", u.Status)
	fmt.Fprintf(w, "Active:      %t\n", u.Active)
	if len(u.Roles) > 0 {
		fmt.Fprintf(w, "Roles:       %s\n", strings.Join(u.Roles, ", "))
	}
	if !u.LastLogin.IsZero() {
		fmt.Fprintf(w, "Last login:  %s\n", u.LastLogin.Format(time.RFC3339))
	}
}

func Channels(w io.Writer, channels []model.Channel) {
	t := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(t, "ID\tNAME\tTYPE\tMEMBERS\tMESSAGES\tUNREAD")
	for _, c := range channels {
		fmt.Fprintf(t, "%s\t%s\t%s\t%d\t%d\t%d\n", c.ID, c.Name, c.Type, c.MemberCount, c.MessageCount, c.Unread)
	}
	_ = t.Flush()
}

func Channel(w io.Writer, c model.Channel) {
	fmt.Fprintf(w, "ID:        %s\n", c.ID)
	fmt.Fprintf(w, "Name:      %s\n", c.Name)
	fmt.Fprintf(w, "Type:      %s\n", c.Type)
	if c.Topic != "" {
		fmt.Fprintf(w, "Topic:     %s\n", c.Topic)
	}
	fmt.Fprintf(w, "Members:   %d\n", c.MemberCount)
	fmt.Fprintf(w, "Messages:  %d\n", c.MessageCount)
	fmt.Fprintf(w, "Unread:    %d\n", c.Unread)
}

func Messages(w io.Writer, messages []model.Message) {
	t := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(t, "ID\tTIME\tUSER\tMESSAGE")
	for _, m := range messages {
		fmt.Fprintf(t, "%s\t%s\t%s\t%s\n", m.ID, formatTime(m.CreatedAt), m.Author.Username, oneLine(m.Text))
	}
	_ = t.Flush()
}

func Message(w io.Writer, m model.Message) {
	fmt.Fprintf(w, "ID:       %s\n", m.ID)
	fmt.Fprintf(w, "Room:     %s\n", m.RoomID)
	if m.ThreadID != "" {
		fmt.Fprintf(w, "Thread:   %s\n", m.ThreadID)
	}
	fmt.Fprintf(w, "Author:   %s\n", m.Author.Username)
	if !m.CreatedAt.IsZero() {
		fmt.Fprintf(w, "Time:     %s\n", m.CreatedAt.Format(time.RFC3339))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, m.Text)
}

func Thread(w io.Writer, thread model.Thread) {
	fmt.Fprintf(w, "%s  %s  %s\n", thread.Root.ID, thread.Root.Author.Username, oneLine(thread.Root.Text))
	fmt.Fprintln(w, "\nTHREAD")
	Messages(w, thread.Replies)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ↵ ")
	if len([]rune(s)) > 120 {
		r := []rune(s)
		return string(r[:117]) + "..."
	}
	return s
}
