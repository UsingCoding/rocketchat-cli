package cmd

import (
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMeCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "me", Short: "Show authenticated user", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		u, err := svc.Me(cmd.Context())
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, u, func() { output.User(cmd.OutOrStdout(), u) })
	}}
}
