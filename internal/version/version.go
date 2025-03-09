package version

var (
	Version = "1.0.7"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
