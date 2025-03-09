package version

var (
	Version = "1.0.6"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
