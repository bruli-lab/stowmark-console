package buildinfo

import (
	"runtime/debug"
	"strings"
)

const (
	DevelopmentVersion = "dev"
	unstampedVersion   = "(devel)"
)

type Reader func() (*debug.BuildInfo, bool)

func Version(injected string) string {
	return Resolve(injected, debug.ReadBuildInfo)
}

func Resolve(injected string, read Reader) string {
	if injected != "" {
		return normalize(injected)
	}

	info, ok := read()
	if !ok || info.Main.Version == "" || info.Main.Version == unstampedVersion {
		return DevelopmentVersion
	}

	return normalize(info.Main.Version)
}

func normalize(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}

	return "v" + version
}
