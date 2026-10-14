package cmd

import (
	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newDMCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "dm", Short: "Browse one-to-one direct messages"}
	cmd.AddCommand(newDMListCmd(o), newDMGetCmd(o), newDMHistoryCmd(o))
	return cmd
}

func newDMListCmd(o *rootOptions) *cobra.Command {
	var all bool
	cmd := &cobra.Command{Use: "list", Short: "List joined direct messages", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		dms, err := svc.DirectMessages(cmd.Context(), all)
		if err != nil {
			return renderDMError(cmd, o, client, err)
		}
		return render(cmd, o, client, dms, func() { output.DirectMessages(cmd.OutOrStdout(), dms) })
	}}
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func newDMGetCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "get <user|room-id>", Args: cobra.ExactArgs(1), Short: "Get direct message information", RunE: func(cmd *cobra.Command, args []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		dm, err := svc.DirectMessage(cmd.Context(), args[0])
		if err != nil {
			return renderDMError(cmd, o, client, err)
		}
		return render(cmd, o, client, dm, func() { output.DirectMessage(cmd.OutOrStdout(), dm) })
	}}
}

func newDMHistoryCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var since, before string
	cmd := &cobra.Command{Use: "history <user|room-id>", Args: cobra.ExactArgs(1), Short: "Show direct message history", RunE: func(cmd *cobra.Command, args []string) error {
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
		messages, err := svc.DirectMessageHistory(cmd.Context(), args[0], api.HistoryOptions{Offset: offset, Count: limit, Oldest: oldest, Latest: latest})
		if err != nil {
			return renderDMError(cmd, o, client, err)
		}
		return render(cmd, o, client, messages, func() { output.Messages(cmd.OutOrStdout(), messages) })
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "number of messages")
	cmd.Flags().IntVar(&offset, "offset", 0, "messages to skip")
	cmd.Flags().StringVar(&since, "since", "", "oldest message time (RFC3339 or duration such as 2h)")
	cmd.Flags().StringVar(&before, "before", "", "latest message time (RFC3339 or duration such as 30m)")
	return cmd
}

func renderDMError(cmd *cobra.Command, o *rootOptions, client *api.Client, serviceErr error) error {
	if o.Raw {
		if err := output.Raw(cmd.OutOrStdout(), client.LastRaw()); err != nil {
			return err
		}
	}
	return classify(serviceErr)
}
