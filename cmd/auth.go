package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/config"
	"github.com/UsingCoding/rocketchat-cli/internal/output"
	"github.com/UsingCoding/rocketchat-cli/internal/service"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage authentication"}
	cmd.AddCommand(newAuthLoginCmd(o), newAuthLogoutCmd(o), newAuthStatusCmd(o))
	return cmd
}

func newAuthLoginCmd(o *rootOptions) *cobra.Command {
	var urlValue, userID, token string
	cmd := &cobra.Command{Use: "login [context]", Args: cobra.MaximumNArgs(1), Short: "Authenticate and save a context", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		name := cfg.CurrentContext
		if len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			name = "default"
		}
		base := cfg.Contexts[name]
		if urlValue == "" {
			urlValue = base.URL
		}
		if userID == "" {
			userID = base.UserID
		}
		if token == "" {
			token = base.Token
		}
		reader := bufio.NewReader(cmd.InOrStdin())
		if urlValue == "" {
			urlValue, err = prompt(reader, cmd, "Rocket.Chat URL")
			if err != nil {
				return err
			}
		}
		if userID == "" {
			userID, err = prompt(reader, cmd, "User ID")
			if err != nil {
				return err
			}
		}
		if token == "" {
			token, err = promptSecret(reader, cmd, "Token")
			if err != nil {
				return err
			}
		}
		client := api.New(strings.TrimRight(urlValue, "/"), userID, token, o.Timeout)
		svc := service.New(client)
		me, err := svc.Me(cmd.Context())
		if err != nil {
			return classify(err)
		}
		cfg.Contexts[name] = config.Context{URL: strings.TrimRight(urlValue, "/"), UserID: userID, Token: token}
		cfg.CurrentContext = name
		if err := config.Save(o.ConfigPath, cfg); err != nil {
			return err
		}
		if o.JSON {
			return output.JSON(cmd.OutOrStdout(), map[string]any{"context": name, "user": me})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s; current context: %s\n", me.Username, name)
		return nil
	}}
	cmd.Flags().StringVar(&urlValue, "url", "", "Rocket.Chat URL")
	cmd.Flags().StringVar(&userID, "user-id", "", "Rocket.Chat user ID")
	cmd.Flags().StringVar(&token, "token", "", "Rocket.Chat auth token")
	return cmd
}

func newAuthLogoutCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "logout [context]", Args: cobra.MaximumNArgs(1), Short: "Remove saved token from a context", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(o)
		if err != nil {
			return err
		}
		name := cfg.CurrentContext
		if len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			return usageErr("no context selected")
		}
		c, ok := cfg.Contexts[name]
		if !ok {
			return cliNotFoundContext(name)
		}
		c.Token = ""
		cfg.Contexts[name] = c
		return config.Save(o.ConfigPath, cfg)
	}}
}

func newAuthStatusCmd(o *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Verify current credentials", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, client, err := buildService(o)
		if err != nil {
			return err
		}
		me, err := svc.Me(cmd.Context())
		if err != nil {
			return classify(err)
		}
		return render(cmd, o, client, me, func() { fmt.Fprintf(cmd.OutOrStdout(), "authenticated as %s (%s)\n", me.Username, me.ID) })
	}}
}

func prompt(r *bufio.Reader, cmd *cobra.Command, label string) (string, error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", label)
	v, err := r.ReadString('\n')
	return strings.TrimSpace(v), err
}
func promptSecret(r *bufio.Reader, cmd *cobra.Command, label string) (string, error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", label)
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		return strings.TrimSpace(string(b)), err
	}
	v, err := r.ReadString('\n')
	return strings.TrimSpace(v), err
}
