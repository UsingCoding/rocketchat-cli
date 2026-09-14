package cmd

import (
	"fmt"

	"github.com/UsingCoding/rocketchat-cli/internal/agentskill"
	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the embedded coding-agent skill",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), agentskill.Text)
			return err
		},
	}
}
