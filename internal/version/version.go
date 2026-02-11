package version

var (
	// CommitHash is the git commit hash, set at build time via -ldflags
	CommitHash string = "unknown"
)
