package cmd

import (
	"fmt"
	"sort"

	"github.com/UsingCoding/rocketchat-cli/internal/config"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/spf13/cobra"
)

type contextView struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	URL      string `json:"url"`
	UserID   string `json:"user_id,omitempty"`
	HasToken bool   `json:"has_token"`
}

func newContextCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "context", Short: "Manage Rocket.Chat contexts"}
	cmd.AddCommand(
		newContextListCmd(o), newContextCurrentCmd(o), newContextGetCmd(o),
		newContextSetCmd(o), newContextUseCmd(o), newContextDeleteCmd(o),
	)
	return cmd
}

func newContextListCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List contexts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(o)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Contexts))
			for n := range cfg.Contexts {
				names = append(names, n)
			}
			sort.Strings(names)
			views := make([]contextView, 0, len(names))
			for _, n := range names {
				c := cfg.Contexts[n]
				views = append(views, contextView{Name: n, Current: n == cfg.CurrentContext, URL: c.URL, UserID: c.UserID, HasToken: c.Token != ""})
			}
			if o.JSON {
				return output.JSON(cmd.OutOrStdout(), views)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "CURRENT\tNAME\tURL\tUSER ID\tTOKEN")
			for _, v := range views {
				cur := ""
				if v.Current {
					cur = "*"
				}
				tok := "-"
				if v.HasToken {
					tok = "set"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", cur, v.Name, v.URL, v.UserID, tok)
			}
			return nil
		},
	}
}

func newContextCurrentCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "current", Short: "Print current context", RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		if cfg.CurrentContext == "" {
			return usageErr("no current context configured")
		}
		if o.JSON {
			return output.JSON(cmd.OutOrStdout(), map[string]string{"current_context": cfg.CurrentContext})
		}
		fmt.Fprintln(cmd.OutOrStdout(), cfg.CurrentContext)
		return nil
	}}
}

func newContextGetCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "get <name>", Args: cobra.ExactArgs(1), Short: "Show a context", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		c, ok := cfg.Contexts[args[0]]
		if !ok {
			return cliNotFoundContext(args[0])
		}
		v := contextView{Name: args[0], Current: args[0] == cfg.CurrentContext, URL: c.URL, UserID: c.UserID, HasToken: c.Token != ""}
		if o.JSON {
			return output.JSON(cmd.OutOrStdout(), v)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\nCurrent:   %t\nURL:       %s\nUser ID:   %s\nToken:     %s\n", v.Name, v.Current, v.URL, v.UserID, map[bool]string{true: "set", false: "-"}[v.HasToken])
		return nil
	}}
}

func newContextSetCmd(o *rootOptions) *cobra.Command {
	var urlValue, userID, token string
	cmd := &cobra.Command{Use: "set <name>", Args: cobra.ExactArgs(1), Short: "Create or update a context", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		c := cfg.Contexts[args[0]]
		if cmd.Flags().Changed("url") {
			c.URL = urlValue
		}
		if cmd.Flags().Changed("user-id") {
			c.UserID = userID
		}
		if cmd.Flags().Changed("token") {
			c.Token = token
		}
		if c.URL == "" {
			return usageErr("--url is required for a new context")
		}
		cfg.Contexts[args[0]] = c
		if cfg.CurrentContext == "" {
			cfg.CurrentContext = args[0]
		}
		if err := config.Save(o.ConfigPath, cfg); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), args[0])
		return nil
	}}
	cmd.Flags().StringVar(&urlValue, "url", "", "Rocket.Chat URL")
	cmd.Flags().StringVar(&userID, "user-id", "", "Rocket.Chat user ID")
	cmd.Flags().StringVar(&token, "token", "", "Rocket.Chat auth token")
	return cmd
}

func newContextUseCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "use <name>", Args: cobra.ExactArgs(1), Short: "Set current context", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		if _, ok := cfg.Contexts[args[0]]; !ok {
			return cliNotFoundContext(args[0])
		}
		cfg.CurrentContext = args[0]
		if err := config.Save(o.ConfigPath, cfg); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), args[0])
		return nil
	}}
}

func newContextDeleteCmd(o *rootOptions) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <name>", Args: cobra.ExactArgs(1), Short: "Delete a context", RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return usageErr("context delete requires --yes")
		}
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		if _, ok := cfg.Contexts[args[0]]; !ok {
			return cliNotFoundContext(args[0])
		}
		delete(cfg.Contexts, args[0])
		if cfg.CurrentContext == args[0] {
			cfg.CurrentContext = ""
		}
		return config.Save(o.ConfigPath, cfg)
	}}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm deletion")
	return cmd
}

func cliNotFoundContext(name string) error { return fmt.Errorf("context %q not found", name) }
