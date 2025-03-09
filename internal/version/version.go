package version

var (
	Version = "0.0.1"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
