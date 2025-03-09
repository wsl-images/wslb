package version

var (
	Version = "1.0.3"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
