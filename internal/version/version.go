package version

var (
	Version = "1.0.9"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
