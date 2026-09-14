package main

import (
	"fmt"
	"os"

	"github.com/UsingCoding/rocketchat-cli/cmd"
	"github.com/UsingCoding/rocketchat-cli/internal/clierr"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(clierr.ExitCode(err))
	}
}
