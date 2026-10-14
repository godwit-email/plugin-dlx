// Test harness using Godwit's real PR #68 loader, without the UI dependencies.
package main

import (
	"os"

	"github.com/godwit-email/godwit/core/plugins"
)

func main() {
	os.Exit(plugins.ServeNative(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
