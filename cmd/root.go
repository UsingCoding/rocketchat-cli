package cmd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/UsingCoding/rocketchat-cli/internal/api"
	"github.com/UsingCoding/rocketchat-cli/internal/clierr"
	"github.com/UsingCoding/rocketchat-cli/internal/config"
	"github.com/UsingCoding/rocketchat-cli/internal/service"
	"github.com/spf13/cobra"
)

var version = "dev"

type rootOptions struct {
	ConfigPath string
	Context    string
	URL        string
	UserID     string
	Token      string
	JSON       bool
	Raw        bool
	Timeout    time.Duration
}

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	o := &rootOptions{}
	cmd := &cobra.Command{
		Use:           "rocketchat",
		Short:         "Shell- and agent-friendly Rocket.Chat CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return clierr.New(clierr.CodeUsage, err) })

	f := cmd.PersistentFlags()
	f.StringVar(&o.ConfigPath, "config", config.DefaultPath(), "config file")
	f.StringVar(&o.Context, "context", "", "context to use")
	f.StringVar(&o.URL, "url", "", "Rocket.Chat URL")
	f.StringVar(&o.UserID, "user-id", "", "Rocket.Chat user ID")
	f.StringVar(&o.Token, "token", "", "Rocket.Chat auth token")
	f.BoolVar(&o.JSON, "json", false, "print normalized JSON")
	f.BoolVar(&o.Raw, "raw", false, "print raw Rocket.Chat API response")
	f.DurationVar(&o.Timeout, "timeout", 30*time.Second, "HTTP timeout")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if o.JSON && o.Raw {
			return clierr.New(clierr.CodeUsage, errors.New("--json and --raw are mutually exclusive"))
		}
		return nil
	}

	cmd.AddCommand(
		newContextCmd(o),
		newAuthCmd(o),
		newMeCmd(o),
		newUserCmd(o),
		newChannelCmd(o),
		newMessageCmd(o),
		newSkillCmd(),
	)
	return cmd
}

func loadConfig(o *rootOptions) (*config.Config, error) {
	cfg, err := config.Load(o.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func buildService(o *rootOptions) (*service.Service, *api.Client, error) {
	cfg, err := loadConfig(o)
	if err != nil {
		return nil, nil, err
	}
	r, err := cfg.Resolve(config.Overrides{Context: o.Context, URL: o.URL, UserID: o.UserID, Token: o.Token})
	if err != nil {
		return nil, nil, err
	}
	client := api.New(r.URL, r.UserID, r.Token, o.Timeout)
	return service.New(client), client, nil
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	var ae *api.Error
	if errors.As(err, &ae) {
		switch ae.StatusCode {
		case http.StatusUnauthorized:
			return clierr.New(clierr.CodeAuth, err)
		case http.StatusForbidden:
			return clierr.New(clierr.CodePermission, err)
		case http.StatusConflict:
			return clierr.New(clierr.CodeConflict, err)
		case http.StatusNotFound:
			return clierr.New(clierr.CodeNotFound, err)
		}
		if ae.NotFound() {
			return clierr.New(clierr.CodeNotFound, err)
		}
	}
	var ue *url.Error
	var ne net.Error
	if errors.As(err, &ue) || errors.As(err, &ne) {
		return clierr.New(clierr.CodeNetwork, err)
	}
	return err
}

func usageErr(msg string, args ...any) error {
	return clierr.New(clierr.CodeUsage, fmt.Errorf(msg, args...))
}
