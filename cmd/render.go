package cmd

import (
	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func render(cmd *cobra.Command, o *rootOptions, client *api.Client, value any, table func()) error {
	if o.Raw {
		return output.Raw(cmd.OutOrStdout(), client.LastRaw())
	}
	if o.JSON {
		return output.JSON(cmd.OutOrStdout(), value)
	}
	table()
	return nil
}
