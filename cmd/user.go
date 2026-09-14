package cmd

import (
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newUserCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "user", Short: "Read Rocket.Chat users"}
	cmd.AddCommand(newUserGetCmd(o), newUserListCmd(o), newUserSearchCmd(o))
	return cmd
}

func newUserGetCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "get <username|id|email>", Args: cobra.ExactArgs(1), Short: "Get a user", RunE: func(cmd *cobra.Command, args []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		u, err := svc.User(cmd.Context(), args[0])
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, u, func() { output.User(cmd.OutOrStdout(), u) })
	}}
}

func newUserListCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var all bool
	cmd := &cobra.Command{Use: "list", Short: "List users", RunE: func(cmd *cobra.Command, _ []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		users, err := svc.Users(cmd.Context(), offset, limit, all)
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

func newUserSearchCmd(o *rootOptions) *cobra.Command {
	var limit, offset int
	var all bool
	cmd := &cobra.Command{Use: "search <text>", Args: cobra.ExactArgs(1), Short: "Search workspace users", RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 {
			return usageErr("--limit must be greater than zero")
		}
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		users, err := svc.SearchUsers(cmd.Context(), args[0], offset, limit, all)
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
