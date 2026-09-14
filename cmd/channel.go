package cmd

import (
	"time"

	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newChannelCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "channel", Short: "Work with public and private channels"}
	cmd.AddCommand(newChannelGetCmd(o), newChannelListCmd(o), newChannelMembersCmd(o), newChannelHistoryCmd(o))
	return cmd
}

func newChannelGetCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "get <name|id>", Args: cobra.ExactArgs(1), Short: "Get channel information", RunE: func(cmd *cobra.Command, args []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		c, err := svc.Channel(cmd.Context(), args[0])
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, c, func() { output.Channel(cmd.OutOrStdout(), c) })
	}}
}

func newChannelListCmd(o *rootOptions) *cobra.Command {
	var publicOnly, privateOnly, unread bool
	cmd := &cobra.Command{Use: "list", Short: "List joined channels", RunE: func(cmd *cobra.Command, _ []string) error {
		if publicOnly && privateOnly {
			return usageErr("--public and --private are mutually exclusive")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		channels, err := svc.Channels(cmd.Context(), publicOnly, privateOnly, unread)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, channels, func() { output.Channels(cmd.OutOrStdout(), channels) })
	}}
	cmd.Flags().BoolVar(&publicOnly, "public", false, "only public channels")
	cmd.Flags().BoolVar(&privateOnly, "private", false, "only private channels")
	cmd.Flags().BoolVar(&unread, "unread", false, "only channels with unread messages")
	return cmd
}

func newChannelMembersCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var all bool
	cmd := &cobra.Command{Use: "members <channel>", Args: cobra.ExactArgs(1), Short: "List channel members", RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		users, err := svc.ChannelMembers(cmd.Context(), args[0], offset, limit, all)
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, users, func() { output.Users(cmd.OutOrStdout(), users) })
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "items to skip")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func newChannelHistoryCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var since, before string
	cmd := &cobra.Command{Use: "history <channel>", Args: cobra.ExactArgs(1), Short: "Show channel message history", RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		oldest, err := parseMoment(since)
		if err != nil {
			return usageErr("invalid --since: %v", err)
		}
		latest, err := parseMoment(before)
		if err != nil {
			return usageErr("invalid --before: %v", err)
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		messages, err := svc.ChannelHistory(cmd.Context(), args[0], api.HistoryOptions{Offset: offset, Count: limit, Oldest: oldest, Latest: latest})
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, messages, func() { output.Messages(cmd.OutOrStdout(), messages) })
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "number of messages")
	cmd.Flags().IntVar(&offset, "offset", 0, "messages to skip")
	cmd.Flags().StringVar(&since, "since", "", "oldest message time (RFC3339 or duration such as 2h)")
	cmd.Flags().StringVar(&before, "before", "", "latest message time (RFC3339 or duration such as 30m)")
	return cmd
}

func parseMoment(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if d, err := time.ParseDuration(v); err == nil {
		return time.Now().Add(-d).UTC().Format(time.RFC3339Nano), nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return "", err
	}
	return t.UTC().Format(time.RFC3339Nano), nil
}
