package version

import (
	"bytes"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	const fullSHA = "0123456789abcdef0123456789abcdef01234567"

	for _, tc := range []struct {
		name    string
		ver     string
		rev     string
		info    *debug.BuildInfo
		wantVer string
		wantRev string
	}{
		{
			name:    "nil build info keeps unknown",
			ver:     unknown,
			rev:     unknown,
			info:    nil,
			wantVer: unknown,
			wantRev: unknown,
		},
		{
			name:    "empty strings with nil build info normalize to unknown",
			ver:     "",
			rev:     "",
			info:    nil,
			wantVer: unknown,
			wantRev: unknown,
		},
		{
			name: "ldflags take precedence over build info",
			ver:  "v1.2.3",
			rev:  "abc1234",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v9.9.9"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: fullSHA},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVer: "v1.2.3",
			wantRev: "abc1234",
		},
		{
			name: "go install tagged module sets version only",
			ver:  unknown,
			rev:  unknown,
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
			},
			wantVer: "v1.2.3",
			wantRev: unknown,
		},
		{
			name: "local build uses devel and full vcs revision",
			ver:  unknown,
			rev:  unknown,
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: fullSHA},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			wantVer: "(devel)",
			wantRev: fullSHA,
		},
		{
			name: "local dirty build appends dirty suffix to full vcs revision",
			ver:  unknown,
			rev:  unknown,
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: fullSHA},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVer: "(devel)",
			wantRev: fullSHA + "-dirty",
		},
		{
			name: "vcs modified without revision keeps unknown",
			ver:  unknown,
			rev:  unknown,
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.0.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVer: "v1.0.0",
			wantRev: unknown,
		},
		{
			name: "ldflags revision with fallback version",
			ver:  unknown,
			rev:  "deadbee",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: fullSHA},
				},
			},
			wantVer: "(devel)",
			wantRev: "deadbee",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotVer, gotRev := resolve(tc.ver, tc.rev, tc.info)
			assert.Equal(t, tc.wantVer, gotVer)
			assert.Equal(t, tc.wantRev, gotRev)
		})
	}
}

func TestWriteAndGetters(t *testing.T) {
	origVer, origRev := Version, Revision
	t.Cleanup(func() {
		Version, Revision = origVer, origRev
	})

	Version = "v2.0.0"
	Revision = "1234567"

	assert.Equal(t, "v2.0.0", GetVersion())
	assert.Equal(t, "1234567", GetRevision())

	var buf bytes.Buffer
	Write(&buf)
	assert.Equal(t, "Version: v2.0.0\nRevision: 1234567\n", buf.String())
}
