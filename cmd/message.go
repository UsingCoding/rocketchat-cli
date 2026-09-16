package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMessageCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "message", Short: "Read and send messages"}
	cmd.AddCommand(newMessageGetCmd(o), newMessageSendCmd(o), newMessageEditCmd(o), newMessageReplyCmd(o), newMessageThreadCmd(o), newMessageSearchCmd(o), newMessageDeleteCmd(o))
	return cmd
}

func newMessageGetCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "get <message-id>", Args: cobra.ExactArgs(1), Short: "Get a message", RunE: func(cmd *cobra.Command, args []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		m, err := svc.Message(cmd.Context(), args[0])
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, m, func() { output.Message(cmd.OutOrStdout(), m) })
	}}
}

func newMessageSendCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "send <target> <text|->", Args: cobra.ExactArgs(2), Short: "Send a message", RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == ":" {
			return usageErr("channel target after ':' is empty")
		}
		text, err := readText(cmd, args[1])
		if err != nil {
			return err
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		m, err := svc.Send(cmd.Context(), args[0], text)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, m, func() { fmt.Fprintf(cmd.OutOrStdout(), "Message sent: %s\n", m.ID) })
	}}
	return cmd
}

func newMessageEditCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "edit <message-id> <text|->", Args: cobra.ExactArgs(2), Short: "Edit a message", RunE: func(cmd *cobra.Command, args []string) error {
		text, err := readText(cmd, args[1])
		if err != nil {
			return err
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		m, err := svc.Edit(cmd.Context(), args[0], text)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, m, func() { fmt.Fprintf(cmd.OutOrStdout(), "Message updated: %s\n", m.ID) })
	}}
	return cmd
}

func newMessageReplyCmd(o *rootOptions) *cobra.Command {
	var alsoSend bool
	cmd := &cobra.Command{Use: "reply <message-id> <text|->", Args: cobra.ExactArgs(2), Short: "Reply in the message thread", RunE: func(cmd *cobra.Command, args []string) error {
		text, err := readText(cmd, args[1])
		if err != nil {
			return err
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		m, err := svc.Reply(cmd.Context(), args[0], text, alsoSend)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, m, func() { fmt.Fprintf(cmd.OutOrStdout(), "Reply sent: %s\n", m.ID) })
	}}
	cmd.Flags().BoolVar(&alsoSend, "also-send", false, "also show the reply in the main channel")
	return cmd
}

func newMessageThreadCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var all bool
	cmd := &cobra.Command{Use: "thread <message-id>", Args: cobra.ExactArgs(1), Short: "Show a complete message thread", RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		thread, err := svc.Thread(cmd.Context(), args[0], offset, limit, all)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, thread, func() { output.Thread(cmd.OutOrStdout(), thread) })
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "replies to skip")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all thread replies")
	return cmd
}

func newMessageSearchCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	cmd := &cobra.Command{Use: "search <channel> <text>", Args: cobra.ExactArgs(2), Short: "Search messages in a channel", RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		messages, err := svc.SearchMessages(cmd.Context(), args[0], args[1], offset, limit)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, messages, func() { output.Messages(cmd.OutOrStdout(), messages) })
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "results to skip")
	return cmd
}

func newMessageDeleteCmd(o *rootOptions) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <message-id>", Args: cobra.ExactArgs(1), Short: "Delete a message", RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			ok, err := confirm(cmd, fmt.Sprintf("Delete message %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		if err := svc.DeleteMessage(cmd.Context(), args[0]); err != nil {
			return classify(err)
		}
		return render(cmd, o, client, map[string]any{"deleted": args[0]}, func() { fmt.Fprintf(cmd.OutOrStdout(), "Deleted: %s\n", args[0]) })
	}}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}

func readText(cmd *cobra.Command, arg string) (string, error) {
	if arg != "-" {
		return arg, nil
	}
	b, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	text := strings.TrimRight(string(b), "\r\n")
	if text == "" {
		return "", usageErr("message text from stdin is empty")
	}
	return text, nil
}
func confirm(cmd *cobra.Command, label string) (bool, error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N] ", label)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}
