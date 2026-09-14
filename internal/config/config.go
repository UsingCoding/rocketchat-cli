package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Context struct {
	URL    string `toml:"url" json:"url"`
	UserID string `toml:"user_id" json:"user_id"`
	Token  string `toml:"token" json:"-"`
}

type Config struct {
	CurrentContext string             `toml:"current_context" json:"current_context"`
	Contexts       map[string]Context `toml:"contexts" json:"contexts"`
}

type Overrides struct {
	Context string
	URL     string
	UserID  string
	Token   string
}

type Resolved struct {
	Name   string
	URL    string
	UserID string
	Token  string
}

func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "rocketchat-cli", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "rocketchat-cli", "config.toml")
	}
	return filepath.Join(home, ".config", "rocketchat-cli", "config.toml")
}

func Load(path string) (*Config, error) {
	cfg := &Config{Contexts: map[string]Context{}}
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]Context{}
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return f.Chmod(0o600)
}

func (c *Config) Resolve(o Overrides) (Resolved, error) {
	name := firstNonEmpty(o.Context, os.Getenv("ROCKETCHAT_CONTEXT"), c.CurrentContext)
	var base Context
	if name != "" {
		var ok bool
		base, ok = c.Contexts[name]
		if !ok && o.URL == "" && os.Getenv("ROCKETCHAT_URL") == "" {
			return Resolved{}, fmt.Errorf("context %q not found", name)
		}
	}

	resolved := Resolved{
		Name:   name,
		URL:    firstNonEmpty(o.URL, os.Getenv("ROCKETCHAT_URL"), base.URL),
		UserID: firstNonEmpty(o.UserID, os.Getenv("ROCKETCHAT_USER_ID"), base.UserID),
		Token:  firstNonEmpty(o.Token, os.Getenv("ROCKETCHAT_TOKEN"), base.Token),
	}
	resolved.URL = strings.TrimRight(resolved.URL, "/")

	if resolved.URL == "" {
		return Resolved{}, errors.New("Rocket.Chat URL is not configured")
	}
	if resolved.UserID == "" {
		return Resolved{}, errors.New("Rocket.Chat user ID is not configured")
	}
	if resolved.Token == "" {
		return Resolved{}, errors.New("Rocket.Chat token is not configured")
	}
	return resolved, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
