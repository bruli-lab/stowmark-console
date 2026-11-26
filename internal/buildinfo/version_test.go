package buildinfo_test

import (
	"runtime/debug"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/buildinfo"
	"github.com/stretchr/testify/require"
)

func stamped(version string) buildinfo.Reader {
	return func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: version}}, true
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		injected string
		read     buildinfo.Reader
		expected string
	}{
		{name: "a version injected by the release build, then it wins over the build info", injected: "1.2.3", read: stamped("v0.0.1"), expected: "v1.2.3"},
		{name: "an injected version with prefix, then it is kept", injected: "v1.2.3", read: stamped(""), expected: "v1.2.3"},
		{name: "no injected version and a version stamped from git, then the stamped one is used", read: stamped("v1.0.3-0.20261126222033-0a91d20a48c6+dirty"), expected: "v1.0.3-0.20261126222033-0a91d20a48c6+dirty"},
		{name: "a build without version control information, then it is a development version", read: stamped("(devel)"), expected: buildinfo.DevelopmentVersion},
		{name: "no build info at all, then it is a development version", read: func() (*debug.BuildInfo, bool) { return nil, false }, expected: buildinfo.DevelopmentVersion},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, buildinfo.Resolve(tt.injected, tt.read))
		})
	}
}
