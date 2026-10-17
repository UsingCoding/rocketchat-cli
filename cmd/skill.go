package cmd

import (
	"fmt"

	"github.com/UsingCoding/rocketchat-cli/internal/agentskill"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSkillCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "skill", Short: "Manage the embedded coding-agent skill"}
	var project bool
	list := &cobra.Command{
		Use: "list", Short: "List skill installation targets", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			statuses, err := agentskill.List(project)
			if err != nil {
				return err
			}
			if o.JSON {
				return output.JSON(cmd.OutOrStdout(), statuses)
			}
			for _, status := range statuses {
				state := "not installed"
				if status.Exists {
					state = "installed"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", status.Target, state, status.Path); err != nil {
					return err
				}
			}
			return nil
		},
	}
	list.Flags().BoolVar(&project, "project", false, "use agent directories under the current working directory")
	cmd.AddCommand(list,
		newSkillActionCmd("install", "Installed", agentskill.Install),
		newSkillActionCmd("update", "Updated", agentskill.Install),
		newSkillActionCmd("remove", "Removed", agentskill.Remove),
	)
	return cmd
}

func newSkillActionCmd(verb, action string, run func(bool, string) ([]string, error)) *cobra.Command {
	var project bool
	var target string
	cmd := &cobra.Command{
		Use: verb, Short: action + " skill bundle in agent directories", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := run(project, target)
			if err != nil {
				return err
			}
			for _, path := range paths {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s rocketchat-cli skill: %s\n", action, path); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "use agent directories under the current working directory")
	cmd.Flags().StringVar(&target, "target", "", "select claude|codex|cursor|opencode|agents")
	return cmd
}
