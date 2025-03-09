package version

var (
	Version = "1.0.2"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
