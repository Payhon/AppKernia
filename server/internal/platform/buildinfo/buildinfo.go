package buildinfo

// These values are overridden with -ldflags for release builds. Keeping the
// defaults explicit prevents local builds from claiming a release version.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
