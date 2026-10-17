package skills

import "embed"

// FS contains the distributable Rocket.Chat agent skill.
//
//go:embed rocketchat-cli/**
var FS embed.FS
