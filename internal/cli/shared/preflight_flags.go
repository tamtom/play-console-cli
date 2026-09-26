package shared

// Usage text for the target SDK policy flags. The preflight, validate and
// publish track commands use the same text so that the help stays aligned.
const (
	PreflightAppTypeUsage      = "Submission policy: mobile, wear, automotive, tv, xr, private (permanently private organization apps) (default: detected from the manifest, else mobile)"
	PreflightMinTargetSDKUsage = "Minimum accepted targetSdkVersion (default: the current Play requirement)"
)
