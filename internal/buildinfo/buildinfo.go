package buildinfo

import (
	"os"
	"runtime/debug"
)

func GitSHA() string {
	if value := os.Getenv("WHISK_GIT_SHA"); value != "" {
		return value
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return setting.Value
		}
	}
	return "unknown"
}
