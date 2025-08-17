package version

var version string

func Version() string {
	if version == "" {
		version = "dev"
	}

	return version
}
