package version

var (
	Version = "0.0.3"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
