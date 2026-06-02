package version

import "time"

// Number is the semantic dashboard version (bump on releases).
var Number = "2.00"

// Build is set via -ldflags; otherwise stamped at process start for cache busting.
var Build string

func init() {
	if Build == "" {
		Build = time.Now().UTC().Format("20060102.150405")
	}
}

// String returns the full version label (e.g. 1.90+20260530.190625).
func String() string {
	return Number + "+" + Build
}

// AssetTag is appended to static asset URLs to avoid stale browser cache.
func AssetTag() string {
	return Build
}
