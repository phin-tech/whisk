package buildinfo

import (
	"os"
	"runtime/debug"
)

var version = ""
var gitSHA = ""
var dirty = ""

type Info struct {
	Version string
	GitSHA  string
	Dirty   bool
}

func Current() Info {
	info, _ := debug.ReadBuildInfo()
	return Info{
		Version: mainVersion(info),
		GitSHA:  GitSHA(),
		Dirty:   dirty == "true" || setting(info, "vcs.modified", "") == "true",
	}
}

func GitSHA() string {
	if value := os.Getenv("WHISK_GIT_SHA"); value != "" {
		return value
	}
	if gitSHA != "" {
		return gitSHA
	}
	info, _ := debug.ReadBuildInfo()
	if value := setting(info, "vcs.revision", ""); value != "" {
		return value
	}
	return "unknown"
}

func mainVersion(info *debug.BuildInfo) string {
	if version != "" {
		return version
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func setting(info *debug.BuildInfo, key string, fallback string) string {
	if info == nil {
		return fallback
	}
	for _, setting := range info.Settings {
		if setting.Key == key && setting.Value != "" {
			return setting.Value
		}
	}
	return fallback
}
