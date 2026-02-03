package cli

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"
)

// Version can be overridden at build time with -ldflags "-X noisyzip/internal/cli.Version=1.2.3".
var Version = "dev"

func printVersion(w io.Writer) {
	fmt.Fprintln(w, "noisyzip", versionString())
}

func versionString() string {
	if v := strings.TrimSpace(Version); v != "" && v != "dev" {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	var rev string
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			rev = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if modified {
		return "dev+" + rev + "-dirty"
	}
	return "dev+" + rev
}
