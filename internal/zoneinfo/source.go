package zoneinfo

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
)

var systemPaths = []string{
	"/usr/share/zoneinfo",
	"/usr/share/lib/zoneinfo",
	"/usr/lib/locale/TZ",
	"/etc/zoneinfo",
}

func Source() string {
	if path := os.Getenv("ZONEINFO"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return "$ZONEINFO=" + path
		}
	}
	for _, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	version := runtime.Version()
	if info, ok := debug.ReadBuildInfo(); ok && info.GoVersion != "" {
		version = info.GoVersion
	}
	return fmt.Sprintf("embedded (%s)", version)
}
