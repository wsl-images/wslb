package version

var (
	Version = "1.0.1"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
