package version

var (
	Version = "1.0.5"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
