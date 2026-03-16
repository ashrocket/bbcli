// Package main is the entry point for the bbcli binary.
package main

import (
	"os"

	"github.com/ashrocket/bbcli/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
