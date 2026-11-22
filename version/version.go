package version

import (
	"fmt"
	"io"
	"runtime/debug"
)

const unknown = "unknown"

var (
	Version  = unknown
	Revision = unknown
)

func resolve(ver, rev string, info *debug.BuildInfo) (string, string) {
	if ver == "" {
		ver = unknown
	}
	if rev == "" {
		rev = unknown
	}
	if info == nil {
		return ver, rev
	}
	if ver == unknown && info.Main.Version != "" {
		ver = info.Main.Version
	}
	if rev == unknown {
		var (
			vcsRev   string
			modified bool
		)
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				vcsRev = s.Value
			case "vcs.modified":
				modified = s.Value == "true"
			}
		}
		if vcsRev != "" {
			rev = vcsRev
			if modified {
				rev += "-dirty"
			}
		}
	}
	return ver, rev
}

func current() (string, string) {
	info, _ := debug.ReadBuildInfo()
	return resolve(Version, Revision, info)
}

func GetVersion() string {
	v, _ := current()
	return v
}

func GetRevision() string {
	_, r := current()
	return r
}

func Write(w io.Writer) {
	v, r := current()
	_, _ = fmt.Fprintf(w, "Version: %s\n", v)
	_, _ = fmt.Fprintf(w, "Revision: %s\n", r)
}
