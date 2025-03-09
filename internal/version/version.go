package version

var (
	Version = "1.0.0"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
