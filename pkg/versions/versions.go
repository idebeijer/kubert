package versions

// variables overwritten by -ldflags -X at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "local"
)
