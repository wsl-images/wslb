package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var Version = "dev"

func GetVersionInfo() string {
	version := Version

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		}

		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				version = fmt.Sprintf("%s (commit: %s)", version, setting.Value[:7])
				break
			}
		}
	}

	return "WSLB version: " + version
}
