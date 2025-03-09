package version

var (
	Version = "0.0.2"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
