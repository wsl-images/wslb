package version

var (
	Version = "0.0.7"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
