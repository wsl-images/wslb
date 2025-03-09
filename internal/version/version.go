package version

var (
	Version = "0.0.4"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
